# Feature: Chimera stack configuration

| Field | Value |
|-------|-------|
| **Doc kind** | `platform-contract` |
| **Areas** | `chimera.yaml`, supervisor launch, gateway config load, indexer materialize |
| **Status** | `current` |
| **Introduced** | Chimera stack config redesign |
| **Originated from** | [`plans/chimera-stack-config.md`](../plans/archive/chimera-stack-config.md) |
| **Related features** | [Product naming](product-naming-contract.md), [Locus desktop ↔ supervisor](locus-desktop-supervisor.md), [Indexer](indexer.md), [Gateway RAG ingest and retrieval](gateway-rag-ingest-and-retrieval.md), [Structured operator log lines](structured-operator-log-lines.md) |
| **Depends on** | Naming contracts, supervisor control plane |
| **Last updated** | See git history |

## At a glance

Operators configure the supervised Chimera stack in **`config/chimera.yaml`**: suite membership per service, an explicit supervisor launch list, per-service log emit levels, a separate supervisor collector gate, indexer tuning (with optional overlay), and the shared **search** platform. Virtual-model routing and workspaces stay in operator SQLite.

## Operator-visible behavior

- Configure `chimera.yaml` (example: `chimera.example.yaml`); `make configure` copies the example when missing.
- Default launch runs vectorstore, broker, gateway, and indexer when each is `enabled: true`.
- Setting only a service to `debug` does not show debug in `/ui` unless `supervisor.log_level` (or `LOG_LEVEL`) allows debug.
- Indexer advanced YAML in settings edits `chimera.yaml` `indexer:` when no overlay is set; otherwise the `config_path` overlay.

## System behavior and contracts

**Invariants**

- **`*.enabled`** = suite membership; **`supervisor.services`** = processes to start.
- Omitted `supervisor.services` → all enabled services; listed + disabled → startup error.
- Log visibility requires emit **and** collector gate; gate is `supervisor.log_level` (`LOG_LEVEL` overrides gate only).
- Search platform YAML key is `search`; HTTP remains `/v1/rag/*`.
- Indexer supervised config is **materialized** merge of inline `indexer:` + optional `config_path` overlay.
- Config path resolve: `CHIMERA_CONFIG` → `./config/chimera.yaml`.

**Decisions**

| Topic | Decision |
|-------|----------|
| File basename | `chimera.yaml` (not gateway-only) |
| Indexer launch | Launch list, not `indexer.supervised.enabled` |
| RAG rename | YAML `search`; keep RAG type / HTTP paths |
| Retrieval knobs in YAML | `search.retrieval_defaults` seed VMs; per-VM overrides in SQLite (v0.4) |
| UI indexer PUT | No `config_path` → patch `chimera.yaml`; with overlay → write that file; rematerialize |

## Interfaces

| Surface | Detail |
|---------|--------|
| Config | `config/chimera.yaml` |
| Env | `CHIMERA_CONFIG`, `LOG_LEVEL` (collector gate) |
| Load | `config.LoadChimeraYAML`, `ResolveChimeraConfigPath` |
| Materialize | `config.MaterializeIndexerConfig` → `data/gateway/indexer.materialized.yaml` |
| UI | `GET/PUT /api/ui/indexer/config` (effective; write target = overlay or `chimera.yaml`) |

## Code map

| Concern | Location |
|---------|----------|
| Schema + resolve | `chimera/internal/config/` |
| Supervisor launch + log gate | `chimera/chimera-supervisor/internal/supervise/` |
| Naming constants | `internal/naming/contracts.go` |
| Indexer UI | `chimera/chimera-gateway/internal/server/adminui/api/indexer/` |

## Verification

- `go test ./chimera/internal/config/ ./chimera/chimera-supervisor/...`
- Manual: `supervisor.log_level: debug` + `vectorstore.log_level: debug` → debug in settings logs; gate `info` → no vectorstore debug.

## Out of scope and known gaps

- Per-VM retrieval UI (harness plans).
- Renaming `/v1/rag/*` HTTP routes.
- Renaming `CHIMERA_GATEWAY_URL` / wrapper `GATEWAY__*` families.

## References

- Plan: [`plans/chimera-stack-config.md`](../plans/archive/chimera-stack-config.md)
- Operator docs: [`configuration.md`](../configuration.md), [`supervisor.md`](../supervisor.md)
