# Generated documentation

Files in this directory are produced by **`make docs-generate`** (markdown only). Operator UI JS/Go contracts also refresh when you run **`make contracts-generate`**. Do not edit generated files by hand.

| Output | Generator |
|--------|-----------|
| [`operator-sqlite-migrations.md`](operator-sqlite-migrations.md) | `go run ./internal/docgen/migrations/cmd` |
| [`harness-stages.md`](harness-stages.md) | `go generate ./chimera/chimera-gateway/internal/harness/...` |
| [`log-slug-index.md`](log-slug-index.md) | `go generate ./internal/operatorcopy/...` |
| [`gateway-http-routes.md`](gateway-http-routes.md) | `go run ./internal/docgen/routes/cmd` |

Stale checks run as part of `make contracts-check`.
