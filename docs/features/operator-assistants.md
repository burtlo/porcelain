# Feature: Operator assistants

| Field | Value |
|-------|-------|
| **Doc kind** | `feature-record` |
| **Areas** | Gateway runtime, operator SQLite, chat routing, settings UI |
| **Status** | `current` |
| **Introduced** | Gateway minor after unified operator cards baseline |
| **Originated from** | [`plans/virtual-models-operator.md`](../plans/virtual-models-operator.md) |
| **Related features** | [Operator settings UI](operator-settings-ui.md), [Operator provider model availability](operator-provider-model-availability.md), [Operator chat UI](operator-chat-ui.md), [Context window admission](context-window-admission.md), [Gateway chat routing pipeline](gateway-chat-routing-pipeline.md) |
| **Depends on** | Operator SQLite, broker catalog, routing policy engine, UI session auth |
| **Last updated** | See git history |

## At a glance

Operators create **assistants** in operator SQLite—each with a client-facing `model_id` (name + version), description, enable flag, and visibility—and attach a **routing stack**: required ordered fallback chain, optional routing-policy rules, and optional tool-router block. The gateway loads enabled assistants into an in-memory registry, exposes them on `GET /v1/models`, and resolves `POST /v1/chat/completions` per assistant. Fresh installs start with **zero** assistants (only a routing-rule definition catalog is seeded); operators create assistants in settings. Settings cards support full CRUD, generate-from-catalog, dry-run evaluate, and scoped routing-decision logs carrying `assistant_id`.

## Operator-visible behavior

- **Assistant cards** on `/ui/settings` — list, create, edit metadata, enable/disable, delete.
- **Routing stack editors** — Fallback chain (required), routing policy YAML (toggleable), tool router models + confidence (toggleable). These configure the [chat routing pipeline](gateway-chat-routing-pipeline.md); cards do not execute routing themselves.
- **Generate from catalog** — Builds fallback or policy from **available** upstream models only (respects provider availability).
- **Evaluate / preview** — Dry-run policy against sample message text without sending chat.
- **Scoped logs** — Card expanded panel shows routing, fallback, and tool-router events for that assistant id.
- **Chat selector** — Enabled public assistants appear in `/ui/chat` model dropdown alongside upstream ids.
- **Disabled / private** — Disabled assistants hidden from catalog and rejected on chat; private assistants visible only to creating principal (single-user desktop uses empty tenant today).

## System behavior and contracts

**Invariants**

- Assistants persist in **operator SQLite** (`assistants` and attachment tables); not in `gateway.yaml` for new config.
- Client protocol unchanged: callers send one `model` string on chat/completions.
- Each assistant compiles routing policy into `routing.InMemoryPolicy` at registry reload.
- Fallback walk skips unavailable models (provider availability), quota/context limits, and retriable upstream errors.
- Structured logs include `assistant_id` on routing resolution and fallback attempts (some legacy lines may still carry `virtual_model_id` with the same value until log schema cleanup).
- RAG remains **gateway-global** for v1 (not per-assistant scoped).

**Decisions**

| Topic | Decision |
|-------|----------|
| Model id format | `{Name}-{Version}` stored unique per tenant |
| Bootstrap | Seed routing-rule catalog only; **no** YAML import into SQLite |
| Fallback | Required non-empty chain; generate uses available catalog only |
| Routing rules | Shared policy YAML body per assistant; first matching `when.min_message_chars` wins |
| Tool router | Optional; `router_models[]`, `confidence_threshold`, enable flag |
| Reload | Registry refresh after CRUD via `ReloadAssistants` |
| Direct upstream | Clients may send any upstream `provider/model` id when not using an assistant |

**Identity / auth / scoping**

- Rows scoped by `tenant_id` (empty string default desktop).
- `created_by_principal_id` tracks owner for `private` visibility.
- Chat resolves assistant by exact `model_id` string match.

**Persistence**

- Migrations under `migrations/chimera-gateway/operator/` (assistant tables).
- Attachments: fallback chain JSON, routing policy YAML blob, tool-router fields on assistant row.

### SQLite upgrade path (existing databases)

Migration **`000011_assistants.sql`** renames legacy virtual-model tables and foreign-key columns to assistant terminology. It applies automatically on gateway startup when the operator DB is behind the current migration version.

| Before | After |
|--------|-------|
| `virtual_models` | `assistants` |
| `virtual_model_fallback` | `assistant_fallback` |
| `virtual_model_routing_policy` | `assistant_routing_policy` |
| `virtual_model_tool_router` | `assistant_tool_router` |
| `virtual_model_rule_bindings` | `assistant_rule_bindings` |
| `virtual_model_harness_modules` | `assistant_harness_modules` |
| `virtual_model_id` column | `assistant_id` |

**Operator steps:** stop the gateway (or let supervisor restart it after upgrade), deploy the new gateway binary, and start again. Migrations run once at open; `model_id` values and routing attachments are preserved. No manual SQL is required. If migration fails, check gateway logs for the `operator_migrations` version and restore from backup before retrying.

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
| `POST /api/ui/assistants/{id}/routing/generate` | Generate stack from catalog |
| `POST /api/ui/assistants/{id}/routing/evaluate` | Dry-run policy |
| `GET /v1/models` | Includes enabled assistants |
| `POST /v1/chat/completions` | Resolves `body.model` through assistant registry |
| `GET /api/ui/state` | `assistants[]` summary for settings cards |
| Log slugs | `chat.routing.resolved`, `conversation.routing.resolved`, `routing.rule.matched`, fallback attempt lines |

## Code map

| Concern | Location |
|---------|----------|
| Operator store | `internal/operatorstore/` — assistant CRUD, bootstrap |
| Runtime registry | `internal/assistant/registry.go` |
| UI API | `internal/server/adminui/api/assistants/` |
| Chat resolution | `internal/server/server.go`, `assistant_chat.go`, `internal/chat/chat.go` |
| Settings cards | `embed/embedui/settings/render/cards/adminAssistants.js` |
| Routing engine | `internal/routing/`, `internal/routinggen/` |
| Generate helpers | `internal/server/runtime/fallback_availability_audit.go` |
| Tests | `internal/server/virtual_models_test.go`, `ui_assistants_http_test.go`, `operatorstore/virtual_models_test.go`, `settings_cards_test.go` |

## Verification

```bash
go test ./chimera/chimera-gateway/internal/server/ -run 'Assistant|VirtualModel'
go test ./chimera/chimera-gateway/internal/operatorstore/ -run VirtualModel
go test ./chimera/chimera-gateway/internal/server/adminui/embed/embedui_test -run assistant
```

Manual: create two assistants with different fallback chains; chat with each; confirm distinct upstream models and scoped log panels.

## Out of scope and known gaps

- Per-assistant RAG / workspace scope.
- Shared routing-rule definition catalog (reusable named rules across assistants) — assistant stores policy YAML directly today.
- Rate-limit policy per assistant.

## References

- Plan: [`plans/virtual-models-operator.md`](../plans/virtual-models-operator.md)
- Provider filtering: [Operator provider model availability](operator-provider-model-availability.md)
- Settings surface: [Operator settings UI](operator-settings-ui.md)
- Operator store: [Operator SQLite store](operator-sqlite-store.md)
