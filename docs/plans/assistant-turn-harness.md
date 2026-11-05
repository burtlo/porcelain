# Plan: Virtual model turn harness (index)

| Field | Value |
|-------|-------|
| **Doc kind** | `feature-plan` |
| **Owners / areas** | Gateway runtime, operator SQLite, chat path, operator embed UI (settings logs, chat), RAG, workspaces |
| **Status** | `done` |
| **Targets** | Gateway v0.4 |
| **Last updated** | See git history |
| **Supersedes / superseded by** | Supersedes ensemble-first framing in prior [`version-v0.4.md`](../version-v0.4.md); ensemble and human escalation become harness modules |
| **As-built** | [Operator virtual models](../features/operator-assistants.md), [Gateway chat routing pipeline](../features/gateway-chat-routing-pipeline.md), [Indexer workspaces](../features/indexer-workspaces.md), [Operator conversation history](../features/operator-conversation-history.md), [Operator chat UI](../features/operator-chat-ui.md) |

## At a glance

Operators pick one **named virtual model** in chat (single-model picker UX — like Cursor's one visible model choice, even when the gateway uses multiple upstream models internally) and the gateway runs a full **turn harness** before answering: classify intent, plan retrieval and tools, run the primary model, optionally evaluate and escalate, then return one reply. Each virtual model toggles harness modules on or off and holds per-module settings. The client sends a single chat request per user turn with an **explicit** `body.model` virtual model id; all orchestration stays inside the gateway.

| Child plan | Outcome | Status |
|------------|---------|--------|
| [Runtime + envelope](assistant-harness-runtime.md) | Stage registry; turn envelope schema and persistence | `done` |
| [VM harness settings](assistant-harness-settings.md) | Module toggles per virtual model in settings | `done` |
| [Per-VM retrieval](assistant-harness-retrieval.md) | VM-scoped retrieval + pluggable evidence compression | `done` |
| [Workspace policy](assistant-harness-workspace-policy.md) | Sensitivity, cloud rules, file permissions; meta-policy stage | `done` |
| [Intent classification](assistant-harness-intent.md) | Heuristic intent + optional LLM; evaluate API | `done` |
| [Evaluator + escalation v1](assistant-harness-evaluator-escalation.md) | Single-pass evaluator; re-retrieve and fallback escalation | `done` |
| [Gateway workspace tools](assistant-harness-workspace-tools.md) | Read/write files under workspace permission in one turn | `done` |
| [Observability](assistant-harness-observability.md) | Stage timeline in settings logs and chat turn details | `done` |
| [Advanced modules](assistant-harness-advanced-modules.md) | Multi-draft evaluator; human escalation target | `done` |

**Suggested execution order:** Runtime → Settings (UI after runtime Phase 1) → Retrieval + Workspace policy (parallel after envelope) → Intent → Evaluator/Escalation → Tools → Observability (after envelope + slugs exist) → Advanced modules last.

---

## Gallery / reference UI contract (normative)

Operators and implementers must be able to **review harness UI states on `/ui/settings/gallery`** without configuring live SQLite or chat. Live `/ui/settings` and `/ui/chat` remain the product path; the gallery is the **configuration-state catalog** for visual and usability review.

**Rules for every child plan that ships operator-visible UI**

1. **Same render path as production** — Gallery fixtures call the same `render/cards/*` (and chat modules when applicable) as the live feed via `embedui/gallery/gallery-card-fixtures.js` (or a sibling fixture module). Do not hand-copy markup that can drift.
2. **Multiple configuration states** — Ship at least two fixture variants that matter for the feature (e.g. module off vs on; default vs dense config; policy deny vs allow). Prefer side-by-side mounts under one gallery subsection so states are comparable without toggling live data.
3. **Navigable section** — Add a stable `id` anchor and top-nav link on `settings/gallery.html` (or extend an existing section) so reviewers can jump feature-by-feature.
4. **Part slugs** — New controls get `data-ui-part` entries and a `styleguide-part-slugs` list under the demo (see `settings/card-parts-registry.md`).
5. **Done means gallery too** — A UI-bearing phase is not `done` until gallery fixtures and nav land in the same delivery train as the live controls. Backend-only phases (runtime envelope with no new controls) may skip gallery; note that explicitly in the child plan.
6. **Chat surfaces** — When a plan adds chat UI (e.g. Turn details), either extend the settings gallery with a chat-message fixture mount or add a dedicated chat gallery subsection that reuses production chat render helpers — same multi-state rule.

**Fixture inventory (expected by child plan)**

| Child plan | Gallery demos (minimum) |
|------------|-------------------------|
| [Runtime](assistant-harness-runtime.md) | None (no new operator controls) |
| [Settings](assistant-harness-settings.md) | VM card: harness section default/off; all-modules-on stub; disabled-until-ready controls |
| [Retrieval](assistant-harness-retrieval.md) | Harness retrieval config: truncate vs summarize; skip-if / empty state |
| [Workspace policy](assistant-harness-workspace-policy.md) | Workspace card: public+cloud vs private+no-cloud; file policy `none` / `read` / `read_write` |
| [Intent](assistant-harness-intent.md) | Intent module off vs heuristic vs LLM; evaluate dry-run panel sample |
| [Evaluator + escalation](assistant-harness-evaluator-escalation.md) | Evaluator off; `single_pass` + each `stream_policy`; escalation budget sample |
| [Workspace tools](assistant-harness-workspace-tools.md) | Tool executor off vs on; read-only vs read_write permission framing |
| [Observability](assistant-harness-observability.md) | Conversation card harness timeline; chat Turn details expanded/collapsed |
| [Advanced modules](assistant-harness-advanced-modules.md) | `multi_draft` config; human escalation copy-paste / paste-back sample |

**Code anchors:** `embedui/settings/gallery.html`, `embedui/gallery/gallery-card-fixtures.js`, `embedui/gallery/README.md`, `embedui/settings/card-parts-registry.md`.

---

## Delivery gates (normative)

Applies to every `assistant-harness-*` child plan.

1. **`make precommit`** — Required before marking a child plan `done` / shipped. Runs `contracts-generate`, `fmt`, `fmt-check`, `vet`, and `test` (see root `Makefile`). Phase check-ins may skip it; plan delivery may not.
2. **Fix timing** — Incomplete UI or temporary stubs are allowed **within** a multi-phase plan when a later phase of that plan (or an explicitly named later child plan) will finish them; log in [`backlog/harness-ui-design-debt.md`](backlog/harness-ui-design-debt.md) with a **Fix by** column. At plan delivery, clear debt owned by that plan or reassign with a real owner plan.
3. **Hard cut / no legacy** — Do not retain dual-read endpoints, config keys, env aliases, or “compat for existing installs.” There are **no existing users** of this service for migration purposes: update all call sites to the new path and delete obsolete ones in the same train.
4. **Design validation** — UI-bearing phases: run the [harness-ui-design-validator](../../.cursor/skills/harness-ui-design-validator/SKILL.md) skill (separate agent). Plan delivery: run [harness-plan-delivery](../../.cursor/skills/harness-plan-delivery/SKILL.md) (includes `make precommit`). See also [`.cursor/skills/README.md`](../../.cursor/skills/README.md).

---

## Background

Virtual models today attach a **routing stack**: fallback chain, routing-policy rules, and optional tool router ([`operator-assistants.md`](../features/operator-assistants.md)). Chat requests pass through a fixed five-step pipeline ([`gateway-chat-routing-pipeline.md`](../features/gateway-chat-routing-pipeline.md)) — tool slimming, global RAG inject, initial upstream pick, fallback loop — then proxy once to the broker. That delivered an early working product but not the orchestration depth described in [`design.md`](../design.md).

**v0.4** reframes the gateway as a **turn harness**: a composable, per-virtual-model workflow that can add LLM-assisted stages when enabled, while **meta-policy** (privacy, cost, eligibility) stays deterministic and runs before any classifier sees full context. The client contract stays minimal: one `model` string on `POST /v1/chat/completions`, one HTTP round-trip per user message. The gateway may call upstream many times internally (classifier, retrieval, primary, evaluator, tool rounds).

Prior v0.4 ideas — two-phase ensemble, `//deep` triggers, external human escalation as standalone pillars — are **modules inside this harness**, not separate product surfaces. MCP-backed tool execution remains **v0.5**; v0.4 defines the tool-invocation contract and ships **gateway-native workspace tools** first.

**Guiding principles (normative for v0.4)**

1. **Virtual model = harness profile** — module toggles and per-module config live on the VM row (operator SQLite).
2. **Single client request per user turn** — internal multi-stage loop; bounded by `max_tool_rounds`, `max_escalation_rounds`, `max_upstream_calls_per_turn`, and wall-clock timeout.
3. **Start simple, iterate per stage** — ship a thin vertical slice (existing pipeline as stages + per-VM RAG + optional single evaluator) before deepening classifiers and ensemble modes.
4. **Observability is first-class** — every stage emits `harness.stage.*` slugs with `virtual_model_id`, `conversation_id`, `request_id`, `turn_index`.
5. **No Continue / IDE agent dependency** — Chimera chat owns orchestration; file actions run in-gateway under workspace policy, not via external agent clients.
6. **Direct upstream ids** remain an escape hatch (`provider/model` bypasses harness); the product path is virtual model ids.
7. **Explicit virtual model id** — no reserved aliases; `body.model` must match a VM `model_id` or a direct upstream id. The product shows a **single-model picker** even when the harness uses multiple upstream models via fallback, evaluator, and escalation.
8. **Hard cut** — no legacy endpoint/config/path dual-support; assume greenfield operators. Plan delivery requires `make precommit`.

**Related docs:** [`version-v0.4.md`](../version-v0.4.md), [`design.md`](../design.md), [`gateway-chat-routing-pipeline.md`](../features/gateway-chat-routing-pipeline.md), [`operator-assistants.md`](../features/operator-assistants.md), [`gateway-rag-ingest-and-retrieval.md`](../features/gateway-rag-ingest-and-retrieval.md), [`indexer-workspaces.md`](../features/indexer-workspaces.md), [`operator-conversation-history.md`](../features/operator-conversation-history.md), [`log-conversations.md`](archive/log-conversations.md), [`virtual-models-operator.md`](archive/virtual-models-operator.md).

---

## Harness modules (reference)

Modules attach to a virtual model. **Meta-policy** and **primary** are always on; others are toggleable.

| Module | Responsibility | Child plan |
|--------|----------------|------------|
| `meta_policy` | Deterministic privacy, cloud eligibility, file-action gate from workspace policy | [workspace policy](assistant-harness-workspace-policy.md) |
| `intent_classifier` | Populate `intent` on turn envelope (heuristic → optional LLM) | [intent](assistant-harness-intent.md) |
| `resource_planner` | Map intent to initial chain entry / model tier | [intent](assistant-harness-intent.md) (heuristic stub) |
| `retrieval` | Per-VM vector search, thresholds, compression | [retrieval](assistant-harness-retrieval.md) |
| `tool_router` | Slim client tool declarations (existing) | [runtime](assistant-harness-runtime.md) (parity) |
| `tool_executor` | Run tools inside gateway turn (`ToolExecutor` interface) | [workspace tools](assistant-harness-workspace-tools.md) |
| `evidence_aggregate` | Merge RAG hits, tool outputs, prior context into structured block | [retrieval](assistant-harness-retrieval.md) |
| `primary` | Main upstream completion + internal tool round loop | [runtime](assistant-harness-runtime.md) (parity) |
| `evaluator` | Correctness / confidence check; `single_pass` then `multi_draft` | [evaluator](assistant-harness-evaluator-escalation.md), [advanced](assistant-harness-advanced-modules.md) |
| `escalation` | Re-retrieve, chain walk, ensemble trigger, human target | [evaluator](assistant-harness-evaluator-escalation.md), [advanced](assistant-harness-advanced-modules.md) |
| `response_builder` | Citations, formatting, response metadata headers | [observability](assistant-harness-observability.md) |
| `telemetry` | Turn envelope persistence, learning hooks (log-only in v0.4) | [runtime](assistant-harness-runtime.md), [observability](assistant-harness-observability.md) |

Module toggles and per-module config surface in [VM harness settings](assistant-harness-settings.md).

---

## Turn envelope schema (normative sketch)

All stages read and write a shared **`TurnEnvelope`** (JSON). Unused fields use `null`, `"unknown"`, or empty arrays until populated.

`scope.workspace_id` is **derived** from project/flavor workspace lookup (see [workspace policy plan](assistant-harness-workspace-policy.md)), not from a client header.

```json
{
  "schema_version": 1,
  "request_id": "…",
  "conversation_id": "…",
  "turn_index": 1,
  "virtual_model_id": "Research-1.0",
  "scope": {
    "tenant_id": "…",
    "project_id": "…",
    "flavor_id": "…",
    "workspace_id": 42,
    "workspace_permission": "read_write_files",
    "sensitivity": "internal",
    "allow_cloud": true
  },
  "intent": {
    "task_type": "unknown",
    "domain": "unknown",
    "complexity": "unknown",
    "ambiguity": "unknown",
    "sensitivity": "unknown",
    "requires_rag": null,
    "requires_tools": null,
    "tags": []
  },
  "plan": {
    "stages": [],
    "primary_model_id": null,
    "evaluator_mode": null,
    "escalation_budget": 0
  },
  "retrieval": {
    "ran": false,
    "top_k": null,
    "hits_count": 0,
    "evidence_ids": [],
    "compress_strategy": null
  },
  "execution": {
    "tool_calls": [],
    "upstream_attempts": [],
    "resolved_model_id": null
  },
  "evaluation": {
    "ran": false,
    "confidence": null,
    "issues": [],
    "recommend_escalation": false
  },
  "escalation": {
    "rounds": 0,
    "last_action": null
  },
  "response": {
    "citations": [],
    "metadata": {}
  }
}
```

Field names and redaction rules are locked in [runtime plan](assistant-harness-runtime.md) Phase 2.

---

## Resolved decisions

1. **Streaming** — Flexible `stream_policy` per VM evaluator config (`immediate` \| `gate_on_evaluator` \| `buffer_until_complete`). See [evaluator plan](assistant-harness-evaluator-escalation.md).
2. **Workspace scope wire format** — `X-Chimera-Project` + `X-Chimera-Flavor-Id` only; gateway derives workspace row via DB lookup with lowest-id tie-break. `X-Chimera-Workspace-Id` is conversation-history metadata only. See [workspace policy plan](assistant-harness-workspace-policy.md).
3. **Tool declarations** — Gateway-injected only when `tool_executor` enabled; client `tools` ignored on VM harness path. See [workspace tools plan](assistant-harness-workspace-tools.md).
4. **Evidence compression** — Pluggable `EvidenceCompressor`; ship `summarize` in v0.4 with fail-open to `truncate`. See [retrieval plan](assistant-harness-retrieval.md).
5. **Model selection** — Explicit VM id only; no `auto` alias or synthetic catalog id. Single-model picker UX; internal multi-upstream orchestration is invisible to the client.

---

## Bounded loop limits (normative defaults)

Document in gateway config and VM harness profile:

| Limit | Default | Purpose |
|-------|---------|---------|
| `max_tool_rounds` | 5 | Internal tool loop per turn |
| `max_escalation_rounds` | 2 | Re-retrieve / re-plan cycles |
| `max_upstream_calls_per_turn` | 15 | Broker call budget |
| `harness_wall_clock_timeout` | 120s | Per client request |

---

## References

- Child plans: `assistant-harness-*.md` (this directory)
- Orchestration today: `chimera/chimera-gateway/internal/server/virtualmodel_chat.go`, `internal/chat/chat.go`
- Virtual models: `internal/virtualmodel/`, `internal/operatorstore/`
- RAG: `internal/rag/`
- Conversation logs: `docs/plans/log-conversations.md`, `embed/embedui/settings/derive/conversationCardModel.js`
- Chat UI: `embed/embedui/chat/`
- North-star: [`design.md`](../design.md)
- Version roadmap: [`version-v0.4.md`](../version-v0.4.md)
