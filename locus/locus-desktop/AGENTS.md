# locus-desktop — agent entrypoint

**What it is:** **Locus desktop launcher** — connect-first to `chimera-supervisor`, opens operator webview, optional native folder picker for indexer workspaces. Does not replace gateway auth or SQLite.

**Docs:** [Locus desktop ↔ supervisor](../../docs/features/locus-desktop-supervisor.md) · [Installation](../../docs/installation.md) · [Supervised stack](../../docs/supervisor.md) · [Service map](../../docs/reference/service-map.md)

**Code entrypoints**

- CLI: [`main.go`](main.go)
- App shell / supervisor attach: [`internal/app/`](internal/app/)
- Shared Locus flags: [`../../internal/locus/`](../../internal/locus/)

**Generated docs:** [`docs/generated/`](../../docs/generated/README.md) — desktop uses gateway embed UI assets from the gateway binary.

**Repo authority:** [`docs/CURRENT.md`](../../docs/CURRENT.md) · [`docs/features/README.md`](../../docs/features/README.md)
