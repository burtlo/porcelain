---
name: operator-log-slug-registry
description: >-
  Adds or updates operator log message slugs in messages.yaml, regenerates
  contracts, and wires Go emit sites to naming constants. Use when adding
  harness.*, conversation.*, rag.*, or other structured slog msg slugs, or
  when the user mentions operator copy, gallery_preview, or log message registry.
---

# Operator log slug registry

Single catalog: `internal/operatorcopy/messages.yaml`.  
Feature record: [`docs/features/operator-log-message-registry.md`](../../docs/features/operator-log-message-registry.md).

## When adding a slug

1. Choose a stable dotted slug (`harness.escalation.re_retrieve`, not a sentence).
2. Add YAML entry with at least:
   - `slug`
   - `summary` and/or `formatter`
   - **`gallery_preview`** (required)
   - `shape` / `metrics_counter` when it should roll up like siblings
3. Prefer **no new aliases** for greenfield harness work (hard cut).
4. Regenerate:

```bash
make contracts-generate
# or: go generate ./internal/operatorcopy/...
make contracts-check
```

5. Emit in Go with generated constants from `internal/naming/log_messages.go` (`naming.Msg…`), never a raw new string in hot paths.
6. Include useful KV: `assistant_id`, `turn_index`, `stage`, `module`, `timeline_kind` as appropriate (match existing harness stage logs).

## Harness conventions

| Slug | Role |
|------|------|
| `harness.stage.started` / `.completed` | Emitted by runner — reuse; don’t duplicate per stage |
| `harness.turn.completed` | Finalize / envelope summary |
| `harness.*` (new) | Module-specific events (escalation, compress fallback, scope ambiguous, …) |

Register new module events in YAML **before** or in the same PR as emit sites.

## Verification

```bash
go test ./internal/operatorcopy/...
go test ./chimera/chimera-gateway/internal/server/adminui/embed/embedui_test -run OperatorMessage
make contracts-check
```

Optional: inventory / msg-audit scripts if touching many emit sites (`make` / `scripts/operatorcopy-*`).

## Anti-patterns

- Hard-coded summary strings in settings JS switches for new slugs
- Shipping emit sites without YAML + `gallery_preview`
- Reintroducing legacy alias soup for brand-new harness slugs
- Forgetting regenerate so `log_messages.go` / `operator_copy.js` drift (CI `contracts-check` fails)

## Related

- Structured lines: [`docs/features/structured-operator-log-lines.md`](../../docs/features/structured-operator-log-lines.md)
- Harness stages: [harness-stage-implementer](../harness-stage-implementer/SKILL.md)
