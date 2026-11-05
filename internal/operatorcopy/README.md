# Operator message registry

Canonical structured-log slugs, legacy aliases, and English operator copy for the gateway logs UI.

| Artifact | Role |
|----------|------|
| `messages.yaml` | Source of truth (embedded; validated by `go generate`) |
| `docs/generated/log-slug-index.md` | Agent slug index (generated; see [`docs/generated/README.md`](../../docs/generated/README.md)) |
| `bootstrap_registry.go` | Go catalog used to regenerate YAML (`go run ./cmd/bootstrap`) |
| `inventory-report.txt` | Last inventory diff (`scripts/operatorcopy-inventory.ps1 -WriteReport`) |

## Decisions (Phase 1)

- **Path:** `internal/operatorcopy/messages.yaml` (Go-owned, embedded).
- **Locale:** English only (`locale: en`).
- **Gallery:** `gallery_preview` required on every message (component gallery, Phase 2+).

## Commands

```bash
go generate ./internal/operatorcopy/...   # validate messages.yaml + write operator_copy.js + log-slug-index.md
go test ./internal/operatorcopy/...
go run ./internal/operatorcopy/cmd/bootstrap   # rewrite messages.yaml from bootstrap_registry.go
make contracts-generate   # bootstrap + generate (Phase 2+); alias: make operator-contracts-generate
make contracts-check      # stale checks for operator_copy.js, log_messages.go, log-slug-index.md
```

```powershell
scripts/operatorcopy-inventory.ps1 -WriteReport
```

Registry holds **copy + identity + presentation hints** (`shape`, `metrics_counter`); storage and normalize logic stay elsewhere (see `docs/plans/operator-message-registry.md`).

Generated `operator_copy.js` also exposes `Slug` (canonical slug constants), `inferShapeForFlat`, and `metricsCounterForFlat`. Shape taxonomy aligns with [`docs/plans/log-presentation-layer.md`](../docs/plans/log-presentation-layer.md) §2 (`http.access`, `chat.request`, `rag`, `ingest`, …).

Phases 2–4 render via `operatorMessage.js` + `operatorMessageServices.js` + `operatorMessageIndexer.js`. Phase 6 wires `inferShape` (`logs_app.js`) and gateway card counters (`gatewayCardModel.js`) to registry tags.
