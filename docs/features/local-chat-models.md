# Feature: Local chat models

| Field | Value |
|-------|-------|
| **Doc kind** | `feature-record` |
| **Areas** | `chimera-llm`, supervisor, broker (BiFrost) provider registration |
| **Status** | `current` |
| **Introduced** | v0.5 local-inference work (M3) |
| **Originated from** | [`plans/local-inference-services.md`](../plans/local-inference-services.md) |
| **Related features** | [Internal embedding provider](internal-embedding-provider.md), [Wrapper binary contract](chimera-wrapper-binary-contract.md), [Chimera stack configuration](chimera-stack-config.md) |
| **Depends on** | Operator-supplied `llama-server` and GGUF weights |
| **Last updated** | See git history |

## At a glance

`chimera.yaml` `local_models:` lists a small, fixed set of chat models. For each enabled entry, `chimera-supervisor` starts one `chimera-llm` wrapper, which runs `llama-server` with that model's tuned settings. Before the broker starts, the supervisor registers every enabled model with BiFrost as a keyless OpenAI-compatible provider named `local-<id>`. Clients (Locus, gateway virtual models) then call `local-<id>/<alias>` like any hosted model.

Porcelain does not download weights or `llama-server`. Both are operator-supplied.

## Operator-visible behavior

- Add models under `local_models.models` (see `config/chimera.example.yaml`). `enabled: false` keeps an entry configured without launching it.
- **Thinking is off by default** (`--reasoning-budget 0` plus `enable_thinking: false` template kwargs). Set `thinking: true` per model to opt in.
- **A model that fails to start or load does not stop the stack.** The supervisor logs `chimera-supervisor.local_model.not_ready`, and `GET /status` on the supervisor shows `ready: false` for it. The wrapper keeps restarting `llama-server` with backoff.
- Supervisor `/status` → `details.local_models[]` lists `id`, `provider`, `model` (the client model id), `endpoint`, `listen`, and `ready`.
- Logs from all local models land in the `chimera-llm` service log stream, with `model_id` on wrapper lines.

## System behavior and contracts

**Startup order:** vectorstore → embed (when enabled) → **local models** → broker → gateway → indexer.
- Local models start concurrently (separate processes).
- The supervisor waits for each model with `wait: true` (default) up to `startup_timeout_seconds` (default 300), unless `-no-wait-llm` is set.

**Broker registration:**
- The supervisor writes `<broker data dir>/chimera-broker.materialized.json`: the seed broker config (`BROKER__CHIMERA_BROKER_CONFIG` or `config/chimera-broker.config.json`) plus one `local-<id>` provider per enabled model.
- It passes that file to `chimera-broker` via `BROKER__CHIMERA_BROKER_CONFIG`.
- BiFrost merges new or changed providers from its config file into its config store at startup. This matters because providers added later through the HTTP API are not routable until a restart.
- A provider with the same name already in the seed is left untouched (the operator's definition wins).

**Stale provider pruning:** after the broker is ready, the supervisor deletes `local-*` providers from BiFrost that are not enabled local models. One configured-but-unreachable provider makes BiFrost's `/v1/models` fail for every provider.

**llama-server launch (chat mode):**
- Always: `-m`, `--host`, `--port`, `-c`, `--n-gpu-layers`, `--jinja`, `--metrics`, `--parallel`.
- When set: `--alias`, and `--n-cpu-moe` (if > 0).
- Vision: `--mmproj <file>` when set, otherwise `--no-mmproj`.
- Thinking flags as described above.
- `extra_args` verbatim, last.
- Readiness is llama-server `GET /health`: 503 while loading, 200 when ready.

**llama-server selection:** `-llm-backend-bin`, then `local_models.llama_server_bin`, then the embed backend default (`-embed-backend-bin`).

**Ports:**
- `chimera-llm` wrappers listen on `127.0.0.1:7770 + index` unless `listen:` is set.
- Each model's `endpoint` is its llama-server.
- Enabled models must not share endpoints or listens (validated at load).
- `chimera-embed` moved to `7760`; `7750` belongs to `chimera-indexer`.

## Interfaces

| Surface | Detail |
|---------|--------|
| Config | `local_models.llama_server_bin`, `models_dir`, `models[]`: `id`, `enabled`, `endpoint`, `listen`, `model`, `mmproj`, `alias`, `ctx_size`, `n_gpu_layers`, `n_cpu_moe`, `parallel`, `thinking`, `extra_args`, `wait`, `startup_timeout_seconds` |
| Launch list | `supervisor.services` entry `local_models` (default: included when any model is enabled) |
| Supervisor flags | `-llm-bin`, `-llm-backend-bin`, `-no-wait-llm` |
| Wrapper flags/env | `chimera-llm -listen -bin -endpoint -model-id -model-path -mmproj-path -alias -cache-dir -ctx-size -n-gpu-layers -n-cpu-moe -parallel -thinking -extra-args` (JSON array); `LLM__*` env equivalents |
| Client model id | `local-<id>/<alias>` through the broker / gateway |

## Code map

| Concern | Location |
|---------|----------|
| Config schema / validation | `chimera/internal/config/local_models.go` |
| Shared llama-server launcher + log normalizer | `chimera/internal/llamaserver/` (also used by `chimera-embed`) |
| Wrapper | `chimera/chimera-llm/` |
| Supervisor lifecycle | `chimera/chimera-supervisor/internal/supervise/local_models.go` |
| Broker provider materialization / pruning | `chimera/chimera-supervisor/internal/supervise/broker_local_providers.go` |
| Status | `chimera/chimera-supervisor/internal/control/` |

## Verification

```bash
go test ./chimera/internal/config/ ./chimera/internal/llamaserver/ ./chimera/chimera-llm/... ./chimera/chimera-supervisor/... -count=1
```

Manual (2026-11-13, RTX 5060 Ti 16 GB, llama.cpp b12365 CUDA 13.5):
- Supervised Qwen3.6-35B-A3B UD-IQ3_XXS: ready on `/status`, listed by BiFrost `/v1/models` as `local-qwen35b/qwen3.6-35b-a3b`.
- Chat, streaming and tool calls work, with no reasoning output, at 94 tok/s.
- Killing `llama-server` → the wrapper restarts it (`restarts: 1`) and chat resumes.
- Disabling the model and enabling another → `local-qwen35b` pruned, and `/v1/models` stays healthy.

## Known gaps

- Enabling, disabling or retuning a model needs a supervisor restart. There is no UI start/stop yet (plan M4).
- Local models don't appear in the Locus model picker until a gateway virtual model or provider-availability entry references them (plan M4).
- No VRAM admission check: two large models enabled together can overflow the GPU, which makes both slow. Keep only what fits enabled.
- Per-model log streams share the `chimera-llm` source.
