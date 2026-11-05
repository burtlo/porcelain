# Plan: Assistant reliability reset

| Field                          | Value                                                                 |
|--------------------------------|-----------------------------------------------------------------------|
| **Doc kind**                   | `feature-plan`                                                        |
| **Owners / areas**             | Gateway chat path, harness, provider limits, harness-eval             |
| **Status**                     | `draft`                                                               |
| **Targets**                    | Takes priority over remaining v0.5 assistant and harness themes       |
| **Last updated**               | 2026-11-05                                                            |
| **Supersedes / superseded by** | Pauses new work from [`assistant-harness-workspace-search-quality.md`](assistant-harness-workspace-search-quality.md), [`operator-model-assist-tools.md`](operator-model-assist-tools.md), and deferred phases of [`gateway-mcp-tool-backends.md`](gateway-mcp-tool-backends.md) |
| **As-built**                   | None — link to [`docs/features/`](../features/README.md) when shipped |

## At a glance

Make an assistant one dependable model endpoint: a coding tool or game sends one request and gets a usable answer, or a clear error, without implementing its own retry and limit handling. New harness features pause until the core is proven by tests, and optional modules stay on only if they measurably beat a no-harness control.

| Phase                                                                    | Outcome                                                              | Status |
|--------------------------------------------------------------------------|----------------------------------------------------------------------|--------|
| [Phase 0 — Consolidate and freeze](#phase-0--consolidate-and-freeze)     | One trunk; unproven modules clearly marked experimental              | `done` |
| [Phase 1 — Assistant contract](#phase-1--assistant-contract)             | Written behavior for every failure a client can see                  | `todo` |
| [Phase 2 — Failure test suite](#phase-2--failure-test-suite)             | Contract proven by repeatable tests with no real LLM                 | `todo` |
| [Phase 3 — Accurate local models](#phase-3--accurate-local-models)       | Ollama never silently loses context or leaks thinking into answers   | `todo` |
| [Phase 4 — Structured output](#phase-4--structured-output)               | Games and mods can request validated JSON through one call           | `todo` |
| [Phase 5 — Outcome-based eval](#phase-5--outcome-based-eval)             | Evidence for which modules and routes actually improve results       | `todo` |
| [Phase 6 — Module decisions](#phase-6--module-decisions)                 | Every frozen module is kept, reworked, or removed based on evidence  | `todo` |

---

## Background

Assistants (formerly virtual models) exist to deliver better LLM service than any single model by routing across free cloud tiers and local inference, working around context and request limits, and hiding retry logic from clients such as coding tools and games.

The v0.4 harness plans are marked `done` because their code shipped, not because they met a quality target. Evidence from the sibling `harness-eval` repo, run `20261001T003156Z` (0 exceed, 8 meet, 13 fail on operations):

- **No demonstrated harness gain.** On prompts shared with the no-harness control, no module beat it. Daily did worse on one (evaluate dry-run).
- **Core goals are untested.** The corpus asks about Porcelain's own docs. All fallback chains are Ollama-only, with no quota, failover, coding-task, or structured-output cases.
- **Failures can't be diagnosed.** Results don't record retrieved chunks or tool calls, so retrieval misses and model failures look alike.
- **Assistants aren't directly comparable.** Each ran a different prompt subset, once. At least one recorded failure reason is wrong (`request_witness` is a real stage).

Measured on 2026-11-05 against Ollama 0.23.2 on the OpenAI-compatible endpoint: `llama3.2:3b` and `qwen3:8b` both capped a roughly 73,000-character prompt at exactly 32,768 prompt tokens with no error. `llama3.2:3b` lost content from the start of the prompt. `config/provider-model-limits.yaml` lists `llama3.2:3b` at 131,072 tokens and has no `qwen3:8b` entry.

**Related docs:** [`operator-assistants.md`](../features/operator-assistants.md), [`gateway-chat-routing-pipeline.md`](../features/gateway-chat-routing-pipeline.md), [`context-window-admission.md`](../features/context-window-admission.md), [`assistant-harness-internals.md`](../features/assistant-harness-internals.md).

---

## Phase 0 — Consolidate and freeze

**Goal.** Everyone works from one trunk, and it's obvious which assistant features are proven.

**Deliverables**

- Merge `version-0.5.0` into `main`, which had diverged since v0.3.3 (23 and 15 commits).
- Label `intent`, `evaluator` (including `multi_draft`), `escalation` (`ensemble`, `human`), and MCP as **experimental** in the settings UI and feature records. Behavior and stored settings stay unchanged.
- Pause new work on the plans this plan supersedes. Non-assistant v0.5 themes are unaffected.
- Add a `plans/README.md` rule: an assistant plan with a user-facing quality claim is `done` only when it has a passing test or eval result.

**Acceptance**

- A single agreed trunk contains the harness and the current `main` work, with `make precommit` passing.
- Existing assistants behave as before; only labels and docs change.

**Status:** `done`

---

## Phase 1 — Assistant contract

**Goal.** A client author can predict what happens for every failure without reading gateway code.

**Deliverables**

- New feature record `docs/features/assistant-client-contract.md` defining behavior for each case below. Behaviors shown are starting proposals to confirm against the code.

| Case | Proposed behavior | Client sees |
|------|-------------------|-------------|
| Upstream 429, or quota known to be exhausted | Skip to next chain entry | Normal answer; upstream in `X-Chimera-Upstream-Model` |
| Prompt too large for a model | Skip before calling it | Normal answer from a larger model |
| Prompt too large for every model | No truncation by default | Distinct, machine-readable error |
| Upstream 5xx or timeout before first byte | Next chain entry | Normal answer |
| Failure after streaming began | Never restart invisibly | Defined terminal stream error |
| Chain exhausted | Stop | Stable error type, per-attempt reasons, retry hint when known |
| Invalid structured output | Repair and retry within a limit ([Phase 4](#phase-4--structured-output)) | Valid JSON or a distinct error |
| Module failure (retrieval, evaluator, tool router) | Fail open, recorded in response metadata | Normal answer |

- Stable error `type` values and a total-time budget for a whole turn, including all fallback attempts.
- Document current gaps as `partial` rather than describing intended behavior as shipped.

**Acceptance**

- Every row maps to named tests in [Phase 2](#phase-2--failure-test-suite).
- A reader can write a minimal client from this document alone.

**Status:** `todo`

---

## Phase 2 — Failure test suite

**Goal.** Reliability is proven without a GPU, network access, or provider quota.

**Deliverables**

- Scriptable fake upstream built on the `httptest` pattern in `internal/chat/chat_limits_test.go`, able to return 429s, quota errors, context errors, 5xx, slow responses, and mid-stream disconnects, per model and per attempt.
- End-to-end tests through `POST /v1/chat/completions` on an assistant, in both streaming and non-streaming modes, with the full harness path enabled.
- Fixes for whatever the suite exposes in `chat.WithAssistantFallback` and related paths.
- Include in `make precommit`.

**Acceptance**

- Every contract row has passing streaming and non-streaming tests, or a documented reason one mode doesn't apply.
- The suite is deterministic across 20 consecutive runs.

**Status:** `todo`

---

## Phase 3 — Accurate local models

**Goal.** Local models fail over instead of silently degrading.

**Deliverables**

- Use the context each local model is actually loaded with for admission, rather than the architecture maximum. Establish how to read or set this through BiFrost and Ollama.
- Detect truncation that slips through: `usage.prompt_tokens` at the effective limit with a larger estimated prompt is logged and treated as a retriable overflow when another candidate exists.
- Correct `provider-model-limits.yaml` Ollama entries, and warn operators about chain models with no known context limit.
- Keep model reasoning out of `content` on every path, streaming and buffered, including inline thinking text from qwen3.
- Check the token estimator against reported usage, since admission depends on it.

**Acceptance**

- Rerunning the oversized-prompt probe through an assistant produces fallback or the defined error, never silent truncation.
- No reasoning text appears in delivered content in tests covering captured qwen3 response shapes.

**Status:** `todo`

---

## Phase 4 — Structured output

**Goal.** A game or mod asks for JSON matching a schema and receives valid JSON or a clear error.

**Deliverables**

- Support `response_format` (`json_object` and `json_schema`) on the assistant path: pass through where supported, and validate at the gateway regardless.
- Bounded repair loop: on invalid output, retry with the validation error, then move down the chain, then return a distinct error.
- Buffer schema-constrained responses, so a stream only delivers validated output.
- A minimal example client showing a game NPC turn with no retry logic of its own.

**Acceptance**

- Fake-upstream tests cover valid, repairable, and permanently invalid output.
- In the Phase 5 eval, game prompts return schema-valid JSON on every successful response.

**Status:** `todo`

---

## Phase 5 — Outcome-based eval

**Goal.** Decisions rest on results across the use cases this product exists for.

**Deliverables**

- Every assistant runs the same prompts within a track, with at least three repeats and pass rates reported.
- Record retrieved chunks and scores, tool calls, attempts, and the upstream that answered, separating retrieval quality from model quality.
- Add deterministic checks wherever possible; agent grading is the exception.
- New tracks:
  - **Coding:** small repo tasks graded by real tests.
  - **Game:** schema validity, latency, and in-character quality.
  - **Routing:** bounded local tasks and harder tasks on mixed chains using Groq and Gemini free tiers, tracking quota consumed.
- Keep the Porcelain knowledge track, with expected answers checked against the feature records.

**Acceptance**

- A published baseline report with a no-harness control per track.
- Rerunning produces pass rates stable enough to compare, with the noise level stated in the report.

**Status:** `todo`

---

## Phase 6 — Module decisions

**Goal.** Each module earns its complexity or leaves.

**Deliverables**

- Keep, rework, or remove each of retrieval, intent, evaluator, escalation, tool router, tool executor, and MCP. Default rule: a module must beat the control on its target track by more than measured noise, at acceptable latency.
- Evaluate the tool loop's model-compensation code (`tool_call_shorthand.go`, `tool_loop_finalize.go`) separately: rewriting a model's final answer is only justified if it improves outcomes, and models with reliable native tool calling may not need it.
- Update feature records, then decide which paused plans resume, and archive the rest as `superseded`.

**Acceptance**

- Each module has a recorded decision that cites eval results.
- Removed modules have a migration path for stored settings.

**Status:** `todo`

---

## Open questions

1. **First game client:** which game or mod library should the Phase 4 example target?
2. **Router tiers:** do hard tasks escalate by explicit assistant choice, by rules, or both? Decide after Phase 5 routing results.
3. **Over-limit prompts:** should clients be able to opt into gateway-side truncation or summarizing for context that fits no model?

---

## References

- Code: `chimera/chimera-gateway/internal/chat/chat.go`, `internal/harness/`, `chimera/internal/providerlimits/`
- Config: `config/provider-model-limits.yaml`, `config/chimera-broker.config.json`
- Eval: `harness-eval/ASSISTANTS_MATRIX.md`, `harness-eval/PORCELAIN_MATRIX_RUN_20261001.md`, `harness-eval/runs/20261001T003156Z/AGENT_REPORT.md`
