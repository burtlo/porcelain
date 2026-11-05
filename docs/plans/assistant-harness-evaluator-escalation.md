# Plan: Virtual model harness — evaluator and escalation v1

| Field | Value |
|-------|-------|
| **Doc kind** | `feature-plan` |
| **Owners / areas** | Gateway harness, chat streaming, broker upstream |
| **Status** | `done` |
| **Targets** | Gateway v0.4 |
| **Last updated** | See git history |
| **Supersedes / superseded by** | Child of [`assistant-turn-harness.md`](assistant-turn-harness.md) |
| **As-built** | [`operator-assistants.md`](../features/operator-assistants.md), [`gateway-chat-routing-pipeline.md`](../features/gateway-chat-routing-pipeline.md) |

## At a glance

Optional single-pass evaluator checks primary output and feeds **escalation v1** (re-retrieve, fallback chain). Streaming behavior is **flexible per VM** via `stream_policy`.

| Phase | Outcome | Status |
|-------|---------|--------|
| [Phase 1 — Evaluator (single pass)](#phase-1--evaluator-single-pass) | Confidence check after primary; configurable stream policy | `done` |
| [Phase 2 — Escalation v1](#phase-2--escalation-v1) | Re-retrieve and fallback-chain retry before giving up | `done` |

**Depends on:** [runtime](assistant-harness-runtime.md), [settings](assistant-harness-settings.md), [retrieval](assistant-harness-retrieval.md), [intent](assistant-harness-intent.md)  
**Blocks:** [observability](assistant-harness-observability.md) (evaluator fields in envelope UI)

**Delivery gates:** [umbrella](assistant-turn-harness.md#delivery-gates-normative) — `make precommit` at plan done; hard cut (no legacy paths); phase debt OK if Fix-by is set.

---

## Background

The gateway today passthrough-streams upstream SSE immediately ([`internal/chat/chat.go`](../../chimera/chimera-gateway/internal/chat/chat.go)). An evaluator needs the full primary text before scoring. **Normative decision:** streaming is flexible — operators choose `stream_policy` per VM evaluator config.

**Related docs:** [`assistant-turn-harness.md`](assistant-turn-harness.md), [`version-v0.4.md`](../version-v0.4.md).

---

## Phase 1 — Evaluator (single pass)

**Goal.** Optional small-model pass checks primary output and sets confidence signals for escalation.

**Deliverables**

- Evaluator module config: `enabled`, `model_id`, `mode: single_pass`, thresholds (`min_confidence`, `hallucination_risk_max`), **`stream_policy`**.
- Evaluator stage runs after primary (and internal tool rounds); writes `TurnEnvelope.evaluation`.
- JSON schema for evaluator response: `{ confidence, issues[], recommend_escalation }`.
- **`stream_policy` values (normative):**

| Value | Behavior |
|-------|----------|
| `immediate` | Stream primary SSE to client as today; evaluator runs after stream completes; results in envelope only; escalation cannot replace in-flight answer on same turn |
| `gate_on_evaluator` | Buffer primary output; run evaluator; if pass, stream buffered content (or deliver as single chunk); if fail and escalation recommends action, run escalation before client delivery |
| `buffer_until_complete` | No client streaming when evaluator enabled; JSON (or single-chunk) response after eval completes |

- When VM `stream_policy` conflicts with `body.stream: true`, gateway applies VM policy; log `harness.stream.policy_applied`.
- Default for new VMs: `immediate`.
- Evaluator failure fail-opens (deliver primary answer, log warning).
- **Gallery** ([umbrella contract](assistant-turn-harness.md#gallery--reference-ui-contract-normative)): evaluator section fixtures — module off; `single_pass` with each `stream_policy` (`immediate`, `gate_on_evaluator`, `buffer_until_complete`); part slugs for mode and stream policy controls.

**Acceptance**

- VM with evaluator on → envelope shows `evaluation.ran: true` and confidence on success path.
- Same VM with three stream policies → verifiable different client-visible timing in test fixtures.
- Evaluator error under any policy → primary answer delivered, warning logged.
- Gallery shows evaluator off plus all three stream-policy configurations without live VM edits.

**Status:** `done`

---

## Phase 2 — Escalation v1

**Goal.** When evaluator or policy recommends escalation, harness re-retrieves or walks fallback chain before giving up.

**Deliverables**

- Escalation module config: `max_rounds`, `on_fail` actions (`re_retrieve`, `fallback_chain`, `ensemble` placeholder, `human` placeholder).
- Implement `re_retrieve` (increase `top_k` or lower threshold within caps) and `fallback_chain` (existing loop, harness-aware).
- Increment `TurnEnvelope.escalation.rounds`; respect `max_escalation_rounds` and `max_upstream_calls_per_turn`.
- Log `harness.escalation.*` slugs with action and outcome.
- Escalation with `gate_on_evaluator` may force buffer path even when client requested stream.
- **Gallery:** escalation config sample (budget + `on_fail` actions visible) on the VM harness fixture; complements Phase 1 stream-policy demos.

**Acceptance**

- Forced low-confidence fixture triggers re-retrieve or next chain entry; client still receives one final answer.
- Escalation budget exhaustion returns best-effort answer with metadata flag.
- Gallery shows an escalation-enabled config state (budget / actions) beside evaluator demos.

**Status:** `done`

---

## References

- Code: `internal/chat/chat.go`, `internal/server/virtualmodel_chat.go`
- Advanced modes: [`assistant-harness-advanced-modules.md`](assistant-harness-advanced-modules.md)
- Umbrella: [`assistant-turn-harness.md`](assistant-turn-harness.md)
