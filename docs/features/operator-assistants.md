# Feature: Operator assistants

| Field | Value |
|-------|-------|
| **Doc kind** | `feature-record` |
| **Areas** | Gateway runtime, operator SQLite, chat routing, settings UI |
| **Status** | `current` |
| **Introduced** | Gateway minor after unified operator cards baseline (formerly “virtual models”) |
| **Originated from** | [`plans/archive/virtual-models-operator.md`](../plans/archive/virtual-models-operator.md), [`plans/assistant-turn-harness.md`](../plans/assistant-turn-harness.md) |
| **Related features** | [Operator settings UI](operator-settings-ui.md), [Operator provider model availability](operator-provider-model-availability.md), [Operator chat UI](operator-chat-ui.md), [Context window admission](context-window-admission.md), [Gateway chat routing pipeline](gateway-chat-routing-pipeline.md) |
| **Depends on** | Operator SQLite, broker catalog, routing policy engine, UI session auth |
| **Last updated** | See git history |

## At a glance

Operators create **assistants** in operator SQLite—each with a client-facing `model_id` (name + version), description, enable flag, and visibility—and attach a **routing stack**: required ordered fallback chain, optional routing-policy rules, and optional tool-router block. The gateway loads enabled assistants into an in-memory registry, exposes them on `GET /v1/models`, and resolves `POST /v1/chat/completions` per assistant id. Fresh installs start with **zero** assistants (only a routing-rule definition catalog is seeded); operators create assistants in settings. Settings cards support full CRUD, generate-from-catalog, dry-run evaluate, and scoped routing-decision logs carrying `assistant_id`.

## Operator-visible behavior

- **Assistant cards** on `/ui/settings` — list, create, edit metadata, enable/disable, delete.
- **Routing stack editors** — Fallback chain (required), routing policy YAML (toggleable), tool router models + confidence (toggleable). These configure the [chat routing pipeline](gateway-chat-routing-pipeline.md); cards do not execute routing themselves.
- **Harness modules** — Per-assistant toggles for retrieval, intent, evaluator, escalation, and workspace tools (`GET/PUT /api/ui/assistants/{id}/harness`). Workspace tools inject gateway-native file/search tools plus optional MCP sidecars when `tool_executor` is on ([Gateway MCP tool backends](gateway-mcp-tool-backends.md)). The harness card shows MCP server bind toggles and per-tool allowlists when operator SQLite defines servers; catalog rebuilds on the next chat turn after save. `max_tool_rounds` defaults to 5 and workspace policy decides whether reads or writes are permitted. Evaluator supports `single_pass` and `multi_draft`; multi-draft uses `draft_count` (default 3, capped at 8 and available fallback models) plus `synthesize_model_id`, buffers the response through synthesis and JSON evaluation, and returns one answer. Escalation accepts `{ "max_rounds":0..2, "on_fail":["re_retrieve","fallback_chain","ensemble","human"], "human_surfaces":[{"name":"…","url":"…"}], "privacy_disclosure":"…", "paste_back_delimiter":"…" }`. The human target is emitted only after internal remediation exhausts; a next user message containing the delimiter supplies external context without blocking ordinary turns. Gallery fixtures show single-pass, multi-draft, and human escalation states.
- **Generate from catalog** — Builds fallback or policy from **available general text/chat** upstream models only. Embedding, audio/TTS/transcription, image-generation, safety/guard, reranker, robotics, and capability-unknown endpoints are excluded or skipped with per-model reasons. Existing fallback order is treated as operator preference; remaining models use deterministic local-first then lexical order instead of parameter-size name scoring.
- **Evaluate / preview** — Dry-run policy against sample message text without sending chat.
- **Harness evaluate** — `POST /api/ui/assistants/{id}/harness/evaluate` runs stack resolution, meta-policy, and intent against a sample message, returning a redacted envelope without retrieval injection or an upstream primary completion.
- **Scoped logs** — Card expanded panel shows routing, fallback, and tool-router events for that assistant id.
- **Chat selector** — Enabled public assistants appear in `/ui/chat` model dropdown alongside upstream ids.
- **Disabled / private** — Disabled assistants hidden from catalog and rejected on chat; private assistants visible only to creating principal (single-user desktop uses empty tenant today).

## System behavior and contracts

**Invariants**

- Assistants persist in **operator SQLite** (`assistants` and attachment tables); not in `chimera.yaml` for new config.
- Client protocol unchanged: callers send one `model` string on chat/completions (wire details: [`configuration.md`](../configuration.md#chat-completions-client-contract)).
- Each assistant compiles routing policy into `routing.InMemoryPolicy` at registry reload.
- Fallback walk skips unavailable models (provider availability), quota/context limits, and retriable upstream errors. Generated chains are capability-filtered before save; manually edited chains remain operator-controlled.
- Structured logs include `assistant_id` on routing resolution and fallback attempts (legacy log fields may still carry `virtual_model_id` during transition).
- RAG remains **gateway-global** for enablement (`search.enabled`); the per-assistant **retrieval** harness module can disable injection or override retrieval knobs for that turn. Empty retrieval config inherits gateway RAG top-k and score floor.
- Evaluator failures fail open to the primary response. `immediate` preserves live SSE; gated policies buffer the primary, evaluate it, and can escalate before delivering one response.

**Decisions**

| Topic | Decision |
|-------|----------|
| Model id format | `{Name}-{Version}` stored unique per tenant |
| Bootstrap | Seed routing-rule catalog only; **no** YAML import into SQLite |
| Fallback | Required non-empty chain; generate uses capability-filtered available catalog, preserves explicit preference, and defaults unranked models local-first |
| Routing rules | Shared policy YAML body per assistant; first matching `when.min_message_chars` wins |
| Tool router | Optional; `router_models[]`, `confidence_threshold`, enable flag |
| Reload | Registry refresh after CRUD via assistant registry reload |
| Direct upstream | Clients may send any upstream `provider/model` id when not using an assistant |

**Identity / auth / scoping**

- Rows scoped by `tenant_id` (empty string default desktop).
- `created_by_principal_id` tracks owner for `private` visibility.
- Chat resolves assistant by exact `model_id` string match.

**Persistence**

- Migrations under `migrations/chimera-gateway/operator/` (assistant tables; rename migration from legacy `virtual_models`).
- Attachments: fallback chain JSON, routing policy YAML blob, tool-router fields on assistant row.

## Interfaces

| Surface | Detail |
|---------|--------|
| `GET /api/ui/assistants` | List summaries |
| `POST /api/ui/assistants` | Create |
| `GET /api/ui/assistants/{id}` | Detail + `fallback_unavailable` hints |
| `PUT /api/ui/assistants/{id}` | Update metadata |
| `DELETE /api/ui/assistants/{id}` | Delete |
| `PUT /api/ui/assistants/{id}/fallback` | Save fallback chain |
| `PUT /api/ui/assistants/{id}/routing-policy` | Save policy YAML + enable flag |
| `PUT /api/ui/assistants/{id}/tool-router` | Save tool router config |
| `GET /api/ui/assistants/{id}/harness` | List harness modules + UI hints |
| `PUT /api/ui/assistants/{id}/harness` | Save harness module toggles / config |
| `GET/PUT /api/ui/mcp/servers` | List or upsert MCP server rows (SQLite); YAML applies until first server row |
| `GET /api/ui/mcp/servers/{id}/tools` | Live `tools/list` for health UI (manager required) |
| `GET/PUT /api/ui/assistants/{id}/mcp` | Per-assistant MCP server bind + per-tool enable flags |
| `POST /api/ui/assistants/{id}/routing/generate` | Generate stack from catalog; accepts optional `preferred_models` and `local_first`, returns `model_decisions` |
| `POST /api/ui/assistants/{id}/routing/evaluate` | Dry-run policy |
| `POST /api/ui/assistants/{id}/harness/evaluate` | Dry-run pre-primary harness stages |
| `GET /v1/models` | Includes enabled assistants |
| `POST /v1/chat/completions` | Resolves `body.model` through assistant registry |
| Log slugs | `chat.routing.resolved`, `conversation.routing.resolved`, `routing.rule.matched`, fallback attempt lines |

## Code map

| Concern | Location |
|---------|----------|
| Operator store | `internal/operatorstore/` — assistant CRUD, harness attachments, bootstrap |
| Runtime registry | `internal/assistant/registry.go` |
| UI API | `internal/server/adminui/api/assistants/` |
| Chat resolution | `internal/server/server.go`, `assistant_chat.go`, `internal/chat/chat.go`, `internal/harness/` |
| Settings cards | `embed/embedui/settings/render/cards/adminAssistants.js` |
| Routing engine | `internal/routing/`, `internal/routinggen/` |
| Generate helpers | `internal/server/runtime/fallback_availability_audit.go` |
| Tests | `internal/server/assistants_test.go`, `operatorstore/assistants_test.go`, `settings_cards_test.go` |

## Verification

```bash
go test ./chimera/chimera-gateway/internal/server/ -run Assistant
go test ./chimera/chimera-gateway/internal/operatorstore/ -run Assistant
go test ./chimera/chimera-gateway/internal/server/adminui/embed/embedui_test -run Assistant
```

Manual: create two assistants with different fallback chains; chat with each; confirm distinct upstream models and scoped log panels.

## Out of scope and known gaps

- Shared routing-rule definition catalog (reusable named rules across assistants) — assistant stores policy YAML directly today.
- Rate-limit policy per assistant.
- Provider response-header remaining-quota persistence; admission currently uses configured ceilings plus locally observed usage.

## References

- Delivery history: [`plans/archive/virtual-models-operator.md`](../plans/archive/virtual-models-operator.md)
- Harness umbrella: [`plans/assistant-turn-harness.md`](../plans/assistant-turn-harness.md)
- Provider filtering: [Operator provider model availability](operator-provider-model-availability.md)
- Settings surface: [Operator settings UI](operator-settings-ui.md)
- Chat wire contract: [`configuration.md`](../configuration.md#chat-completions-client-contract)
