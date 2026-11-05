# Chimera configuration reference

For an auto-generated catalog of **chimera-gateway** and **chimera-supervisor** HTTP routes (method, path, auth), see [generated/gateway-http-routes.md](generated/gateway-http-routes.md).

Chimera (the gateway runtime in Porcelain) reads **YAML files** and **environment variables**.

## Operator vocabulary (Chimera products)

In UI, logs, supervisor output, and this doc set, use **Chimera product names**:

| Product | Role |
|---------|------|
| **chimera-gateway** | Client-facing API, routing, RAG orchestration, operator UI |
| **chimera-broker** | LLM relay (supervised wrapper around the OpenAI-compatible backend) |
| **chimera-vectorstore** | Vector retrieval for RAG (supervised wrapper around the storage backend) |
| **chimera-embed** | Optional internal embedding wrapper around operator-supplied llama-server |
| **chimera-indexer** | File ingest into collections scoped by workspace |

**Config file keys** use `broker.*`, top-level `vectorstore.*`, and `rag.*` (orchestration) in `gateway.yaml`. **`GET /status`** exposes `broker` and `vectorstore` blocks; nested debug fields name the wrapped storage/relay implementation only.

Naming refactor status: [plans/chimera-gateway-refactor.md](plans/archive/chimera-gateway-refactor.md). `gateway.yaml`, `api-keys.yaml`, and `provider-free-tier.yaml` (when configured) are picked up when their file **modification time** changes (`gateway.yaml` reload also runs when `provider-free-tier.yaml` alone changes). **Gateway metrics** (the `metrics` block in `gateway.yaml`) are applied **at process start** only (changing paths requires a **restart**).

## Go gateway binary

The `chimera` program (`go build -o chimera ./cmd/chimera`) reads:

- **Config path:** `CHIMERA_GATEWAY_CONFIG`, or `-config /path/to/gateway.yaml`, or default `./config/gateway.yaml` (relative to the process working directory).
- **Listen address:** from `gateway.listen_host` and `gateway.listen_port`, unless overridden with `-listen` (e.g. `:3001` or `host:port`).
- **Log level:** `gateway.log_level` unless `LOG_LEVEL` is set (`debug`, `info`, `warn`, `error`); Go uses `log/slog` text logs on stdout.
- **Broker endpoint (YAML `broker.*`):** `broker.base_url`, `broker.api_key_env`, `health.*`, `paths.*` — see tables below. Points at **chimera-broker** (or a standalone OpenAI-compatible proxy during local dev).
- **`.env`:** At startup, the runtime loads an optional `.env` in the **process working directory** (via `github.com/joho/godotenv`). Missing file is normal when the environment is injected by your shell or service manager.

`GET /health` returns JSON including `checks.vectorstore` when RAG is enabled and `checks.upstream` (broker/backend probe). `GET /v1/models` lists **enabled assistants** from operator SQLite (when any exist) merged with the **chimera-broker** catalog (API/SQLite may still label these rows *virtual models*). `POST /v1/chat/completions` validates the gateway Bearer token; when `body.model` matches an assistant id the gateway runs that assistant's harness and routing stack; otherwise it proxies directly to the upstream `provider/model` id. See [Chat completions client contract](#chat-completions-client-contract) below.

To run **chimera-broker** and **chimera-vectorstore** as supervised wrappers, use `chimera serve` or make target `chimera-supervisor-run` — see [supervisor.md](supervisor.md). BiFrost/Qdrant remain the typical local backends behind those wrappers.

## Environment variables

| Variable | Required | Description |
|----------|----------|-------------|
| `CHIMERA_BROKER_API_KEY` | Yes for typical setups | Bearer token the gateway sends to chimera-broker (`broker.base_url`). Name is configurable via `broker.api_key_env` in `gateway.yaml` (default `CHIMERA_BROKER_API_KEY`). Any non-empty placeholder works when the broker does not enforce governance keys. |
| `LOG_LEVEL` | No | Log level for `log/slog`: `debug`, `info`, `warn`, `error`. Overrides `gateway.log_level` when set. |
| `CHIMERA_GATEWAY_CONFIG` | No | Path to `gateway.yaml`. Default `./config/gateway.yaml` on the host. |

Provider keys (`GROQ_API_KEY`, `GEMINI_API_KEY`, `OPENAI_API_KEY`, etc.) are **not** read by the gateway; **BiFrost** (`config/bifrost.config.json`) consumes them.

**Model listing (BiFrost):** `GET /v1/models` on BiFrost alone may return entries like `groq/*`. The gateway first calls BiFrost’s `GET /api/models?unfiltered=true&limit=500`, maps each `{ provider, name }` to an OpenAI-style id `provider/name`, then merges enabled **assistants** from operator SQLite. If that route is missing, the gateway uses `GET /v1/models` only. See `scripts/list-bifrost-models.sh`.

## `config/gateway.yaml`

| Field | Description |
|-------|-------------|
| `gateway.semver` | Gateway release version string (display / diagnostics). |
| **`gateway.listen_port` / `listen_host`** | HTTP bind address. |
| `gateway.log_level` | Suggested log level (use `LOG_LEVEL` env for a simple override). |
| `broker.base_url` | chimera-broker root URL (no trailing slash required), e.g. `http://127.0.0.1:8080`. |
| `broker.api_key_env` | Name of the process env var holding the Bearer token (default `CHIMERA_BROKER_API_KEY`). |
| `broker.log_level` | Log level for the chimera-broker wrapper subprocess (typical: `info` or `debug`). |
| `vectorstore.url` | chimera-vectorstore HTTP endpoint for RAG (default `http://127.0.0.1:6333`). Required when `rag.enabled` is **true**. |
| `vectorstore.log_level` | Storage-backend log level when chimera-vectorstore supervises the child process. |
| `vectorstore.api_key` | Optional API key for the vector store HTTP API. |
| `rag.enabled` | **true** → ingest, indexer APIs, retrieval, and `GET /health` `checks.vectorstore`. |
| `rag.embedding.*` | Embedding model/path/dim; `base_url` defaults to `broker.base_url` when empty. |
| `rag.chunking.*` / `rag.retrieval.*` / `rag.ingest.*` / `rag.defaults.*` | Chunking, search, ingest limits, and default project/flavor ids. |
| `rag.coherence.mode` | Stale-index behavior: `off`, `warn` (default), or `strict` (blocks `/v1/rag/*` expansion when stale). |
| `rag.tooling.enabled` | **true** (default) → `/v1/rag/segments`, `/context`, `/adjacent`, `/tools`. |
| `rag.tooling.expansion_cache_*` | LRU TTL (default 300s) and max entries (default 256) for merged context slices. |
| `health.upstream_url` | Optional explicit URL for `GET /health` broker probe; default `{broker.base_url}/health`. Deprecated alias: `health.litellm_url`. |
| `health.timeout_ms` | Timeout for the upstream health request and for `GET /v1/models` upstream list (default **5000**). |
| `health.chat_timeout_ms` | Timeout for each upstream `POST /v1/chat/completions` attempt (default **300000**). |
| `paths.api_keys` | Path to `api-keys.yaml` (relative to `gateway.yaml`’s directory unless absolute). |
| `paths.provider_free_tier` | Path to `provider-free-tier.yaml` (default `./provider-free-tier.yaml` next to `gateway.yaml`). |
| `paths.provider_model_limits` | Path to `provider-model-limits.yaml` (default `./provider-model-limits.yaml` next to `gateway.yaml`). Missing or empty file means **no enforcement**; invalid file is logged and the gateway starts with an empty spec. See `docs/plans/version-v0.1.1.md` §3.7. **Quota enforcement on chat** compares limits to live usage in the metrics DB — keep `metrics.enabled: true` (default) or limits are not applied even when the YAML is populated. |
| `metrics.enabled` | Default **true**. When **false**, the gateway does **not** open SQLite metrics or record upstream chat outcomes. |
| `metrics.sqlite_path` | SQLite database file for gateway metrics (relative to **`gateway.yaml`’s directory** unless absolute). Default `../data/gateway/metrics.sqlite`. |
| `metrics.migrations_dir` | Directory containing `NNNNNN_description.sql` migration files (default `../migrations/chimera-gateway/metrics`). Migrations run **once at startup**; see `docs/plans/version-v0.1.1.md` §3.6. |

Reload: change file and **save** (mtime update). On reload, if token paths change, those stores are re-opened.

### Internal embedding (`internal_embedding`)

| Key | Purpose |
|-----|---------|
| `internal_embedding.enabled` | **true** → supervise the local embedding backend and override the resolved `search.embedding` base URL, model, and dimension; default **false**. |
| `internal_embedding.provider` / `model` / `dim` | Internal embedding identity and expected output dimension. Defaults: `internal`, `nomic-embed-text`, `768`. |
| `internal_embedding.base_url` | Direct OpenAI-compatible llama-server root URL. Default `http://127.0.0.1:8090`. |
| `internal_embedding.model_path` | Operator-provisioned GGUF model path; relative paths resolve from the directory containing `chimera.yaml`. Required when enabled. |
| `internal_embedding.cache_dir` | llama-server working/cache directory; relative paths resolve from the directory containing `chimera.yaml`. |

Enabling internal embedding changes the resolved gateway RAG target immediately on config load, but process lifecycle is supervisor-owned: restart the supervised stack after changing `internal_embedding` settings. Porcelain does not download `llama-server` or model weights; supply the backend with `-embed-backend-bin` (or place it beside the supervisor/on `PATH`) and provision the GGUF path. See [Internal embedding provider](features/internal-embedding-provider.md).

### Chat routing (assistants)

Fallback chains, policy rules, tool router, and harness modules are configured **per assistant** in operator SQLite via `/ui/settings` assistant cards — not in `gateway.yaml`. See [Operator assistants](features/operator-assistants.md) and [Gateway chat routing pipeline](features/gateway-chat-routing-pipeline.md).

| Client `body.model` value | Gateway behavior |
|---------------------------|------------------|
| Assistant id (explicit operator name, e.g. `Research-1.0`) | Resolve assistant registry → run harness stages (retrieval, tools, evaluator, escalation) inside one HTTP turn; walk fallback on 429/selected 5xx |
| Upstream id (e.g. `groq/llama-3.1-8b-instant`) | Direct proxy to chimera-broker; **no** harness, fallback walk, or workspace tool injection |

Fresh installs have **zero** assistants until the operator creates them in settings.

### Chat completions client contract

Integrators call `POST /v1/chat/completions` with the gateway Bearer token from `api-keys.yaml` (same as OpenAI-style clients). The gateway owns orchestration: the client sends **one user turn per HTTP request** and does **not** run tool loops, evaluator passes, or escalation hops — those run internally when `body.model` is an assistant id.

**Related behavior (not duplicated here):** [Operator chat UI](features/operator-chat-ui.md), [Gateway chat routing pipeline](features/gateway-chat-routing-pipeline.md), [Gateway RAG ingest and retrieval](features/gateway-rag-ingest-and-retrieval.md), [Operator conversation history](features/operator-conversation-history.md), [Operator assistants](features/operator-assistants.md).

#### Request body

| Field | Contract |
|-------|----------|
| `model` | **Assistant path:** operator-named assistant id (must match an enabled row in the assistant registry). **Escape hatch:** `provider/model` broker catalog id → plain upstream relay, no harness. |
| `messages` | Standard chat roles; gateway may prepend retrieval/system content on the assistant path. |
| `stream` | Client preference for SSE streaming. On the assistant path, per-assistant evaluator **`stream_policy`** may override `true` (see [Streaming](#streaming) below). |
| `tools` | **Assistant path:** when `tool_executor` is enabled, the gateway **injects** workspace tool declarations and **ignores** client `tools`. When the module is off, behavior follows the routing stack (tool router may slim client `tools` if configured). **Direct upstream path:** optional `tools` array is forwarded (tool router may slim it). Chimera `/ui/chat` does not send `tools`. |

#### Request headers (chat)

| Header | Purpose |
|--------|---------|
| `X-Chimera-Conversation-Id` | Stable thread id for history and logs. Omit to let the gateway generate one; response echoes the effective id. |
| `X-Chimera-Project` | RAG collection scope (tenant + project + flavor). |
| `X-Chimera-Flavor-Id` | RAG scope and **workspace policy** lookup (with project). |
| `X-Chimera-Workspace-Id` | Operator SQLite **workspace row id** for conversation metadata and history snapshot — **not** the primary policy/RAG scope key (use project + flavor for that). Chimera chat sends the selected workspace row id. |
| `X-Chimera-Tool-Router` | Set to `skip` to disable tool slimming for this request (direct upstream / integrators with large `tools` lists). |
| `X-Chimera-Tool-Confidence-Threshold` | Optional per-request override (0–1) for tool-router confidence floor when slimming applies. |

#### Response headers (chat)

| Header | Purpose |
|--------|---------|
| `X-Chimera-Upstream-Model` | Broker-resolved `provider/model` that produced the assistant text (after fallback/evaluator/escalation on the assistant path). |
| `X-Chimera-RAG-Hits` | Base64-encoded JSON array of retrieval snippets for the turn. Each hit includes at least `source`, `text`, and `score`; when manifest ingest applies, entries may also carry `start_line`, `end_line`, and related line-metadata fields (see [Gateway RAG ingest and retrieval](features/gateway-rag-ingest-and-retrieval.md)). |
| `X-Chimera-Conversation-Id` | Effective conversation id for the turn (echo or assign). |
| `X-Chimera-Harness-Summary` | Base64-encoded **redacted** harness `TurnEnvelope` JSON (`schema_version: 1`). Secrets and large payloads are stripped before encoding; use for integrator debugging, not as a full audit log. Persisted history stores a similar payload on each turn (`harness_summary_json`). |

#### Streaming

When the evaluator module is enabled, each assistant's `stream_policy` controls how `body.stream: true` is honored:

| `stream_policy` | Behavior (summary) |
|-----------------|-------------------|
| `immediate` | Stream primary SSE as it arrives; evaluator runs after the primary completes; escalation cannot replace an answer already streamed to the client. |
| `gate_on_evaluator` | Buffer primary output until the evaluator runs; on pass, deliver buffered content (stream or chunk); on fail, run escalation v1 before client delivery when recommended. |
| `buffer_until_complete` | No client streaming while the evaluator is enabled; deliver after eval (and any remediation) completes — often as non-stream JSON or a single SSE chunk. |

If policy conflicts with `body.stream: true`, the gateway applies the assistant policy and may log `harness.stream.policy_applied`. Details: [assistant harness evaluator plan](plans/assistant-harness-evaluator-escalation.md).

#### Chimera `/ui/chat` vs integrators

On a **live** turn, the embed client reads **`X-Chimera-Upstream-Model`**, **`X-Chimera-RAG-Hits`**, and **`X-Chimera-Conversation-Id`** from the HTTP response (including streaming responses) for footnotes and model labels. It does **not** parse **`X-Chimera-Harness-Summary`** on the live stream. **Turn details** (stage timeline) come from persisted **`harness_summary`** on history reload — see [Operator conversation history](features/operator-conversation-history.md).

#### Example request

```bash
curl -sS -X POST 'http://127.0.0.1:7720/v1/chat/completions' \
  -H 'Authorization: Bearer YOUR_API_SECRET' \
  -H 'Content-Type: application/json' \
  -H 'X-Chimera-Conversation-Id: 550e8400-e29b-41d4-a716-446655440000' \
  -H 'X-Chimera-Project: demo' \
  -H 'X-Chimera-Flavor-Id: default' \
  -H 'X-Chimera-Workspace-Id: 1' \
  -d '{
    "model": "Research-1.0",
    "messages": [{"role": "user", "content": "Summarize the indexed README."}],
    "stream": false
  }'
```

Replace host/port with `gateway.listen_host` / `gateway.listen_port`, use a real assistant id from `GET /v1/models`, and align project/flavor with an indexed workspace.

### Supervised file indexer (`indexer.supervised`)

Used by `chimera serve` and `locus-desktop`: optional supervision of `chimera-indexer` as a child process after BiFrost is healthy. The child gets `CHIMERA_GATEWAY_URL` and a single merged `--config` file (tuning only); watch paths live in **operator SQLite** and are polled via `GET /v1/indexer/workspaces`. Set `CHIMERA_GATEWAY_TOKEN` in the environment for ingest. Operator UI: `/ui/settings` Workspaces section. Optional `workspaces_poll_interval_ms` in the supervised YAML (default 30s). Legacy `roots:` in an empty database are imported once at gateway startup. Behavior: **[indexer.md](indexer.md)**, **[indexer-workspaces.md](features/indexer-workspaces.md)**; process tree: **[supervisor.md](supervisor.md)**.

| Field | Description |
|-------|-------------|
| `indexer.supervised.enabled` | **true** → start `chimera-indexer` beside the gateway binary (or `indexer.supervised.bin` / `PATH`). Ignored unless `rag.enabled` is **true** or `start_when_rag_disabled` is **true**. |
| `indexer.supervised.log_json` | Default **true** (omitted = JSON). Passes `--log-json` so the indexer writes structured logs on stderr (filter `/ui/settings` by source `indexer`). Set **false** to opt out. |
| `indexer.supervised.bin` | Optional explicit path to the `chimera-indexer` executable. Empty → resolve next to the gateway binary or `PATH`. |
| `indexer.supervised.config_path` | Path to the indexer tuning YAML passed as `--config` (default `indexer.yaml` next to `gateway.yaml`; see `config/indexer.example.yaml`). |
| `indexer.supervised.start_when_rag_disabled` | **true** → allow starting the supervised indexer when `rag.enabled` is **false** (default **false**). |

## `config/api-keys.yaml`

```yaml
api_keys:
  - label: optional-human-name
    secret: "secret-bearer-value"
    tenant_id: "tenant-slug"
```

- `secret` — must match the client’s `Authorization: Bearer` value exactly.
- `tenant_id` — carried in logs today; **v0.2+** RAG scopes by tenant.

## `config/provider-model-limits.yaml`

Operator-maintained ceilings compared against live metrics and per-request context by the gateway's admission guard (`internal/providerlimits`). See `docs/plans/version-v0.1.1.md` §3.7 for RPM/TPM schema; `docs/plans/context-window-admission.md` for context fields (schema v2). Short version:

```yaml
schema_version: 2
defaults:
  usage_day_timezone: UTC
  context_safety_factor: 0.9
  max_body_bytes: 3500000
providers:
  groq:
    usage_day_timezone: UTC  # IANA tz for rpd/tpd day boundaries (e.g. America/Los_Angeles for Gemini)
    rpm: 30
    rpd: 14400
    tpm: 6000
    models:
      groq/llama-3.3-70b-versatile:
        tpm: 12000
      groq/groq/compound-mini:
        context_window: 131072
        max_prompt_tokens: 8192
```

**Quota fields** (`rpm`, `rpd`, `tpm`, `tpd`): compared to live usage in the metrics DB — keep `metrics.enabled: true` (default) or quota limits are not applied even when the YAML is populated. On deny the gateway logs `chat.provider_limits.blocked` with `reason` set to `rpm`, `tpm`, `rpd`, or `tpd`.

**Context fields** (schema v2, no metrics I/O):

| Field | Purpose |
|-------|---------|
| `context_window` | Total context tokens (typically from catalog `context_length`) |
| `max_prompt_tokens` | Stricter prompt-only cap when vendor/catalog overstates the window |
| `max_body_bytes` | Marshalled JSON byte cap |
| `context_safety_factor` | Per-layer multiplier on the token cap (global default e.g. `0.9` in `defaults`) |

Effective token cap: `floor(min(context_window, max_prompt_tokens_if_set) × safety_factor)`. Admission compares `est_prompt_tokens + max_tokens` to that cap; body size is checked separately against `max_body_bytes`.

When `context_window` is omitted in YAML, the gateway overlays `context_length` from the live broker catalog poll (`health.available_models_poll_ms`, default ~30s) if the snapshot is fresh; YAML values always win.

**Context vs quota (TPM/RPM):**

| Dimension | Question | Data | On deny |
|-----------|----------|------|---------|
| TPM / RPM / RPD / TPD | Would minute/day **quota** be exceeded? | Metrics SQLite + YAML caps | Skip model; log `reason=tpm` (etc.) |
| Context / body | Does this **single request** fit the model window? | YAML (+ optional live catalog) | Skip model; log `reason=context_window` or `request_body_bytes` with `outgoingTokens`, `max_tokens`, `body_bytes`, `context_cap` |

Seeding: `make catalog-limits` (after `make catalog-available`) copies `context_length` from `catalog-available.snapshot.yaml` into this file without changing RPM/TPM values.

- Merge: **model > provider > defaults**. Unset fields mean **no enforcement** for that dimension at that layer; if every layer leaves a field unset, the gateway does not cap it.
- When a provider (or any of its models) sets `rpd` or `tpd`, the provider must be able to resolve an `usage_day_timezone` from its own block or `defaults` — otherwise startup rejects the file.
- Unknown fields are rejected at parse time; negative numbers, non-positive `context_safety_factor`, and invalid IANA names are rejected.
- Config-only: **reloading requires a restart** today. A copy-paste starter lives at `config/provider-model-limits.example.yaml`. Run `make catalog-limits` to seed `context_window` from `catalog-available.snapshot.yaml` (after `make catalog-available`).

## `config/provider-free-tier.yaml`

Operator-maintained allowlist of BiFrost `provider/model` ids (and optional `patterns`). See comments inside the file for `format_version`, `effective_date`, and editing rules.

**Reference snapshot (optional):** `make catalog-free` fetches [Groq rate limits](https://console.groq.com/docs/rate-limits) and [Gemini API pricing](https://ai.google.dev/gemini-api/docs/pricing), derives BiFrost-style ids, and writes `config/free-tier-catalog.snapshot.yaml` (gitignored by default). Use `INTERSECT=`_path_ to restrict lines to ids that fuzzy-match a catalog file: JSON or YAML with `data`.`id` (same shape as `GET /v1/models`, e.g. `config/catalog-available.snapshot.yaml` from `make catalog-available`). `make catalog-available` calls `GET /v1/models` on BiFrost (defaults `BIFROST_BASE_URL=http://127.0.0.1:8080`, optional `CHIMERA_BROKER_API_KEY`) and writes `config/catalog-available.snapshot.yaml`. These snapshots are **not** loaded by the gateway automatically; merge entries into `provider-free-tier.yaml` by hand if you want them enforced.

**Operator model availability:** Per-tenant model visibility is configured in the admin UI (`/ui/settings` → provider card → **Configure** under model availability), persisted in operator SQLite, and applied at runtime. `GET /v1/models` and chat routing filter by the bearer token’s `tenant_id` from `api-keys.yaml`. The UI **Apply free-tier defaults** button (Groq/Gemini) proposes toggles from this YAML file but does not save until the operator confirms.

## `config/bifrost.config.json`

BiFrost bootstrap file. Provider keys use `env.VAR` for secrets.

**Per-key `models`:** In BiFrost, an **empty** or **omitted** `models` list means the key may be used for **any** model for that provider (minus `blacklisted_models` if set). **`"models": ["*"]` is not a wildcard** — it is treated as the literal model name `*`, so chat requests for real model ids will fail with *no keys found that support model*. Use no `models` field (or `[]`) when you want full catalog access without enumerating models.

## Logging semantics

### Baseline (v0.1)

- **INFO**: each HTTP response (method, path, status, duration, redacted `Authorization` prefix).
- **INFO**: upstream chat probe summary (status, model, stream flag).
- **DEBUG**: routing rule match, config path resolution, reload events, upstream relay details.

### Correlation (v0.2.1+)

Structured logs may include `request_id` (middleware), `service`, chat `conversation_id` (header `X-Chimera-Conversation-Id` or gateway-generated), `principal_id`, and stable `msg` slugs on ingest/RAG/indexer paths. Ingest/indexer flows may carry `index_run_id`. Operator-facing breakdown: [plans/archive/log-presentation-layer.md](plans/archive/log-presentation-layer.md).
