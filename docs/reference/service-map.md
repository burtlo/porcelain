# Chimera service map

Hand-maintained index of **wrapper binaries**, default loopback ports, primary config, health surfaces, and feature records. Defaults match a typical **supervised local stack**; overrides use `-listen`, `GATEWAY__*`, `BROKER__*`, `VECTORSTORE__*`, `INDEXER__*`, and supervisor flags — **verify in code** (`internal/naming`, each binary’s `internal/config`, [`network.md`](../network.md), [`supervisor.md`](../supervisor.md), [`probes.go`](../../chimera/internal/wrapper/contract/probes.go)).

## Services

| Binary | Role | Default control port (`-listen`) | Backend / serve port | Primary config | Health (defaults) | Feature record |
|--------|------|----------------------------------|----------------------|----------------|-------------------|----------------|
| **chimera-supervisor** | Orchestrates wrapper children; tees logs to operator buffer | `127.0.0.1:7710` | *(none — supervises others)* | `-config` → `config/gateway.yaml`; child paths via `-gateway-bin`, `-broker-bin`, `-vectorstore-bin`, `-indexer-bin` | Control: `GET /healthz`, `/readyz`, `/status`, `/metrics`; `POST /shutdown` | [Wrapper binary contract](../features/chimera-wrapper-binary-contract.md), [Locus desktop ↔ supervisor](../features/locus-desktop-supervisor.md) |
| **chimera-gateway** | Client-facing API, operator UI (`/ui/*`), RAG orchestration, operator SQLite | `127.0.0.1:7720` | `127.0.0.1:3000` (gateway backend) | `config/gateway.yaml`, `config/api-keys.yaml`, path refs in YAML; `operator.sqlite` at runtime | Wrapper: `/healthz`, `/readyz`, `/status`, `/metrics`, optional `/debug/broker/logs`. Backend readiness probe: `GET /healthz` (200). Operator summary: `GET /health` on **3000** | [Gateway chat routing pipeline](../features/gateway-chat-routing-pipeline.md), [Operator SQLite store](../features/operator-sqlite-store.md), [Gateway RAG ingest and retrieval](../features/gateway-rag-ingest-and-retrieval.md) |
| **chimera-broker** | Wrapper around BiFrost HTTP relay | `127.0.0.1:7730` | `127.0.0.1:8080` | `config/bifrost.config.json`; `BROKER__*` env | Wrapper: `/healthz`, `/readyz`, `/status`, `/metrics`, optional `/debug/broker/logs`. Readiness probes backend `GET /models` (200) | [Wrapper binary contract](../features/chimera-wrapper-binary-contract.md), [BiFrost upstream](bifrost-upstream.md) |
| **chimera-vectorstore** | Wrapper around Qdrant (default `--bin`) | `127.0.0.1:7740` | `127.0.0.1:6333` HTTP, `6334` gRPC | `VECTORSTORE__*` env; upstream via `--bin` | Wrapper: `/healthz`, `/readyz`, `/status`, `/metrics`, optional `/debug/vectorstore/logs`. Readiness probes backend `GET /collections` (200) | [Gateway RAG ingest and retrieval](../features/gateway-rag-ingest-and-retrieval.md), [Wrapper binary contract](../features/chimera-wrapper-binary-contract.md) |
| **chimera-indexer** | Workspace file watcher; ingest via gateway APIs | `127.0.0.1:7750` | In-process worker (`--indexer-backend`; no separate HTTP backend port) | Supervised: `indexer.supervised.yaml` (from gateway YAML); standalone: layered YAML + `CHIMERA_GATEWAY_URL` / `CHIMERA_GATEWAY_TOKEN` | Wrapper: `/healthz`, `/readyz`, `/status`, `/metrics`, optional `/debug/broker/logs` | [Workspace file indexer](../features/indexer.md), [Indexer workspaces](../features/indexer-workspaces.md) |
| **locus-desktop** | Desktop shell; connect-first to supervisor; native folder picker | *(none — not a wrapper)* | Webview targets supervisor-derived gateway base | Launcher flags + inherited `env` / `.env`; see [installation.md](../installation.md) | Attach/readiness via supervisor + gateway (not a Chimera wrapper control plane) | [Locus desktop ↔ supervisor](../features/locus-desktop-supervisor.md) |

## Readiness probe lock (Phase 1)

From `InitialBinaryReadinessProbes` in [`probes.go`](../../chimera/internal/wrapper/contract/probes.go): gateway backend **`GET /healthz`**, broker backend **`GET /models`**, vector store **`GET /collections`**. Wrapper **`GET /readyz`** reflects backend readiness using these probes.

## Related runbooks

- [Network architecture](../network.md) — traffic flow and port table
- [Supervised stack](../supervisor.md) — flags, make targets, shutdown
- [Configuration](../configuration.md) — gateway YAML, reload, env vars
