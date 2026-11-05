# chimera-vectorstore — agent entrypoint

**What it is:** **Wrapper** around upstream vector storage (default Qdrant via `--bin`). Translates `VECTORSTORE__*` env; backend HTTP/gRPC on Qdrant defaults.

**Docs:** [Gateway RAG ingest and retrieval](../../docs/features/gateway-rag-ingest-and-retrieval.md) · [Service map](../../docs/reference/service-map.md) · [Wrapper binary contract](../../docs/features/chimera-wrapper-binary-contract.md)

**Code entrypoints**

- Wrapper: [`main.go`](main.go) → [`adapter/`](adapter/)
- Flags / env: [`internal/config/`](internal/config/)
- Shared contract: [`../internal/wrapper/`](../internal/wrapper/) · readiness probe `GET /collections` on backend

**Generated docs:** [`docs/generated/`](../../docs/generated/README.md) — no vectorstore-specific generated schema doc.

**Repo authority:** [`docs/CURRENT.md`](../../docs/CURRENT.md) · [`docs/features/README.md`](../../docs/features/README.md)
