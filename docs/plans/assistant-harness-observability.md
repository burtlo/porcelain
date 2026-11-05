# Plan: Virtual model harness — observability

| Field | Value |
|-------|-------|
| **Doc kind** | `feature-plan` |
| **Owners / areas** | Operator embed UI (settings logs, chat), log message registry |
| **Status** | `done` |
| **Targets** | Gateway v0.4 |
| **Last updated** | See git history |
| **Supersedes / superseded by** | Child of [`assistant-turn-harness.md`](assistant-turn-harness.md) |
| **As-built** | [Operator conversation history](../features/operator-conversation-history.md), [Operator chat UI](../features/operator-chat-ui.md), [Operator log message registry](../features/operator-log-message-registry.md) |

## At a glance

Operators see per-turn harness stages in settings conversation cards and collapsible **Turn details** in chat — without reading raw gateway logs.

| Phase | Outcome | Status |
|-------|---------|--------|
| [Phase 1 — Harness observability in conversation views](#phase-1--harness-observability-in-conversation-views) | Stage timeline in settings and chat from envelope summary | `done` |

**Depends on:** [runtime](assistant-harness-runtime.md) Phase 2 (`harness_summary_json`), stage slugs from shipped harness modules  
**Can parallel:** log registry entries once slug names are stable

**Delivery gates:** [umbrella](assistant-turn-harness.md#delivery-gates-normative) — `make precommit` at plan done; hard cut (no legacy paths); phase debt OK if Fix-by is set.

---

## Background

Conversation cards already group routing and tool-router lines ([`log-conversations.md`](archive/log-conversations.md)). This plan extends derive models and chat UI to surface harness stage timelines from persisted envelope summaries and structured logs.

**Related docs:** [`assistant-turn-harness.md`](assistant-turn-harness.md), [`operator-log-message-registry.md`](../features/operator-log-message-registry.md).

---

## Phase 1 — Harness observability in conversation views

**Goal.** Operators see per-turn harness stages in settings conversation cards and collapsible details in chat.

**Deliverables**

- Register `harness.stage.*` and `harness.escalation.*` in [`operator-log-message-registry.md`](../features/operator-log-message-registry.md).
- Extend `conversationCardModel` derive: group harness lines under `turn_index`; stage pills in expanded timeline.
- Chat UI: collapsible **Turn details** on assistant messages (intent chips, RAG count, resolved model, evaluator verdict, compress strategy); load from `harness_summary_json` on history open.
- VM scoped log panel: filter harness events by `virtual_model_id`.
- **Gallery** ([umbrella contract](assistant-turn-harness.md#gallery--reference-ui-contract-normative)): (1) conversation-card fixture with harness stage timeline / pills expanded; (2) chat Turn details collapsed vs expanded mounts (settings gallery subsection or chat gallery using production chat render helpers); fixture `harness_summary_json` covering retrieval + evaluator + escalation fields.

**Acceptance**

- One chat turn with retrieval + evaluator shows ordered stages in settings conversation expanded view.
- Reopened history thread shows turn details without replaying raw gateway logs.
- Gallery shows harness timeline and Turn details states; reviewers can inspect chips/timeline without running chat.

**Status:** `done`

---

## References

- Code: `embed/embedui/settings/derive/conversationCardModel.js`, `embed/embedui/chat/`
- Umbrella: [`assistant-turn-harness.md`](assistant-turn-harness.md)
