# chimera-indexer — agent entrypoint

**What it is:** Workspace **file indexer**: watches roots (from gateway SQLite in supervised mode), hashes files, POSTs to gateway `/v1/ingest`. Wrapper supervises in-process `--indexer-backend` worker.

**Docs:** [Operator guide](../../docs/indexer.md) · [Workspace file indexer feature](../../docs/features/indexer.md) · [Indexer workspaces](../../docs/features/indexer-workspaces.md) · [Service map](../../docs/reference/service-map.md)

**Code entrypoints**

- Wrapper + backend: [`main.go`](main.go) (`--indexer-backend` worker path in same binary)
- Wrapper adapter: [`adapter/`](adapter/)
- Indexing pipeline: [`internal/indexer/`](internal/indexer/)
- Wrapper config: [`internal/config/`](internal/config/)

**Generated docs:** [`docs/generated/`](../../docs/generated/README.md) — indexer tuning is YAML + gateway workspace API, not codegen.

**Repo authority:** [`docs/CURRENT.md`](../../docs/CURRENT.md) · [`docs/features/README.md`](../../docs/features/README.md)
