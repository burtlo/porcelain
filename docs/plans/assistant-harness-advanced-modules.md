# Plan: Virtual model harness — advanced modules

| Field | Value |
|-------|-------|
| **Doc kind** | `feature-plan` |
| **Owners / areas** | Gateway harness, evaluator, escalation, chat UI |
| **Status** | `done` |
| **Targets** | Gateway v0.4 (same train; after MVP phases 1–10) |
| **Last updated** | See git history |
| **Supersedes / superseded by** | Child of [`assistant-turn-harness.md`](assistant-turn-harness.md) |
| **As-built** | [`operator-assistants.md`](../features/operator-assistants.md), [`gateway-chat-routing-pipeline.md`](../features/gateway-chat-routing-pipeline.md), [`operator-chat-ui.md`](../features/operator-chat-ui.md) |

## At a glance

**Multi-draft evaluator** (ensemble as an evaluator mode) and **human escalation** as an escalation target when in-loop options exhaust. Ships after harness MVP (runtime through observability).

| Phase | Outcome | Status |
|-------|---------|--------|
| [Phase 1 — Evaluator multi-draft mode](#phase-1--evaluator-multi-draft-mode) | Parallel drafts, synthesize, judge — one client answer | `done` |
| [Phase 2 — Human escalation module](#phase-2--human-escalation-module) | Copy-paste external escalation with paste-back merge | `done` |

**Depends on:** [evaluator + escalation v1](assistant-harness-evaluator-escalation.md)

**Delivery gates:** [umbrella](assistant-turn-harness.md#delivery-gates-normative) — `make precommit` at plan done; hard cut (no legacy paths); phase debt OK if Fix-by is set.

---

## Background

Prior v0.4 drafts treated two-phase ensemble and external human escalation as standalone features. They are now modules inside the harness ([`assistant-turn-harness.md`](assistant-turn-harness.md)). Streaming policies from the evaluator plan extend to multi-draft phases (buffer until synthesize completes unless VM `stream_policy` is `immediate` for telemetry-only eval).

**Related docs:** [`version-v0.4.md`](../version-v0.4.md), [`design.md`](../design.md).

---

## Phase 1 — Evaluator multi-draft mode

**Goal.** Ensemble becomes an evaluator implementation: parallel drafts, synthesize, then judge — unified config surface.

**Deliverables**

- Evaluator `mode: multi_draft`: config `draft_count` (default 3, cap by catalog availability), `synthesize_model_id`.
- Gateway orchestrates N parallel draft completions via broker, then synthesize pass, then evaluator JSON.
- VM-level triggers (complexity threshold, manual depth flag in message or header) — replace legacy `//deep` on fixed semver id.
- Streaming semantics documented: draft-phase failure, synthesize failure, interaction with `stream_policy`.
- Reuse escalation module when evaluator still recommends escalation after multi-draft.
- **Gallery** ([umbrella contract](assistant-turn-harness.md#gallery--reference-ui-contract-normative)): evaluator `multi_draft` config fixture (`draft_count`, synthesize model) beside existing single-pass demos.

**Acceptance**

- VM with `multi_draft` enabled completes draft → synthesize → evaluate → one client-visible answer.
- Logs show draft count, models used, phase boundaries.
- Gallery shows `multi_draft` configuration state distinct from `single_pass`.

**Status:** `done`

---

## Phase 2 — Human escalation module

**Goal.** External human escalation is one escalation target when in-loop options exhaust.

**Deliverables**

- Escalation `on_fail: human` config: named surfaces (name + URL), privacy disclosure text, paste-back delimiter.
- Response construction stage emits escalation body with copy-paste prompt when triggered.
- Later message with delimiter merges external answer into thread (conversation merge or harness entry hook).
- Non-blocking: no delimiter on next message → normal chat continues.
- **Gallery:** human-escalation sample — escalation message with privacy disclosure + paste-back delimiter framing (chat or conversation fixture); config surface for named human targets on the VM harness card.

**Acceptance**

- Documented path: low confidence → escalation message with privacy line → paste-back → merged continuation.
- Human escalation never runs before internal escalation budget exhausted (configurable).
- Gallery shows human-escalation config and a sample escalation / paste-back message state.

**Status:** `done`

---

## References

- Code: `internal/harness/`, `internal/chat/chat.go`
- Evaluator v1: [`assistant-harness-evaluator-escalation.md`](assistant-harness-evaluator-escalation.md)
- Umbrella: [`assistant-turn-harness.md`](assistant-turn-harness.md)
