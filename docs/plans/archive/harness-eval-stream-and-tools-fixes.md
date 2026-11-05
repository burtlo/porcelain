# Plan: Harness stream delivery and Ollama tool parsing

| Field                          | Value                                                                 |
|--------------------------------|-----------------------------------------------------------------------|
| **Doc kind**                   | `feature-plan`                                                        |
| **Owners / areas**             | Gateway harness (`internal/harness`), harness-eval runner             |
| **Status**                     | `shipped`                                                             |
| **Targets**                    | gateway v0.4, harness qualitative eval                                |
| **Last updated**               | 2026-08-03                                                            |
| **Supersedes / superseded by** | None                                                                  |
| **As-built**                   | None — link to [`gateway-chat-routing-pipeline.md`](../features/gateway-chat-routing-pipeline.md) when shipped |

## At a glance

Streaming clients (including the harness eval runner) must receive SSE when they request `stream: true`, even when the turn harness buffers an internal non-stream upstream completion for evaluator gating or workspace tool loops. Ollama coder models often emit tool intent as JSON in `message.content` rather than OpenAI `tool_calls`; the harness must parse those shapes so tool_executor profiles return answers instead of raw tool JSON.

| Phase                                                                 | Outcome                                                                 | Status |
|-----------------------------------------------------------------------|-------------------------------------------------------------------------|--------|
| [Phase 1 — Buffered stream delivery](#phase-1--buffered-stream-delivery) | Gated evaluator and tool-loop profiles stream SSE to clients            | `done` |
| [Phase 2 — Ollama tool-call parsing](#phase-2--ollama-tool-call-parsing) | Workspace tool loop executes content JSON and `<tool_call>` tags        | `done` |
| [Phase 3 — Reasoning-aware completion text](#phase-3--reasoning-aware-completion-text) | Evaluator and capture hooks see Qwen3 answers when content is empty   | `done` |
| [Phase 4 — Eval rerun and grade](#phase-4--eval-rerun-and-grade)      | Repeat baseline harness matrix; confirm Judge-esc and Tools-coder      | `done` |

---

## Background

Harness qualitative eval run `20260803T014047Z` (same models as prior M1 baseline) showed **30/36** transport-ok cells but **6/6 Eval-Judge-esc** empty bodies when the client used `stream: true`. Live comparison proved Judge-esc returns valid JSON with content on `stream: false` but raw JSON (zero SSE lines) on `stream: true` — a harness delivery bug, not a model failure. **Eval-Tools-coder** often returned bare `read_file` JSON because `completionToolCalls` only parses OpenAI `tool_calls`, while Ollama `qwen2.5-coder:7b` puts tool intent in `content`. Qwen3 models may also populate BiFrost `reasoning` separately from `content`; evaluator capture currently ignores `reasoning`.

**Related docs:** [`virtual-model-harness-evaluator-escalation.md`](virtual-model-harness-evaluator-escalation.md), [`virtual-model-harness-workspace-tools.md`](virtual-model-harness-workspace-tools.md), [`gateway-chat-routing-pipeline.md`](../features/gateway-chat-routing-pipeline.md), harness-eval `HANDOFF.md`.

---

## Phase 1 — Buffered stream delivery

**Goal.** Clients that request streaming receive valid SSE for harness paths that buffer internal completions (evaluator `gate_on_evaluator`, `buffer_until_complete`, workspace tool loop final delivery).

**Deliverables**

- `writeBufferedResponse` accepts client stream intent (`TurnContext.Stream`) and converts buffered JSON completions to SSE when needed.
- Unit tests: JSON buffer + `clientStream=true` produces `data:` lines and `[DONE]`.
- All harness call sites pass stream intent.

**Acceptance**

- Live smoke: `Eval-Judge-esc-1.0` with `stream: true` returns SSE with non-empty `delta.content`.
- `go test ./chimera/chimera-gateway/internal/harness/...` passes.

**Status:** `done`

---

## Phase 2 — Ollama tool-call parsing

**Goal.** Workspace tool_executor loop recognizes tool intent from Ollama coder models and executes tools instead of returning raw JSON to the client.

**Deliverables**

- Extend `completionToolCalls` to parse JSON objects in `message.content` (including markdown-fenced JSON) and `<tool_call>...</tool_call>` blocks per Ollama template.
- Unit tests mirroring eval failure payloads (`read_file` JSON-only responses).

**Acceptance**

- Live smoke: `Eval-Tools-coder-1.0` on `rag-scope-headers` returns prose, not a single tool-call JSON blob.
- Harness tests pass.

**Status:** `done`

---

## Phase 3 — Reasoning-aware completion text

**Goal.** Internal evaluator capture and broker completion paths treat BiFrost/Ollama `reasoning` as fallback when `content` is empty.

**Deliverables**

- `completionText` and `brokerCompletion` read `reasoning` / `thinking` when `content` is blank (JSON and SSE).
- `run_eval.py` optional: parse JSON body when no SSE lines (defense in depth).

**Acceptance**

- Unit tests for reasoning-only JSON/SSE payloads.
- Evaluator `single_pass` receives non-empty primary text when model puts answer in reasoning field only.

**Status:** `done`

---

## Phase 4 — Eval rerun and grade

**Goal.** Confirm fixes on the unchanged baseline model stack (`qwen3:8b` / `llama3.2:3b` / `qwen2.5-coder:7b`) before any `qwen3:14b` follow-up run.

**Deliverables**

- Rebuild gateway; restart supervised stack.
- `setup_vms.py` (unchanged profiles) + `run_eval.py` full 36-cell matrix.
- `AGENT_REPORT.md` grading; compare Judge-esc and Tools-coder cells to `20260803T014047Z`.

**Acceptance**

- Judge-esc: 6/6 non-empty responses with measurable TTFT.
- Tools-coder: majority prose answers on file-oriented prompts (not raw tool JSON only).
- Documented grade delta in run `SUMMARY.md` and `AGENT_REPORT.md` for `20260803T234717Z`.

**Status:** `done`

---

## Open questions

1. Should gateway expose Ollama `think: false` on virtual models (VM config) vs prompt `/no_think` only? Deferred — Phase 3 uses capture fallback first.
2. Should `buffer_until_complete` share the same SSE encoder as Phase 1? Yes — same helper.

---

## References

- Code: `chimera/chimera-gateway/internal/harness/stages.go`, `tools_stage.go`, `evaluator.go`
- Eval: `../harness-eval/run_eval.py`, run `20260803T014047Z`
- Diagnosis: harness eval handoff session (stream vs non-stream comparison)
