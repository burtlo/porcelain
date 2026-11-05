# chimera-supervisor — agent entrypoint

**What it is:** Parent **orchestrator** for the Chimera wrapper stack (vector store, broker, gateway, optional indexer). Exposes a control HTTP plane and tees child logs into the operator servicelog buffer.

**Docs:** [Supervised stack](../../docs/supervisor.md) · [Service map](../../docs/reference/service-map.md) · [Wrapper binary contract](../../docs/features/chimera-wrapper-binary-contract.md) · [Locus desktop ↔ supervisor](../../docs/features/locus-desktop-supervisor.md)

**Code entrypoints**

- CLI: [`main.go`](main.go)
- Config / flags: [`internal/config/`](internal/config/)
- Child lifecycle: [`internal/supervise/`](internal/supervise/)
- Control HTTP (`/healthz`, `/readyz`, `/status`, `/shutdown`): [`internal/control/`](internal/control/)

**Generated docs:** [`docs/generated/`](../../docs/generated/README.md) — supervisor itself has no generated config reference.

**Repo authority:** [`docs/CURRENT.md`](../../docs/CURRENT.md) · [`docs/features/README.md`](../../docs/features/README.md)
