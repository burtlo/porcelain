## Plan: Local inference services (replace Ollama)

| Field | Value |
|-------|-------|
| **Doc kind** | `feature-plan` |
| **Owners / areas** | Supervisor, broker (Bifrost), gateway, indexer |
| **Status** | `active`: M0–M3 done; M4+ open |
| **Targets** | v0.5 |
| **Supersedes / superseded by** | Answers Phase 2 of [`internal-embedding-provider-exploration.md`](internal-embedding-provider-exploration.md) |

## At a glance

Replace Ollama with a **small, fixed set of hand-picked models**, each served by its **own `llama-server` process** (llama.cpp), launched and health-checked by **chimera-supervisor**, registered in **Bifrost as one keyless OpenAI-compatible custom provider per model**, and reached by Locus through the gateway exactly like any cloud model. Runs **natively on Windows with CUDA** — no Docker, no WSL2.

```
Locus ──► chimera-gateway ──► chimera-broker (Bifrost) ──┬─► local-qwen9b   llama-server :8180  (Qwen3.5-9B Q4_K_M + mmproj)
                                                          ├─► local-qwen35b  llama-server :8181  (Qwen3.5-35B-A3B IQ3_XXS, optional)
chimera-indexer ─► gateway /v1/embeddings ───────────────┘─► local-embed    llama-server :8182  (nomic-embed-text-v1.5 Q8_0)
          ▲ all started / stopped / health-gated by chimera-supervisor (one child per model)
```

## What the earlier project did (`C:\Users\lynnv\src\assistants`)

- Engine: **llama.cpp `llama-server`**, built from source in Docker (`docker/llmservice-llamacpp/Dockerfile.cuda`, CUDA 12.4, Ubuntu 22.04, arch `89;90`), one container per model, plus a FastAPI gateway on `:8888` merging `/v1/models`.
- Chat 9B: `Qwen3.5-9B-Q4_K_M.gguf` + `mmproj-Qwen3.5-9B-BF16.gguf`, `-ngl 99`, `-c 102915`, `--parallel 2`, `--reasoning-budget 0 --flash-attn on --no-mmap --perf --log-timestamps`.
- Chat 35B overlay: `Qwen3.5-35B-A3B-UD-IQ3_XXS.gguf` + `mmproj-F32.gguf`, `-ngl 29`, `-c 262144`, `--parallel 4`, `-t 8 -b 512 -ub 512 --flash-attn on --kv-unified` (mirrors LM Studio).
- Embed: `nomic-embed-text-v1.5.Q8_0.gguf`, `-c 2048 --batch-size 2048 --ubatch-size 2048`.
- Weights (~21 GB) already live in `assistants/models/{chat,embed}` and are reusable as-is.

So the shape we want — one server per model behind a router — was already proven. Porcelain already has the router (gateway + Bifrost) and the process manager (supervisor); what's missing is the model processes.

## Engine recommendation: llama.cpp `llama-server`

| Option | Verdict |
|--------|---------|
| **llama.cpp `llama-server`** | **Pick.** You already own the GGUFs and tuned flags. Native Windows CUDA builds exist. One process per model, small VRAM footprint you control. Serves OpenAI `/v1/chat/completions`, `/v1/embeddings`, `/v1/models`, `/health`. Handles Qwen tool calls (`--jinja`), reasoning budget, mmproj vision. |
| vLLM | Linux only (WSL2 at best). Grabs a fixed fraction of VRAM per process (`gpu_memory_utilization`), so two independent models on 16 GB fight each other. Wants safetensors/AWQ, not your GGUFs. Its strength is high-concurrency serving, which a single user doesn't need. |
| SGLang | Same Linux/VRAM-allocation problems as vLLM. |
| LM Studio / Ollama | What we're leaving: a catalog app that owns model lifecycle, not porcelain. |

## Where it runs: native Windows, not Docker or WSL2

**About the earlier advice that Linux is needed to compile and build tooling:** that's true for vLLM/SGLang, and it was true of the old setup because the Dockerfile compiled llama.cpp inside a Linux image. It isn't true for llama.cpp itself:

1. **ggml-org publishes prebuilt Windows CUDA binaries** on every release (`llama-<tag>-bin-win-cuda-<ver>-x64.zip`, plus the matching `cudart-*` zip). Nothing to compile. Pin a release tag in `chimera/deps.lock` next to Bifrost and Qdrant, and fetch it the same way the other `.deps` are fetched.
2. If you ever want to build from source, it builds on Windows with Visual Studio Build Tools, CMake and the CUDA Toolkit (`cmake -B build -DGGML_CUDA=ON`). Linux isn't required.

**Why native is the better fit here:**

- **The supervisor is a native Windows process manager.** It already handles tree-kill, no-console windows and log teeing (`internal/proc/*_windows.go`). A `llama-server.exe` child slots straight in. A Docker child would mean the supervisor manages Docker Desktop, a WSL2 VM and container lifecycles, all of which can fail on their own.
- **GPU access is direct.** No NVIDIA Container Toolkit, no WSL2 GPU paravirtualization, and no 9p/bind-mount penalty on 5–13 GB model loads (that penalty is why the old config used `--no-mmap`).
- **Memory:** the WSL2 VM reserves host RAM on top of the model, and the 35B MoE spills layers to CPU RAM.

**GPU detail you need:** this machine has an **RTX 5060 Ti 16 GB (Blackwell, `sm_120`)**, driver 596.49. The old Dockerfile targeted CUDA 12.4 with `89;90`. That doesn't produce native Blackwell kernels: it either fell back to PTX JIT or ran slower. **Use a CUDA 12.8+ (or 13.x) llama.cpp build.** The 596 driver supports both. This alone may change your benchmark numbers, so measure it in M0.

Fallback: if a future model needs vLLM, run it in WSL2 as an **external** endpoint (supervisor not involved). The Bifrost side of this design doesn't change.

## VRAM budget (16 GB)

| Model | Weights | Fits together? |
|-------|---------|----------------|
| nomic-embed v1.5 Q8_0 | ~0.15 GB (+ small KV) | always on |
| Qwen3.5-9B Q4_K_M + mmproj | ~6.5 GB + KV (≈100k ctx × 2 slots is several GB) | yes, with embed |
| Qwen3.5-35B-A3B IQ3_XXS + mmproj | ~13 GB, 29 layers on GPU, rest on CPU | **not** together with the 9B |

## Model shortlist (current + recommended downloads)

Fixed set, not a catalog. Exact repo names and quants must be confirmed on Hugging Face at download time (prefer `unsloth/` or `ggml-org/` / `Qwen/` official GGUFs; Unsloth "UD" dynamic quants are usually best per-GB).

| Role | Model | Status | Why |
|------|-------|--------|-----|
| Daily chat (fast) | **Qwen3.5-9B** Q4_K_M (+ mmproj) | have | Fits fully on GPU with long ctx; baseline. Try **Q5_K_M / Q6_K** too — 16 GB has room and quality per tok/s may be better. |
| Big chat (MoE) | **Qwen3.5-35B-A3B** | have (IQ3_XXS) | Only ~3B active → fast if experts are placed well. Re-download as **UD-Q4_K_XL** and run with `--n-cpu-moe` instead of `-ngl 29` (see optimization). |
| Mid dense | **Qwen3.5-27B** (UD-Q3/Q4) | **download, evaluate** | Dense middle option; check whether it beats the 35B-A3B on quality at acceptable tok/s on 16 GB. |
| Coding / tool use | **Qwen3-Coder-30B-A3B-Instruct** (UD-Q4_K_XL) | **download** | Non-thinking by design (no think tax), strong tool calling; same MoE offload trick. |
| Draft model (speculative decoding) | **Qwen3.5-0.8B** or **2B** (Q8_0) | **download** | Same tokenizer family as the 9B/27B → `-md` draft model; typically +30–80% gen tok/s on code/structured output. Not a service on its own. |
| Embeddings | **Qwen3-Embedding-0.6B** (Q8_0) | **download, evaluate** vs nomic v1.5 (have) | Much stronger retrieval than nomic, still tiny; 1024-d (MRL-truncatable). Requires re-index (we're re-indexing anyway). |
| Reranker (optional) | **Qwen3-Reranker-0.6B** | **download later** | llama-server `--reranking` → `/v1/rerank`; improves RAG top-k if the gateway adds a rerank step (separate plan). |

**Update (checked on Hugging Face, 2026-11-13):** newer releases exist and replace two rows: **Qwen3.6-35B-A3B** (`unsloth/Qwen3.6-35B-A3B-GGUF`, UD-Q4_K_XL) supersedes the 3.5 MoE, and **Qwen3.8-27B** (`unsloth/Qwen3.8-27B-GGUF`, UD-Q3_K_XL / Q4_K_XL) supersedes Qwen3.5-27B. No newer 9B/small size was found, so those stay on 3.5. Qwen3.8-Flash-Next (~180B total) is too large for this machine. llama.cpp Windows build: `b12365`, `win-cuda-13.5-x64` + matching `cudart` zip.

So "independent services" means each model is its own process with its own port, config and lifecycle, but the **launch list** decides which ones run. Default: `embed` + `qwen9b`. Switch to `qwen35b` by editing the list or using a start/stop control (M4).

## Architecture

### 1. Supervisor: generic `models:` children

Today each child (`startGatewayChild`, `startBrokerChild`, …) is hard-coded in `chimera-supervisor/internal/supervise/children.go`. Add **one generic loop** over `chimera.yaml` `models:` entries that are listed in `supervisor.services`, without adding new bespoke functions.

Per model, the supervisor:

1. Starts `llama-server.exe` directly with args built from config. No wrapper binary at first: llama-server already exposes `GET /health`, which returns 503 while loading and 200 when ready, so it can act as the readyz URL.
2. Tees stdout/stderr into servicelogs under source `chimera-model/<id>`.
3. Health-gates **before the broker starts**, with a long timeout (35B load ≈ 30–90 s). Make it optional (`wait: false`) so a slow model doesn't block the stack.
4. Restarts on crash with backoff. If not started, Bifrost returns a connection error for that provider only.
5. Reports per-model state in `/status`.

A wrapper (`chimera-modelhost`, following the [wrapper binary contract](../features/chimera-wrapper-binary-contract.md)) is deferred. Add it only if we need arg normalization, log-line normalization, or a stable `/readyz` beyond what `/health` gives.

### 2. Per-model config shape (`chimera.yaml`)

```yaml
models:
  runtime:
    llama_server_bin: "./chimera/.deps/llama.cpp/llama-server.exe"   # pinned in deps.lock
    models_dir: "C:/Users/lynnv/src/assistants/models"               # or ./data/models
  services:
    - id: qwen9b                     # → Bifrost provider "local-qwen9b"
      enabled: true
      kind: chat                     # chat | embed
      listen: "127.0.0.1:8180"
      model: "chat/Qwen3.5-9B-Q4_K_M.gguf"
      mmproj: "chat/mmproj-Qwen3.5-9B-BF16.gguf"
      served_name: "qwen3.5-9b"      # --alias; what clients put after "local-qwen9b/"
      gpu_layers: 99
      ctx: 102915
      parallel: 2
      sampling: { temp: 0.7, top_p: 0.8, top_k: 20, min_p: 0.0 }   # Qwen non-thinking defaults → server-side defaults
      extra_args: ["--jinja", "--reasoning-budget", "0", "--flash-attn", "on", "--perf"]
      wait: true
      startup_timeout: 120s
    - id: qwen35b
      enabled: true
      kind: chat
      listen: "127.0.0.1:8181"
      model: "chat/Qwen3.5-35B-A3B-UD-IQ3_XXS.gguf"
      mmproj: "chat/mmproj-F32.gguf"
      served_name: "qwen3.5-35b-a3b"
      gpu_layers: 29
      ctx: 262144
      parallel: 4
      extra_args: ["--jinja", "-t", "8", "-b", "512", "-ub", "512", "--flash-attn", "on", "--kv-unified"]
      wait: false
    - id: embed
      enabled: true
      kind: embed                    # adds --embeddings, --pooling as configured
      listen: "127.0.0.1:8182"
      model: "embed/nomic-embed-text-v1.5.Q8_0.gguf"
      served_name: "nomic-embed-text-v1.5"
      ctx: 2048
      extra_args: ["-b", "2048", "-ub", "2048"]

supervisor:
  services: [vectorstore, model:embed, model:qwen9b, broker, gateway, indexer]
```

Typed fields cover the knobs you actually tune. `extra_args` is the escape hatch, so new llama.cpp flags don't need a schema change. Port range `8180+` avoids the existing stack ports. Check against [`service-map.md`](../reference/service-map.md) before fixing it.

### 3. Bifrost wiring (chat + embed)

Each enabled model becomes a **keyless custom provider** in the broker config (`config/chimera-broker.config.json`, currently `providers.ollama` → `127.0.0.1:11434`):

```json
"local-qwen9b": {
  "keys": [],
  "network_config": { "base_url": "http://127.0.0.1:8180", "default_request_timeout_in_seconds": 600 },
  "custom_provider_config": { "base_provider_type": "openai", "is_key_less": true,
    "allowed_requests": { "chat_completion": true, "chat_completion_stream": true } }
},
"local-embed": {
  "keys": [],
  "network_config": { "base_url": "http://127.0.0.1:8182" },
  "custom_provider_config": { "base_provider_type": "openai", "is_key_less": true,
    "allowed_requests": { "embedding": true } }
}
```

(`CustomProviderConfig` lives in `.deps/bifrost/core/schemas/provider.go:401`; exact `allowed_requests` keys to confirm in M1.)

- **Chat:** clients and virtual models use `local-qwen9b/qwen3.5-9b`. The gateway's provider/model availability UI and routing generation (`brokeradmin`, `routinggen`, `operatorstore/provider_models_bootstrap.go`) currently special-case `ollama`. Teach them a `local-*` provider class: no API key, models come from llama-server `/v1/models`, never free-tier/catalog.
- **Embed:** `search.embedding.model: "local-embed/nomic-embed-text-v1.5"` replaces `ollama/nomic-embed-text:latest`. Still 768 dims, but a different quant and runtime than Ollama's build, so **re-index workspaces** rather than mixing vectors (the purge flow from `indexer-embedding-model-and-workspace-purge.md` applies).
- **Source of truth:** generate the `local-*` provider entries from `chimera.yaml` `models:` at supervisor start (merge into broker config, don't hand-edit), so a model is declared once.

### 4. Locus

No client changes. Locus talks to the gateway and sees `local-*` models in the model list like any other provider.

## Thinking control (no wasted reasoning tokens)

Qwen3.5 models think by default; leaked or runaway `<think>` blocks cost the most tok/s of anything here. Enforce "off" at three layers so it's consistent regardless of client:

1. **Server default (per service):** `--jinja --reasoning-budget 0` **and** `--chat-template-kwargs '{"enable_thinking": false}'` so the template never opens a think block. Config field: `thinking: off | on | budget:<n>`.
2. **Gateway guard:** for `local-*` chat providers, strip client attempts to enable thinking unless the virtual model allows it, and strip any residual empty `<think></think>` / `reasoning_content` from responses.
3. **Opt-in thinking as a separate thing:** if wanted, a second `served_name` (e.g. `qwen3.5-9b-think`) or a virtual model that sets `enable_thinking: true` + a budget — never the default.
4. **Measured:** the benchmark (below) records reasoning tokens per response; any non-zero count in "off" mode fails the run.

Sampling per Qwen's model card for non-thinking mode (≈ `temp 0.7, top_p 0.8, top_k 20, min_p 0`, presence penalty if loops appear) set as server defaults.

## Optimization loop (after it runs)

Goal: **maximize generation tok/s** (and prompt-processing tok/s for RAG-heavy turns) per model, with fixed quality checks so speed doesn't silently cost correctness.

**Measurement tooling**

- `llama-bench.exe` (ships in the same llama.cpp release) for raw engine numbers: pp512/pp4096 and tg128 across flag grids.
- A porcelain bench command (`chimera bench-models` or a Go test under `chimera/cmd/`) that goes **end-to-end through gateway → Bifrost → llama-server** with a fixed prompt set (short chat, long-context RAG prompt, tool call, code), streaming, and records: time-to-first-token, gen tok/s, pp tok/s (from llama-server `timings`), reasoning tokens, VRAM (`nvidia-smi`), and a pass/fail correctness check per prompt.
- Results appended to `data/bench/<model>/<date>.jsonl` with the exact args + llama.cpp build tag, so runs are comparable across reboots/driver/build changes. A small markdown summary per model goes into the feature doc.

**Knobs to sweep (in this order, biggest wins first)**

| Knob | Expectation |
|------|-------------|
| CUDA build: 12.4 vs **12.8+/13 with native sm_120** | Possibly large; verify first. |
| `--parallel 1` vs 2/4 | Single user → 1 slot gives all KV + compute to one stream; best single-request tok/s. |
| MoE: `--n-cpu-moe N` / `-ot "exps=CPU"` instead of `-ngl 29` | Keeps attention + shared layers on GPU, experts on CPU — usually far faster for 35B-A3B / Coder-30B-A3B on 16 GB. |
| KV cache quant `-ctk q8_0 -ctv q8_0` (with flash-attn) | Halves KV VRAM → more layers / longer ctx on GPU. |
| Context size | Only as large as actually used; huge ctx steals VRAM from layers. |
| Speculative decoding `-md <0.8B> --draft-max 16 --draft-min 4 -ngld 99` | Big gen tok/s win when acceptance is high. |
| `-b` / `-ub` batch sizes | Mainly prompt-processing speed. |
| `-t` threads (match P-cores), `--mlock`, `--no-mmap` | Matters for CPU-offloaded MoE layers and load time. |
| Quant choice (Q4_K_M vs Q5/Q6 vs UD) | Speed vs quality trade, judged by the correctness set. |

Winning settings get written back into that model's `chimera.yaml` entry; the bench run that justified them is linked in a comment.

### M0 baseline (2026-11-13, llama.cpp b12365 win-cuda-13.5, RTX 5060 Ti 16 GB, pre-reboot)

| Model | Settings | Prompt tok/s | Gen tok/s |
|-------|----------|-------------:|----------:|
| Qwen3.5-9B Q4_K_M | `-ngl 99 -fa 1` | ~3,250 | ~62 (58–67 via server) |
| Qwen3.6-35B-A3B UD-Q4_K_XL | `-ngl 99 -fa 1 --n-cpu-moe 14` | ~520–600 | ~50 |
| ″ | `--n-cpu-moe 8 / 11 / 12` | 63–200 | 2–13 (VRAM overflow → spill; avoid) |
| Qwen3.8-27B UD-Q3_K_XL | `-ngl 99 -fa 1` | ~970 | ~27 |

**Tuned (same day):** adding `-ctk q8_0 -ctv q8_0 -ub 2048 -t 8`:

| Model | Prompt tok/s | Gen tok/s |
|-------|-------------:|----------:|
| Qwen3.5-9B Q4_K_M | ~3,350 | **~70** |
| Qwen3.6-35B-A3B UD-Q4_K_XL (`--n-cpu-moe 14`) | ~1,550–2,050 | **~65–67** |
| Qwen3.8-27B UD-Q3_K_XL | ~1,000 | ~28 |

Losers: `-t 16` (SMT threads; tg drops to ~45), `-ub 4096` (VRAM spill), `--n-cpu-moe` < 14 (spill).
Embedders (server, 32-chunk batches, ~110-token chunks): nomic v1.5 Q8 ~530 chunks/s (768-d); Qwen3-Embedding-0.6B Q8 ~80 chunks/s (1024-d, `--pooling last`).

**Round 2 (more models, server-level, speculative decoding):**

| Model | Best setting | Prompt tok/s | Gen tok/s | Notes |
|-------|--------------|-------------:|----------:|-------|
| Qwen3.5-9B Q6_K | all GPU | ~2,870 | ~55 | 20% slower than Q4_K_M; quality gain unmeasured |
| Qwen3-Coder-30B-A3B UD-Q4_K_XL | `--n-cpu-moe 16` | ~2,200 | ~61–65 | <16 spills VRAM (noisy 30–45) |
| Qwen3.6-35B-A3B @ **64k ctx**, 19k-token prompt | `--n-cpu-moe 18` | ~1,490 | ~48 | `14` → 4 tok/s, `16` → 36 tok/s at this ctx: **CPU-MoE count must be tuned per ctx size** |

Speculative decoding (server, temp 0, 400-token code + prose answers):

| Config | Code tok/s | Prose tok/s |
|--------|-----------:|------------:|
| 9B no draft | 65.0 | 64.9 |
| 9B + 2B draft | 68.4 | 67.8 |
| 9B + n-gram (`--spec-type ngram-mod`) | 66.6 | 67.0 |
| 35B-A3B no draft | 52.5 | 57.9 |
| 35B-A3B + 0.8B draft | 57.8 | 60.6 |
| Coder no draft / + n-gram | 60.9 / 62.1 | 63.3 / 64.3 |

Gains are small (+3–10%). MTP (`--spec-type draft-mtp`) unavailable: these GGUFs lack MTP layers. Draft models cost VRAM that the MoE models need for context, so drafting is **not worth it by default**; n-gram (free, no VRAM) can stay on for code.
Note: without `--chat-template-kwargs '{"enable_thinking":false}'` the 9B thinks by default (seen in a test) — the flag is mandatory.

Thinking-off check (9B server, `--jinja --reasoning-budget 0 --chat-template-kwargs '{"enable_thinking":false}'`): no `reasoning_content`, no `<think>`, "17*23" answered in 4 tokens.

### M1 status (2026-11-13) — working

- Hand-started `llama-server` on `:8180` (Qwen3.5-9B, alias `qwen3.5-9b`, thinking off, 64k ctx) and `:8182` (nomic embed).
- Registered keyless custom providers `local-qwen9b` and `local-embed` via Bifrost `POST /api/providers` (persisted in `data/broker/config.db`) and in the seed `config/chimera-broker.config.json`.
- Verified through **gateway `:3000`** with the API key: chat, streaming, tool calls (`get_weather` → `{"city":"Paris"}`); embeddings via Bifrost `:8080` (768-d).
- **Gotchas found:**
  1. Bifrost only recognizes a new custom provider prefix (`local-x/…`) after a **broker restart**. Requests before that fail with "model should be in provider/model format". M3 must create providers before the broker starts, or restart it.
  2. An unreachable **`ollama` provider breaks Bifrost's whole `/v1/models`** (400) and with it the gateway catalog refresh. Remove or disable `ollama` in M5, or sooner.
  3. Gateway `/v1/models` lists only operator virtual models. Local models need VM/provider-availability entries (M4) before they appear in Locus's picker. Explicit `local-qwen9b/qwen3.5-9b` ids already work.

### M2 status (2026-11-13) — working

- `config/chimera.yaml` `search.embedding.model` → `local-embed/nomic-embed-text-v1.5` (dim 768 unchanged). Ollama provider removed from Bifrost (running DB + seed config).
- Re-indexed workspace 1 (`porcelain`): 4,795 files, 0 errors, 55,397 chunks (was 18,795 — the old Ollama-built index was incomplete; no duplicate `(source, chunk_index)` points). Search via `/api/ui/rag/search` returns relevant hits (scores ~0.67–0.74).
- **Bugs found (not fixed here):**
  1. `POST /api/ui/indexer/reindex-all` and `/workspaces/{id}/reindex` match `tenant_id = 'default'`, but the existing workspace row has `tenant_id = ''`. Result: "workspaces_bumped: 0" / "workspace not found". Worked around by bumping `reindex_generation` in `operator.sqlite` directly.
  2. `.claude/worktrees/**` was indexed, so most files appeared ~3× in results. **Fixed:** added repo-root `.chimeraignore` with `.claude/`, deleted 36,431 `.claude/` points from Qdrant, and restarted. The collection holds ~18,970 chunks, stable after restart.
- Qwen3-Embedding-0.6B (1024-d) vs nomic quality comparison deferred; it needs `dim: 1024` and a fresh collection.

### Evaluation round 3 (2026-11-13): quality + retrieval

**Embedders**: 22 paraphrased questions → expected `docs/features/*.md`, corpus = all 126 docs files (1,114 chunks):

| Embedder | hit@1 | hit@5 | MRR |
|----------|------:|------:|----:|
| nomic v1.5, no prefixes (deployed) | 7 | 18 | 0.55 |
| nomic v1.5 + `search_query:` / `search_document:` | **9** | 19 | **0.61** |
| Qwen3-Embedding-0.6B + instruct query | 8 | 20 | 0.61 |
| Qwen3-Embedding-0.6B plain | 7 | **21** | 0.55 |

Takeaway: no clear winner. nomic prefixes are a free gain (gateway change + re-index). The embed server's `-b/-ub 8192` held ~2.4 GB VRAM; `-b/-ub 2048` cuts it to ~1.3 GB at ample throughput.

**Chat quality**: 9 auto-scored tasks at temp 0, thinking off, 64k ctx, embed server co-resident. Tasks: word math, Go rune reverse (go test), Go ParsePort (go test), JSON extraction, tool choice, bug line, doc-grounded answer, 3-bullet format, ordering logic. Plus one long code generation timed for speed.

| Model | Score | Fails | Gen tok/s | VRAM |
|-------|------:|-------|----------:|-----:|
| Qwen3-Coder-30B-A3B (cpu-moe 18) | **8/9** | ParsePort (unused import) | 55 | 15.8 GB |
| Qwen3.8-27B Q3_K_XL | 7/9 | math, logic | 28 | 15.6 GB |
| Qwen3.6-35B-A3B (cpu-moe 18) | 7/9 | math, ParsePort | 48–53 | 15.6 GB |
| Qwen3.5-35B-A3B IQ3_XXS (old) | 7/9 | math, ParsePort | 71 | 12.4 GB |
| Qwen3.5-9B Q6_K | 7/9 | math, ParsePort | 53 | 10.6 GB |
| Qwen3.5-9B Q4_K_M | 7/9 | math, ParsePort | 67 | 9.3 GB |

Thinking leaks: 0 across all runs. Takeaway: this task set is **too easy to separate the models**; every model fails the same two non-thinking traps (pack-pricing math, Go unused import). Next eval needs harder, repo-specific tasks (multi-file reasoning, real tool loops via the harness). Surprise: the old IQ3_XXS 35B is the fastest MoE (smaller file fits more on GPU) at equal score.

### Evaluation round 4 (2026-11-13): harder, repo-specific set

10 points, temp 0, thinking off, 64k ctx, embed server co-resident. Tasks:
- **2 real-code rewrites:** blank `excerptFromFirstUserMessage` (`conversationtitle/title.go`) and `UpsertIfAbsent` (`envfile/upsert.go`), judged by the repo's own tests via `go test -overlay`, with one compiler-error repair round.
- **3 long-context facts:** buried in ~27k tokens of docs.
- **3 tool-loop questions:** real `list_dir` / `read_file` / `grep` over the repo, ≤12 steps.
- **2 multi-step reasoning problems.**

Harness: `%TEMP%\claude\bench\eval\hard.py`.

| Model | Score | Gen tok/s | Long-prompt tok/s | Wall |
|-------|------:|----------:|------------------:|-----:|
| Qwen3.8-27B Q3_K_XL | **9/10** | 24 | 275 (VRAM spill at 64k) | 361 s |
| Qwen3.5-35B-A3B IQ3_XXS (old) | 8/10 | **71** | 2,236 | **70 s** |
| Qwen3.6-35B-A3B UD-Q4_K_XL | 8/10 | 55 | 1,520 | 331 s |
| Qwen3.5-9B Q6_K | 7/10 | 50 | 2,328 | 135 s |
| Qwen3-Coder-30B-A3B | 7/10 | 51 | 603 | 159 s |
| Qwen3.5-9B Q4_K_M | 6/10 | 66 | 2,265 | 84 s |

Patterns:
- Every model except the 27B fails the `title.go` truncation logic, even after a repair round.
- Unused vars/imports are the dominant Go failure mode without thinking.
- The weekday problem fails for every MoE/9B except the 9B Q6.
- Thinking leaks: 0. A 1-point gap is within noise at n=10.

**Follow-up checks:**

| Check | Score | Gen tok/s | Long-prompt tok/s | Wall |
|-------|------:|----------:|------------------:|-----:|
| 27B @ **32k** ctx (vs 64k) | 9/10 | 23 | 892 (was 275) | 254 s (was 361) |
| 9B Q4, **thinking on** | **8/10** (was 6) | 65 | 2,861 | 421 s (was 84) |
| Old 35B IQ3_XXS, **thinking on** | 6/10 (was 8) | **17** (was 71) | 2,159 | **3,253 s** (was 70) |

- Thinking gives the 9B +2 points at ~5× the time.
- On the MoE, thinking was a disaster: long reasoning grew the context past the VRAM headroom of `--n-cpu-moe 8`, generation collapsed to 17 tok/s, and answers ran out of tokens.
- Confirms **thinking off as the default**. If thinking is ever offered, it needs a hard budget and a CPU-MoE count tuned for the longer context.

**Qwen3.6-35B-A3B UD-IQ3_XXS (13.2 GB)**, downloaded for this check:

| Setting | Score | Gen tok/s | Long-prompt tok/s | Wall |
|---------|------:|----------:|------------------:|-----:|
| `--n-cpu-moe 8` | 9/10 | 45 | 2,159 | 658 s |
| **`--n-cpu-moe 0` (all GPU)** | 8/10 | **95** | **3,459** | **72 s** |

- Fit check at 64k ctx with the embed server co-resident: a 28k-token prompt runs at 3,411 tok/s, gen 79 tok/s, using 15.3 / 16.3 GB VRAM.
- `--n-cpu-moe 2` gives ~0.5 GB extra headroom at ~74 tok/s.
- The 8 vs 9 score difference between offload settings is within noise.

**Decision (2026-11-13):**
- **Main chat:** Qwen3.6-35B-A3B UD-IQ3_XXS, all on GPU, 64k ctx, thinking off. Use `--n-cpu-moe 2` if other GPU apps need headroom.
- **Fast/light:** Qwen3.5-9B Q4_K_M.
- **Embeddings:** nomic v1.5 Q8 with `search_query:` / `search_document:` prefixes (to implement), `-b/-ub 2048`.
- **Thinking:** off by default. It only paid off on the 9B (+2 pts, 5× slower) and collapsed the MoE.
- **Drop candidates** (≈60 GB): Qwen3.5-35B IQ3_XXS (old), Qwen3.6-35B Q4_K_XL, Qwen3.8-27B, Coder-30B, 9B Q6, 0.8B/2B drafts.

### M3 status (2026-11-13): done

`main` already had a supervised `chimera-embed` (llama-server for internal RAG embeddings), so M3 extends that pattern instead of adding a parallel one:
- **Shared launcher:** `chimera/internal/llamaserver` holds the launcher and log normalizer, now with chat mode (alias, mmproj, `--n-cpu-moe`, parallel, thinking off by default, `extra_args`). `chimera-embed` uses it unchanged.
- **New wrapper:** `chimera-llm`, one per `local_models` entry, on the shared wrapper runtime (readyz, restart/backoff, log normalization).
- **Supervisor:**
  - Starts enabled models after embed and before the broker. A model failure is non-fatal.
  - Writes `local-<id>` providers into a materialized broker config, which BiFrost merges at startup. This avoids the "restart before new provider routes" gotcha from M1.
  - Prunes stale `local-*` providers.
  - Reports `details.local_models[]` on `/status`.
- **Port fix:** `chimera-embed`'s default listen moved 7750 → 7760; it collided with `chimera-indexer`.
- **Embeddings:** use the existing `internal_embedding:` block (`chimera-embed`) rather than a `local-embed` broker provider. The seed broker config no longer ships hand-made `local-*` or `ollama` providers.
- **Live-verified** in a scratch runtime: 35B ready, chat/stream/tools at 94 tok/s, crash → auto-restart, disable/enable → prune.

## Milestones

| # | Outcome | Notes |
|---|---------|-------|
| **M0** ✅ | Manual baseline on native Windows | Download a pinned llama.cpp Windows **CUDA 12.8+** release, run the 9B and embed by hand with the old flags, record tok/s (prompt + gen) and VRAM. Compare against the old Docker numbers and LM Studio. **Do after the reboot measurements; nothing long-running before then.** |
| **M1** ✅ | Bifrost → hand-started servers | Add `local-qwen9b` / `local-embed` custom providers to the broker config. Chat from Locus works through the gateway. Validate streaming, tool calls (`--jinja`) and vision via mmproj. |
| **M2** ✅ | Embedder swap | Point `search.embedding.model` at `local-embed`, purge/re-index one workspace, sanity-check retrieval quality vs. Ollama. |
| **M3** ✅ | Supervisor-managed models | Shipped as `local_models:` + `chimera-llm`. See [feature record](../features/local-chat-models.md). |
| **M4** | Operator controls | Start/stop individual models from `/ui` settings (9B ↔ 35B swap), per-model log view, `local-*` handling in provider availability UI. |
| **M4b** | Bench harness + thinking guard | End-to-end bench command, jsonl results, reasoning-token check; gateway thinking guard for `local-*`. |
| **M4c** ✅ (manual) | Optimization passes | Per model: CUDA build → parallel → MoE offload → KV quant → spec decoding → quant choice. Download + evaluate shortlist models (27B, Coder-30B-A3B, 0.8B draft, Qwen3-Embedding, reranker). Lock winners into config. |
| **M5** | Retire Ollama defaults | Remove `ollama` from example configs/docs and `make configure` defaults (keep the Bifrost provider usable for anyone who wants it). Add feature doc `docs/features/local-inference-services.md`. |

## Open questions for you

1. **Model storage:** reference `C:\Users\lynnv\src\assistants\models` in place, or copy into porcelain `data/models/` (gitignored)?
2. **Default set:** `embed` + `qwen9b` always on, `qwen35b` opt-in. Is that right? Any other model (e.g. a Qwen coder, a reranker) in the fixed set?
3. **Thinking mode:** keep `--reasoning-budget 0` (no thinking) as the default, or expose a thinking variant as a second `served_name`/service?
4. **Embedding model:** stay on nomic-embed v1.5 (known-good, 768-d), or switch now to something like Qwen3-Embedding-0.6B while we're re-indexing anyway?
5. **Wrapper:** is launching `llama-server.exe` directly OK, or do you want it behind a `chimera-modelhost` wrapper from day one for consistency with the other services?
