# Plan: Chimera stack configuration redesign

| Field | Value |
|-------|-------|
| **Doc kind** | `feature-plan` |
| **Owners / areas** | `chimera-supervisor`, gateway config, indexer, naming contracts, operator docs |
| **Status** | `shipped` |
| **Targets** | Chimera supervised stack (locus-desktop / chimera-supervisor) |
| **Last updated** | See git history |
| **Supersedes / superseded by** | None — complements [`locus-desktop-supervisor-contract.md`](locus-desktop-supervisor-contract.md); aligns with v0.4 harness YAML-vs-SQLite ownership |
| **As-built** | [`chimera-stack-config`](../features/chimera-stack-config.md), [`product-naming-contract`](../features/product-naming-contract.md), [`locus-desktop-supervisor`](../features/locus-desktop-supervisor.md), [`indexer`](../features/indexer.md), [`gateway-rag-ingest-and-retrieval`](../features/gateway-rag-ingest-and-retrieval.md), [`structured-operator-log-lines`](../features/structured-operator-log-lines.md) |

## At a glance

Operators configure the whole supervised Chimera stack in one file (`chimera.yaml`): which services belong to the suite, which the supervisor launches, how each process connects and logs, and the shared search platform. Per-service debug logging works without fighting a hidden gateway filter; the supervisor’s collector gate is explicit. Virtual-model taste stays in operator SQLite — YAML holds platform defaults and hard gates only.

| Phase | Outcome | Status |
|-------|---------|--------|
| [Phase 1 — Rename and path resolution](#phase-1--rename-and-path-resolution) | Stack config is `chimera.yaml` / `CHIMERA_CONFIG` (legacy removed) | `done` |
| [Phase 2 — Supervisor section and launch list](#phase-2--supervisor-section-and-launch-list) | Explicit `supervisor.services`; suite `enabled` on every service; indexer no longer a special toggle | `done` |
| [Phase 3 — Emit vs collector log levels](#phase-3--emit-vs-collector-log-levels) | Per-service emit levels; `supervisor.log_level` gates the shared log feed | `done` |
| [Phase 4 — Nested schema and search rename](#phase-4--nested-schema-and-search-rename) | Clear ownership for timeouts, auth, metrics, broker models; `rag` → `search` with retrieval defaults | `done` |
| [Phase 5 — Indexer inline config and overlay](#phase-5--indexer-inline-config-and-overlay) | Indexer tuning in `chimera.yaml` with optional overlay; materialize for supervised child | `done` |
| [Phase 6 — Docs and feature records](#phase-6--docs-and-feature-records) | Operator docs + shipped feature records / contract updates | `done` |

---

## Background

Today’s `chimera.yaml` name understates that the file configures broker, vectorstore, indexer supervision, and RAG. The indexer is the only service with an explicit supervised enable flag; others start by convention. The supervisor’s log ring filters all children using `gateway.log_level`, so setting `vectorstore.log_level: debug` has no effect in `/ui/logs`. Top-level `paths`, `health`, `metrics`, and `rag` mix unrelated concerns. v0.4 virtual-model harness plans move per-VM retrieval into SQLite while keeping a global RAG/search gate in YAML — this redesign names that split clearly.

**Related docs:** [`configuration.md`](../configuration.md), [`supervisor.md`](../supervisor.md), [`indexer.md`](../archive/indexer.md), [`features/product-naming-contract.md`](../features/product-naming-contract.md), [`features/locus-desktop-supervisor.md`](../features/locus-desktop-supervisor.md), [`features/gateway-rag-ingest-and-retrieval.md`](../features/gateway-rag-ingest-and-retrieval.md), [`virtual-model-turn-harness.md`](virtual-model-turn-harness.md), [`virtual-model-harness-retrieval.md`](virtual-model-harness-retrieval.md).

**Working design detail** (target YAML shape, merge diagrams, key map) also lives in the Cursor plan artifact used during authoring; this file is the durable delivery plan.

---

## Ownership layers (YAML vs SQLite)

| Layer | Home | Examples |
|-------|------|----------|
| Stack / deploy | `chimera.yaml` | Supervisor launch, listen URLs, timeouts, auth file, metrics SQLite, search platform (embed/chunk/ingest), indexer tuning |
| Product / per-VM | Operator SQLite | Virtual models, harness modules, per-VM retrieval overrides, workspaces, conversations |

**Search inheritance (future harness):**

- `search.enabled: false` → all VM retrieval modules no-op (hard gate).
- `search.enabled: true` → platform available; new VMs default retrieval on; per-VM can disable.
- `search.embedding` / `chunking` / `ingest` stay global (corpus rebuild).
- `search.retrieval_defaults` seed new VM retrieval modules; chat eventually reads VM config for those knobs.

---

## Target shape (normative sketch)

```yaml
supervisor:
  listen: "127.0.0.1:7710"
  log_json: true
  log_level: info                 # collector gate (ring / UI / tee)
  services: [vectorstore, broker, gateway, indexer]  # omit = all enabled
gateway:
  enabled: true
  log_level: info                 # emit only
  timeouts: { broker_ms: 5000, chat_ms: 300000 }
  catalog: { poll_ms: 30000 }
  auth: { api_keys: "./api-keys.yaml" }
  metrics: { enabled: true }
broker:
  enabled: true
  url: "http://127.0.0.1:8080"
  log_level: info
  models: { free_tier: "./provider-free-tier.yaml", limits: "./provider-model-limits.yaml" }
vectorstore:
  enabled: true
  url: "http://127.0.0.1:6333"
  log_level: debug
indexer:
  enabled: true
  # config_path: "indexer.yaml"   # optional last-wins overlay
  log_level: debug
  job_skip_log: debug
search:
  enabled: true
  embedding: { model: "ollama/nomic-embed-text:latest", dim: 768, path: "/v1/embeddings" }
  chunking: { size: 512, overlap: 128 }
  ingest: { max_bytes: 10485760 }
  retrieval_defaults: { top_k: 8, score_threshold: 0.50 }
  coherence: { mode: warn }
  tooling: { enabled: true }
```

Section order in examples: `supervisor` → `gateway` → `broker` → `vectorstore` → `indexer` → `search`.

**Suite vs launch:** `*.enabled` = suite membership (not process start). `supervisor.services` = children to start. Omitted list → all enabled services. Name listed but `enabled: false` → startup error. Enabled but not launched → external URL only.

**Logging:** visibility requires emit **and** collector gate. `LOG_LEVEL` env overrides `supervisor.log_level` (gate only). Gateway slog uses `gateway.log_level`.

**Indexer:** same `FileConfig` keys as standalone `indexer.yaml` under `indexer:`; optional `config_path` overlays; supervisor materializes merged YAML for `--config`. UI GET = effective; UI PUT = overlay file.

HTTP `/v1/rag/*` routes unchanged in this plan (YAML `search` rename only).

---

## Phase 1 — Rename and path resolution

**Goal.** The supervised stack config file is named for the whole product, not only the gateway.

**Deliverables**

- Rename `config/chimera.yaml` / `gateway.example.yaml` → `chimera.yaml` / `chimera.example.yaml`.
- Naming constants: `ChimeraConfigFileTarget`, `DefaultChimeraConfigRelPath`, `CHIMERA_CONFIG` only.
- Resolve order: `CHIMERA_CONFIG` → `./config/chimera.yaml`.
- **Hard cut:** no `gateway.yaml` / `CHIMERA_GATEWAY_CONFIG` fallback (legacy removed).
- Update configure/release/clean scripts and loader call sites (`LoadChimeraYAML` or alias).

**Acceptance**

- Fresh clone: `make` configure produces `config/chimera.yaml`; locus-desktop / supervisor start with no `chimera.yaml`.
- Only `CHIMERA_CONFIG` / `chimera.yaml` load; leftover `chimera.yaml` is ignored.

**Status:** `done`

---

## Phase 2 — Supervisor section and launch list

**Goal.** Operators see an explicit list of processes the supervisor will run, with parity across services.

**Deliverables**

- `supervisor:` block: listen, bins, waits, shutdown knobs (today’s CLI flags), `services` list, `log_json`.
- `enabled` on gateway, broker, vectorstore, indexer.
- Remove `indexer.supervised.enabled` / `start_when_rag_disabled` as launch gates; indexer starts when listed (and suite-enabled).
- Default launch = all `enabled: true` services; start order vectorstore → broker → gateway → indexer.
- Bootstrap (no api-keys): document gateway-only override.

**Acceptance**

- Default config starts indexer without a special supervised flag.
- Omitting indexer from `services` leaves other children up; gateway still connects to external services when suite-enabled.

**Status:** `done`

---

## Phase 3 — Emit vs collector log levels

**Goal.** Setting a service to debug actually works when the collector gate allows it; the gate is obvious in config.

**Deliverables**

- `LogSink` min-level from `supervisor.log_level` only (stop using `gateway.log_level` as the shared filter).
- Pass emit levels into children (`-log-level`, indexer `log_level` in materialized config).
- Tests for gate vs emit matrix; docs call out both axes.

**Acceptance**

- `supervisor.log_level: debug` + `vectorstore.log_level: debug` → debug lines in `/ui/logs` and supervisor log file.
- `supervisor.log_level: info` + service `debug` → no debug in the shared feed.

**Status:** `done`

---

## Phase 4 — Nested schema and search rename

**Goal.** Top-level keys match who owns the setting; RAG is named for operators as search platform config.

**Deliverables**

- Map `health` → `gateway.timeouts` / `gateway.catalog`; `paths` → `gateway.auth` + `broker.models`; `metrics` → `gateway.metrics`.
- Rename `rag` → `search`; `rag.retrieval` → `search.retrieval_defaults`; `broker.base_url` → `broker.url` (aliases one release).
- Wire Resolved + callers; deprecation logs for old keys.
- Leave `/v1/rag/*` HTTP paths unchanged.

**Acceptance**

- Example `chimera.example.yaml` has no top-level `rag` / `health` / `paths` / `metrics`.
- With only legacy keys, stack still starts (alias path) and logs hints.

**Status:** `done`

---

## Phase 5 — Indexer inline config and overlay

**Goal.** Supervised indexer tuning lives in `chimera.yaml` with the same schema as standalone indexer YAML.

**Deliverables**

- Parse `indexer` FileConfig keys inline; optional `indexer.config_path` last-wins overlay.
- Materialize merged YAML to a runtime path; rematerialize on chimera config reload.
- Admin UI GET effective / PUT overlay (`config_path` or default `indexer.yaml`).

**Acceptance**

- Inline-only supervised run (no `config_path`) indexes after workspace save.
- Overlay edits via UI survive rematerialize of inline base.

**Status:** `done`

---

## Phase 6 — Docs and feature records

**Goal.** Operator runbooks and as-built feature records match the shipped schema.

**Deliverables**

- Update [`configuration.md`](../configuration.md), [`supervisor.md`](../supervisor.md), [`indexer.md`](../archive/indexer.md).
- Ship feature records per [Feature records on ship](#feature-records-on-ship); set this plan **As-built** links; add README rows; mark plan `shipped`.

**Acceptance**

- An agent reading only `docs/features/` can configure `chimera.yaml`, explain suite vs launch, and explain emit vs collector logging without reading this plan.

**Status:** `done`

---

## Feature records on ship

Create or update using [`docs/features/_template.md`](../features/_template.md). Distill invariants and interfaces — do not copy phase checklists.

| Action | Feature record | Doc kind | What to capture |
|--------|----------------|----------|-----------------|
| **Create** | [`chimera-stack-config.md`](../features/chimera-stack-config.md) | `platform-contract` | `chimera.yaml` layout; suite `enabled` vs `supervisor.services`; emit vs collector log levels; YAML vs SQLite ownership; `search` platform + `retrieval_defaults`; indexer inline + overlay; config path / `CHIMERA_CONFIG` resolve order |
| **Update** | [`product-naming-contract.md`](../features/product-naming-contract.md) | `platform-contract` | Config basename `chimera.yaml`; `CHIMERA_CONFIG` (+ legacy alias) |
| **Update** | [`locus-desktop-supervisor.md`](../features/locus-desktop-supervisor.md) | `platform-contract` | Launch list from config; collector gate; indexer not optional-by-default |
| **Update** | [`indexer.md`](../features/indexer.md) | `feature-record` | Supervised config from `chimera.yaml` / overlay; remove `indexer.supervised.enabled` narrative |
| **Update** | [`gateway-rag-ingest-and-retrieval.md`](../features/gateway-rag-ingest-and-retrieval.md) | `feature-record` | Config key `search.*` (note HTTP `/v1/rag/*` unchanged); `retrieval_defaults` vs future per-VM retrieval |
| **Update** | [`structured-operator-log-lines.md`](../features/structured-operator-log-lines.md) | `platform-contract` | Supervisor collector min-level from `supervisor.log_level` |
| **Update** | [`docs/features/README.md`](../features/README.md) | index | Row for new platform contract; refresh related rows |

Also update this plan’s **As-built** front-matter cell to link the new/updated records when status becomes `shipped`.

---

## Out of scope

- Renaming wrapper env families or `CHIMERA_GATEWAY_URL` / `TOKEN`.
- Moving workspace roots or virtual-model routing into YAML.
- Changing `/v1/rag/*` HTTP routes or implementing per-VM retrieval UI (v0.4 child plans).

---

## References

- Code: `chimera/internal/config/`, `chimera/chimera-supervisor/internal/supervise/`, `chimera/chimera-indexer/internal/indexer/config.go`, `internal/naming/contracts.go`
- Docs: [`configuration.md`](../configuration.md), [`supervisor.md`](../supervisor.md)
- Related plans: [`virtual-model-harness-retrieval.md`](virtual-model-harness-retrieval.md), [`locus-desktop-supervisor-contract.md`](locus-desktop-supervisor-contract.md)
