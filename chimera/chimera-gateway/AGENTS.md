# chimera-gateway — agent entrypoint

**What it is:** Chimera **gateway wrapper** plus `--gateway-backend` process: OpenAI-style `/v1/*`, operator embed UI (`/ui/*`), chat turn harness, RAG ingest/retrieval, operator SQLite.

**Docs:** [Gateway chat routing pipeline](../../docs/features/gateway-chat-routing-pipeline.md) · [Operator assistants](../../docs/features/operator-assistants.md) · [Configuration](../../docs/configuration.md) · [Service map](../../docs/reference/service-map.md)

**Code entrypoints**

- Wrapper: [`main.go`](main.go) (default control `127.0.0.1:7720`, backend serve `127.0.0.1:3000`)
- Backend: `main.go --gateway-backend` → [`internal/server/`](internal/server/)
- Harness stages: [`internal/harness/`](internal/harness/)
- Operator store: [`internal/operatorstore/`](internal/operatorstore/)
- Shared wrapper runtime: [`../internal/wrapper/`](../internal/wrapper/)

**Generated docs:** [`docs/generated/`](../../docs/generated/README.md) (operator log constants and other codegen — not config schema).

**Repo authority:** [`docs/CURRENT.md`](../../docs/CURRENT.md) · [`docs/features/README.md`](../../docs/features/README.md)
