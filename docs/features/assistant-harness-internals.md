# Feature: Assistant harness internals

| Field | Value |
|-------|-------|
| **Doc kind** | `platform-contract` |
| **Areas** | Gateway assistant chat path, turn envelope, harness stages |
| **Status** | `current` |
| **Introduced** | v0.4 turn harness |
| **Originated from** | [`plans/assistant-turn-harness.md`](../plans/assistant-turn-harness.md) |
| **Related features** | [Gateway chat routing pipeline](gateway-chat-routing-pipeline.md), [Operator assistants](operator-assistants.md), [Operator log message registry](operator-log-message-registry.md) |
| **Depends on** | Assistant registry, operator SQLite harness modules |
| **Last updated** | See git history |

## At a glance

Assistant chat completions run inside `internal/harness`: ordered **stages** share a **turn envelope** and proxied JSON body. `DefaultRunner()` is the production pipeline wired from `handleAssistantChat`; `PrePrimaryRunner()` runs only deterministic pre-proxy stages for operator harness dry-run. Stage order and log `module` ids are generated into [`docs/generated/harness-stages.md`](../generated/harness-stages.md) so docs stay aligned with `runner.go`.

## Operator-visible behavior

- Scoped assistant logs show `harness.stage.started` / `harness.stage.completed` with `stage` and `module` fields per step.
- Settings **harness evaluate** (`POST /api/ui/assistants/{id}/harness/evaluate`) uses `PrePrimaryRunner` — no retrieval injection or upstream proxy.
- Per-assistant harness module toggles (retrieval, intent, evaluator, escalation, tool_executor) live on the assistant card; see [operator assistants](operator-assistants.md).

## System behavior and contracts

**Invariants**

- Stages run sequentially; `AbortError` stops the turn with a gateway JSON error; `ErrTurnComplete` ends after the terminal proxy stage.
- Fail-safe defaults by stage kind are documented on the `harness` package — transforms and retrieval fail-open; initial pick fails closed when no upstream resolves.
- New production stages belong in `DefaultRunner()` (and `PrePrimaryRunner()` when appropriate for dry-run); regenerate harness stage docs after changing order.

**Decisions**

| Topic | Decision |
|-------|----------|
| Stage contract | `harness.Stage`: `Name()`, `Module()`, `Run(ctx, tc, env, body)` |
| Runner entry points | `DefaultRunner`, `PrePrimaryRunner` in `runner.go` |
| Persisted module ids | `operatorstore.HarnessModuleIDs` — operator UI order, not 1:1 with every stage `module` |

### Generated stage order

Authoritative ordered tables (including `PrePrimaryRunner` and SQLite module ids):

- [`docs/generated/harness-stages.md`](../generated/harness-stages.md) — regenerate with `make harness-docs-generate`.

Narrative stage purposes, mermaid diagram, and routing invariants remain in [Gateway chat routing pipeline](gateway-chat-routing-pipeline.md).

## Interfaces

| Surface | Detail |
|---------|--------|
| Chat entry | `handleAssistantChat` → `harness.DefaultRunner().Run` |
| Harness dry-run | `POST /api/ui/assistants/{id}/harness/evaluate` → `PrePrimaryRunner` |
| Response metadata | `X-Chimera-Harness-Summary`, conversation `harness_summary_json` |
| Log slugs | `harness.stage.started`, `harness.stage.completed` ([registry](operator-log-message-registry.md)) |

## Code map

| Concern | Location |
|---------|----------|
| Runner + stage order | `chimera/chimera-gateway/internal/harness/runner.go` |
| Stage implementations | `stages.go`, `meta_policy.go`, `intent.go`, `tools_stage.go`, `evaluator.go` |
| Turn context / envelope | `context.go`, `envelope.go`, `finalize.go` |
| Stage docs generator | `harness/gendocs/`, `harness/cmd/gendocs/` |
| SQLite harness modules | `chimera/chimera-gateway/internal/operatorstore/assistant_harness.go` |
| Chat wiring | `chimera/chimera-gateway/internal/server/assistant_chat.go` |

## Verification

- `go test ./chimera/chimera-gateway/internal/harness/... -run 'TestGeneratedHarnessStagesMarkdownMatchesFile|TestDefaultRunnerStageOrder' -count=1`
- `make harness-docs-check`

## Out of scope and known gaps

- Formal router plugin registry (see **Target extensibility** in [gateway chat routing pipeline](gateway-chat-routing-pipeline.md)).
- Direct upstream `POST /v1/chat/completions` (non-assistant model id) — bypasses harness.

## References

- Delivery plan: [`plans/assistant-turn-harness.md`](../plans/assistant-turn-harness.md)
- Package doc: `chimera/chimera-gateway/internal/harness/doc.go`
