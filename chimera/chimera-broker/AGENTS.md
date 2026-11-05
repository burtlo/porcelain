# chimera-broker — agent entrypoint

**What it is:** **Wrapper** around upstream BiFrost HTTP (`bifrost-http` via `--bin`). Translates `BROKER__*` env, exposes Chimera control plane, proxies LLM traffic on the backend port.

**Docs:** [BiFrost upstream](../../docs/reference/bifrost-upstream.md) · [Service map](../../docs/reference/service-map.md) · [Wrapper binary contract](../../docs/features/chimera-wrapper-binary-contract.md) · [Product naming](../../docs/features/product-naming-contract.md)

**Code entrypoints**

- Wrapper: [`main.go`](main.go) → [`adapter/`](adapter/)
- Flags / env: [`internal/config/`](internal/config/)
- Shared contract: [`../internal/wrapper/`](../internal/wrapper/) · readiness probe `GET /models` on backend

**Generated docs:** [`docs/generated/`](../../docs/generated/README.md) — broker config remains `config/bifrost.config.json` + env (see [configuration.md](../../docs/configuration.md)).

**Repo authority:** [`docs/CURRENT.md`](../../docs/CURRENT.md) · [`docs/features/README.md`](../../docs/features/README.md)
