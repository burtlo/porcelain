# Chimera operator documentation

Documentation is split by **purpose**. Start with [`CURRENT.md`](CURRENT.md) for authority and the current release line, then follow the link that matches your task.

```
docs/
├── README.md              ← you are here (index)
├── CURRENT.md             ← authority map + current release
├── chimera.plan.md        product vision + normative requirements
├── design.md              north-star routing architecture (future depth)
├── installation.md …      runbooks (install, configure, network, supervisor)
├── indexer.md             chimera-indexer operator quick start
├── version-v0.*.md        release-train roadmaps
├── features/              as-built behavior (source of truth for agents)
├── plans/                 open work (draft / active only)
│   └── archive/           shipped delivery history
├── reference/             stable integration references (BiFrost, service map, …)
└── generated/             curated codegen index (see generated/README.md)
```

**Current release line: v0.4.0** (see [CURRENT.md](CURRENT.md)) — assistant turn harness, per-assistant retrieval and workspace policy, gateway workspace tools, harness observability in chat and settings. Prior trains: [version-v0.3.md](version-v0.3.md) (shipped), [version-v0.2.md](version-v0.2.md).

**Operator UI:** after login, app shell at `/ui`; configuration and live logs at **`/ui/settings`**. JSON/SSE APIs under `/api/ui/*`.

---

## Runbooks — install, run, configure

| Document | Description |
|----------|-------------|
| [installation.md](installation.md) | Toolchains, `make chimera-install`, wrapper builds, first run |
| [supervisor.md](supervisor.md) | `chimera-supervisor` and the wrapper stack |
| [network.md](network.md) | Process layout, ports, traffic flow |
| [reference/service-map.md](reference/service-map.md) | Binaries, default ports, health URLs, feature links |
| [configuration.md](configuration.md) | Gateway config files, env vars, reload semantics; [chat completions client contract](configuration.md#chat-completions-client-contract) |
| [packaging.md](packaging.md) | GoReleaser releases, artifacts, `chimera -version` |
| [indexer.md](indexer.md) | `chimera-indexer` operator quick start |
| [../SECURITY.md](../SECURITY.md) | Tokens, logging redaction, local attack surface |

Naming hard-cut notes live in the feature record [product-naming-contract](features/product-naming-contract.md) and archived plan [v0-3-naming-migration](plans/archive/v0-3-naming-migration.md).

---

## As-built — feature records

**[`features/README.md`](features/README.md)** — platform contracts (wrappers, naming, log lines, chat pipeline) and operator features (UI, indexer, assistants, RAG). Use these when extending or debugging shipped behavior.

---

## Product requirements and vision

| Document | Description |
|----------|-------------|
| [CURRENT.md](CURRENT.md) | What is authoritative now; current release; agent summary |
| [chimera.plan.md](chimera.plan.md) | Normative product requirements and roadmap pointers |
| [design.md](design.md) | Cognitive routing north star (not all shipped) |
| [reference/bifrost-upstream.md](reference/bifrost-upstream.md) | BiFrost backend behind `chimera-broker` |
| [reference/tokencount-notes.md](reference/tokencount-notes.md) | Token estimate trade-offs (design discussion) |
| [reference/mcp-sidecar-catalog.md](reference/mcp-sidecar-catalog.md) | Blessed MCP sidecars (`fake_echo`) and Tier B templates for `chimera.yaml` |

---

## Release trains

| Version | Doc | Status |
|---------|-----|--------|
| v0.1 | [version-v0.1.md](version-v0.1.md) | shipped |
| v0.1.1 | [version-v0.1.1.md](version-v0.1.1.md) | shipped |
| v0.2 | [version-v0.2.md](version-v0.2.md) | shipped |
| v0.3 | [version-v0.3.md](version-v0.3.md) | shipped |
| **v0.4.0 (current)** | [version-v0.4.md](version-v0.4.md) | active — harness train `done`; roadmap doc `draft` until release sign-off |
| v0.5 | [version-v0.5.md](version-v0.5.md) | draft |

Template for new version docs: [_version-template.md](_version-template.md).

---

## Plans — open work and archive

**[`plans/README.md`](plans/README.md)** — **Active** and **Draft** plans agents can pick up. Completed plans live under [`plans/archive/`](plans/archive/README.md). Plans preserve *why* and *how*; feature records preserve *what must stay true*.
