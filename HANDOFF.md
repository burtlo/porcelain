# Handoff: Chimera harness qualitative eval

Continue this evaluation on a machine with more resources. Paste this entire document to the agent on that system.

**Your job (in order):**

1. Inventory local Ollama models and hardware headroom
2. Suggest (and after operator OK, download) better models for this eval
3. Wire those models into the eval VM profiles
4. Verify stack timeouts, create VMs, run the matrix
5. Grade the run with `EVAL_AGENT.md`

Do not skip the Ollama survey — the previous machine was M1 16GB and used small models as a necessity.

---

## Context

Sparse qualitative matrix for Chimera **virtual-model turn harness** profiles:

- Fixed **prompt corpus** about the porcelain codebase
- Several **named virtual models** with different harness settings
- Each prompt × each model, with workspace scope
- Record **TTFT** and **total latency**
- Grade against per-prompt criteria (skill/MCP usefulness)

Goal: compare harness features (retrieval, summarize, intent, tools, judge/escalation), not every knob.

---

## What is in git vs what is not

### Porcelain repo (checked in / pushed)

- **Branch:** `version-0.4.0` (`origin/version-0.4.0`)
- Timeout fix: VM chat uses `gateway.timeouts.chat_ms` (no hard 60s clamp)
- Config:
  - `config/chimera.yaml` → `gateway.timeouts.chat_ms: 300000`
  - `config/chimera-broker.config.json` → Ollama `default_request_timeout_in_seconds: 300`

### Eval pack (NOT in porcelain git)

Sibling of porcelain so results are not indexed:

```
<path>/porcelain/          # git repo, branch version-0.4.0
<path>/harness-eval/       # COPY THIS SEPARATELY
```

| Path | Purpose |
|------|---------|
| `HANDOFF.md` | This document |
| `README.md` | Short run notes |
| `EVAL_AGENT.md` | Grading instructions |
| `profiles.yaml` | Six VM harness profiles (edit when changing models) |
| `prompts.yaml` | Six prompts + criteria |
| `setup_vms.py` | Upsert Eval-* VMs; set workspace `file_action_policy=read_write` |
| `run_eval.py` | Stream matrix; write `runs/<UTC>/` |
| `runs/` | Prior results (optional) |

---

## Phase 0 — Inventory Ollama and suggest downloads

Run these first and report to the operator before pulling anything large:

```bash
# Hardware
sysctl -n machdep.cpu.brand_string hw.memsize 2>/dev/null || true
# Linux alternative:
# nproc; free -h; nvidia-smi 2>/dev/null || true

# What is already installed / loaded
ollama list
ollama ps

# What BiFrost currently sees (after stack is up)
curl -s -H "Authorization: Bearer chimera-local-open" http://127.0.0.1:3000/v1/models \
  | python3 -c "import sys,json; print('\n'.join(sorted(x['id'] for x in json.load(sys.stdin).get('data',[]) if str(x.get('id','')).startswith('ollama/'))))"
```

### Roles to fill (keep this structure)

| Role | Used by | Previous (M1 16GB) | Prefer on bigger machine |
|------|---------|--------------------|---------------------------|
| **Primary** | Baseline, RAG-trunc, RAG-sum, Intent-RAG, Judge | `qwen3:8b` | Stronger general chat/reasoner |
| **Side / summarize / judge** | Summarize model, evaluator | `llama3.2:3b` | Still small–mid for latency; can bump if RAM allows |
| **Tools / coder** | Eval-Tools-coder primary | `qwen2.5-coder:7b` | Stronger coder with tool calling |
| **Embeddings** | Gateway search | `nomic-embed-text` | Keep unless operator already uses another embed model with matching dim in `chimera.yaml` |

### Suggested Ollama pulls by RAM tier

Pick one tier; do not pull everything. Prefer Q4 defaults from Ollama library tags.

**~16GB (baseline — what we already ran)**  
Keep: `qwen3:8b`, `llama3.2:3b`, `qwen2.5-coder:7b`, `nomic-embed-text`

**~24–32GB (recommended next step)**  
Suggest downloading if missing:

```bash
ollama pull qwen3:14b          # or qwen3:30b-a3b if available and fits
ollama pull qwen2.5-coder:14b  # tools primary upgrade
ollama pull llama3.2:3b        # keep as fast side model (or phi4-mini)
ollama pull nomic-embed-text
# optional quality bump:
# ollama pull gemma3:12b
```

Proposed role mapping for 24–32GB:

- Primary: `ollama/qwen3:14b`
- Side/judge/summarize: `ollama/llama3.2:3b` (or `phi4-mini`)
- Tools: `ollama/qwen2.5-coder:14b`
- Embed: `ollama/nomic-embed-text:latest`

**~48–64GB+**  

```bash
ollama pull qwen3:32b
ollama pull qwen2.5-coder:32b
# optional:
# ollama pull gemma3:27b
ollama pull llama3.2:3b
ollama pull nomic-embed-text
```

Proposed mapping:

- Primary: `ollama/qwen3:32b`
- Side: `ollama/llama3.2:3b` or a mid judge like `qwen3:8b`
- Tools: `ollama/qwen2.5-coder:32b`

### Agent behavior for suggestions

1. Print `ollama list` and estimated free RAM
2. Propose a concrete table: role → current tag → suggested tag → pull command
3. **Ask the operator to confirm** before long downloads (unless they already said “pull what you recommend”)
4. After pulls, confirm models appear in gateway `/v1/models` (may need a short wait / broker catalog refresh)
5. Update `profiles.yaml` fallback chains + `defaults.primary_model` / `side_model` / summarize / evaluator `model_id` fields to the new tags
6. Re-run `setup_vms.py` so SQLite VMs match

**Pin one primary across Baseline / RAG / Intent / Judge** so harness differences are not confounded with model strength. Only Tools may use a different coder primary.

---

## Phase 1 — Stack + timeout preflight

```
run_eval.py (600s)
  → gateway :3000  (chat_ms=300000)
    → bifrost :8080  (Ollama timeout=300)
      → ollama :11434
```

Verify live BiFrost timeout (config on disk ≠ process until restart):

```bash
curl -s http://127.0.0.1:8080/api/providers | python3 -c \
  "import sys,json; p=[x for x in json.load(sys.stdin)['providers'] if x['name']=='ollama'][0]; print(p['network_config'])"
# Must show default_request_timeout_in_seconds: 300
```

If wrong: set `config/chimera-broker.config.json` (and live `data/broker/config.db` if needed), **restart broker/supervisor**, re-check.

Smoke:

```bash
curl -N -H "Authorization: Bearer chimera-local-open" \
  -H "Content-Type: application/json" \
  -H "X-Chimera-Project: porcelain" \
  -d '{"model":"Eval-Baseline-plain-1.0","stream":true,"messages":[{"role":"user","content":"Say PONG"}]}' \
  http://127.0.0.1:3000/v1/chat/completions
```

Workspace: indexed **project_id=`porcelain`**. API key: `chimera-local-open`.

---

## Phase 2 — Create VMs and run the matrix

```bash
cd /path/to/harness-eval
python3 -m venv .venv && .venv/bin/pip install pyyaml   # once
.venv/bin/python setup_vms.py
.venv/bin/python run_eval.py
```

Writes:

```
harness-eval/runs/<YYYYMMDDTHHMMSSZ>/
  SUMMARY.md
  results.json
  responses/*.md
  EVAL_AGENT.md
```

Matrix: **6 prompts × 6 profiles = 36 cells**. Leave overnight if needed.

### Profiles (names hint config)

| Model id | Intent |
|----------|--------|
| `Eval-Baseline-plain-1.0` | All modules off |
| `Eval-RAG-trunc-1.0` | Retrieval truncate |
| `Eval-RAG-sum-1.0` | Retrieval summarize |
| `Eval-Intent-RAG-1.0` | Intent + retrieval |
| `Eval-Tools-coder-1.0` | Tools (+ retrieval); coder primary |
| `Eval-Judge-esc-1.0` | Retrieval + single_pass eval + escalation; gated stream |

### Prompts (`prompts.yaml`)

`harness-modules-overview`, `rag-scope-headers`, `retrieval-compress-strategies`, `routing-pipeline-order`, `tools-read-feature-record`, `hard-escalation-style` — all porcelain-workspace, with must/should/must_not criteria.

---

## Phase 3 — Grade

Follow `EVAL_AGENT.md`. Write `AGENT_REPORT.md` in the run directory; update `SUMMARY.md` Quality evaluation section.

Compare to prior best run if available: `runs/20260727T050045Z/` (0 exceed · 17 meet · 19 fail; recommended **Eval-RAG-trunc**; 0 i/o timeouts after BiFrost restart).

Known remaining product gaps (not timeout):

- **Eval-Judge-esc** — empty bodies at long latency
- **Eval-Tools-coder** — tool JSON instead of answers
- **tools-read-feature-record** — weak across profiles

---

## Prior best run (reference)

`20260727T050045Z` on M1 16GB after 5m timeouts + broker restart:

- 30/36 ok, `finish_reason=stop` on 30, **0** `:11434 i/o timeout`
- Recommended default: **Eval-RAG-trunc**

---

## Success criteria for this machine

1. Ollama inventory + recommended download list delivered; models pulled (with operator OK)
2. `profiles.yaml` + VMs updated to new model ids; visible on `/v1/models`
3. Live BiFrost Ollama timeout = 300; smoke stream works
4. New `runs/<datetime>/` with 36 responses + timing matrix
5. `AGENT_REPORT.md` with exceed/meet/fail, latency notes, skill/MCP usefulness, recommended default profile
6. Explicit callout: did larger models improve RAG/Intent; are Judge/Tools still broken?

---

## Do not confuse

- Gallery fixtures ≠ this eval
- Feature records in `docs/features/` are SoT; plans are history
- Keep results **outside** porcelain so they are not RAG-indexed
- Empty Judge-esc ≠ the old 60s Ollama timeout

---

## Cheat sheet

```bash
ollama list && ollama ps

curl -s http://127.0.0.1:8080/api/providers | python3 -c \
  "import sys,json; p=[x for x in json.load(sys.stdin)['providers'] if x['name']=='ollama'][0]; print(p['network_config'])"

cd /path/to/harness-eval
# edit profiles.yaml if changing models
.venv/bin/python setup_vms.py
.venv/bin/python run_eval.py
# Grade runs/<latest>/ with EVAL_AGENT.md
```
