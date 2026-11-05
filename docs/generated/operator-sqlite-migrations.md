# Operator SQLite migrations (generated)

DO NOT EDIT; run: `make contracts-generate`.

Gateway applies versioned SQL under `migrations/chimera-gateway/` at startup. The **operator store** (`operator.sqlite`, table `operator_migrations`) and **gateway metrics** DB (`gateway_migrations`) use separate directories and migrators.

## Operator migrator notes

From `internal/operatorstore/migrate.go` (`//docgen:migrations`):

Operator SQL files live under migrations/chimera-gateway/operator/.
Filenames must match migrationFileRE: 000NNN_snake_name.sql (six-digit version prefix).
Pending files run in lexicographic order; applied versions are stored in operator_migrations.
Version 11 (000011_rename_virtual_models_to_assistants.sql): the .sql file is comment-only;
ApplyMigrations runs applyRenameVirtualModelsToAssistants when legacy virtual_models exists (see migrate_rename.go).

## Operator store

- **SQL directory:** `migrations/chimera-gateway/operator`
- **Applied versions table:** `operator_migrations`
- **Go:** `chimera/chimera-gateway/internal/operatorstore/migrate.go`

| Version | File | Summary |
|---------|------|--------|
| 1 | `000001_workspaces.sql` | Operator SQLite: indexer workspaces + conversation merge state. Applied by internal/operatorstore; add new numbered files only. |
| 2 | `000002_assistants.sql` | Operator SQLite: assistants and per-model routing stacks. |
| 3 | `000003_provider_model_availability.sql` | Operator SQLite: per-tenant provider model availability (operator-curated catalog filter). |
| 4 | `000004_conversation_history.sql` | Operator conversation history (transcripts, not lifecycle logs). |
| 5 | `000005_drop_conversation_merge.sql` | Remove semantic conversation merge state (feature removed). |
| 6 | `000006_reindex_and_corpus_segments.sql` | Operator SQLite: workspace re-index intent + corpus segment index for manifest ingest. |
| 7 | `000007_manifest_retrieval_lines.sql` | Manifest ingest: line ranges on saved RAG hits (no backfill for legacy rows). |
| 8 | `000008_harness_turn_summary.sql` | Per-turn harness envelope summary (redacted JSON) on assistant/error rows. |
| 9 | `000009_assistant_harness_modules.sql` | Per-assistant harness module toggles and config (v0.4 turn harness). |
| 10 | `000010_workspace_policy.sql` | Workspace policy used by the assistant harness meta-policy stage. |
| 11 | `000011_rename_virtual_models_to_assistants.sql` | Upgrade path for operator DBs created before the virtual-model → assistant rename. |
| 12 | `000012_mcp_servers.sql` | Operator MCP server definitions and per-assistant bindings (v0.5 Phase 2c). |

### Version 1 — `000001_workspaces.sql`

<details>
<summary>SQL</summary>

```sql
-- Operator SQLite: indexer workspaces + conversation merge state. Applied by internal/operatorstore; add new numbered files only.

CREATE TABLE IF NOT EXISTS workspaces (
	id INTEGER PRIMARY KEY AUTOINCREMENT,
	tenant_id TEXT NOT NULL DEFAULT '',
	project_id TEXT NOT NULL,
	flavor_id TEXT NOT NULL DEFAULT '',
	created_at TEXT NOT NULL,
	updated_at TEXT NOT NULL
);

CREATE INDEX IF NOT EXISTS idx_workspaces_tenant ON workspaces (tenant_id);

CREATE TABLE IF NOT EXISTS workspace_paths (
	id INTEGER PRIMARY KEY AUTOINCREMENT,
	workspace_row_id INTEGER NOT NULL REFERENCES workspaces (id) ON DELETE CASCADE,
	path TEXT NOT NULL,
	created_at TEXT NOT NULL,
	updated_at TEXT NOT NULL
);

CREATE INDEX IF NOT EXISTS idx_workspace_paths_workspace ON workspace_paths (workspace_row_id);

-- Semantic conversation merge + rolling fingerprint state (moved from metrics DB).
CREATE TABLE IF NOT EXISTS conversation_context (
	conversation_id TEXT NOT NULL PRIMARY KEY,
	tenant_id TEXT NOT NULL,
	project_id TEXT NOT NULL DEFAULT '',
	flavor_id TEXT NOT NULL DEFAULT '',
	last_user_embedding BLOB NOT NULL,
	embedding_dim INTEGER NOT NULL,
	last_user_text_normalized TEXT NOT NULL DEFAULT '',
	last_model_text_normalized TEXT NOT NULL DEFAULT '',
	last_updated_unix REAL NOT NULL,
	rolling_fingerprint TEXT NOT NULL DEFAULT ''
);

CREATE INDEX IF NOT EXISTS idx_conversation_context_scope_time
	ON conversation_context (tenant_id, project_id, flavor_id, last_updated_unix DESC);

-- Short-lived JSON completion cache for duplicate HTTP retries (same scope + fingerprint + user text).
CREATE TABLE IF NOT EXISTS conversation_dedup_cache (
	dedup_key TEXT NOT NULL PRIMARY KEY,
	response_body BLOB NOT NULL,
	created_unix REAL NOT NULL
);

CREATE INDEX IF NOT EXISTS idx_conversation_dedup_created ON conversation_dedup_cache (created_unix);
```

</details>

### Version 2 — `000002_assistants.sql`

<details>
<summary>SQL</summary>

```sql
-- Operator SQLite: assistants and per-model routing stacks.

CREATE TABLE IF NOT EXISTS assistants (
	id INTEGER PRIMARY KEY AUTOINCREMENT,
	model_id TEXT NOT NULL UNIQUE,
	name TEXT NOT NULL,
	version TEXT NOT NULL,
	description TEXT NOT NULL DEFAULT '',
	enabled INTEGER NOT NULL DEFAULT 1,
	visibility TEXT NOT NULL DEFAULT 'public',
	created_by_principal_id TEXT NOT NULL DEFAULT '',
	tenant_id TEXT NOT NULL DEFAULT '',
	created_at TEXT NOT NULL,
	updated_at TEXT NOT NULL
);

CREATE INDEX IF NOT EXISTS idx_assistants_tenant ON assistants (tenant_id);
CREATE INDEX IF NOT EXISTS idx_assistants_enabled ON assistants (enabled);

CREATE TABLE IF NOT EXISTS routing_rule_definitions (
	id INTEGER PRIMARY KEY AUTOINCREMENT,
	name TEXT NOT NULL,
	slug TEXT NOT NULL UNIQUE,
	default_config_json TEXT NOT NULL DEFAULT '{}',
	description TEXT NOT NULL DEFAULT '',
	created_at TEXT NOT NULL,
	updated_at TEXT NOT NULL
);

CREATE TABLE IF NOT EXISTS assistant_fallback (
	assistant_id INTEGER NOT NULL PRIMARY KEY REFERENCES assistants (id) ON DELETE CASCADE,
	chain_json TEXT NOT NULL,
	updated_at TEXT NOT NULL
);

CREATE TABLE IF NOT EXISTS assistant_routing_policy (
	assistant_id INTEGER NOT NULL PRIMARY KEY REFERENCES assistants (id) ON DELETE CASCADE,
	enabled INTEGER NOT NULL DEFAULT 1,
	policy_yaml TEXT NOT NULL DEFAULT '',
	updated_at TEXT NOT NULL
);

CREATE TABLE IF NOT EXISTS assistant_tool_router (
	assistant_id INTEGER NOT NULL PRIMARY KEY REFERENCES assistants (id) ON DELETE CASCADE,
	enabled INTEGER NOT NULL DEFAULT 0,
	router_models_json TEXT NOT NULL DEFAULT '[]',
	confidence_threshold REAL NOT NULL DEFAULT 0.5,
	updated_at TEXT NOT NULL
);

CREATE TABLE IF NOT EXISTS assistant_rule_bindings (
	id INTEGER PRIMARY KEY AUTOINCREMENT,
	assistant_id INTEGER NOT NULL REFERENCES assistants (id) ON DELETE CASCADE,
	routing_rule_definition_id INTEGER NOT NULL REFERENCES routing_rule_definitions (id) ON DELETE CASCADE,
	enabled INTEGER NOT NULL DEFAULT 1,
	override_config_json TEXT NOT NULL DEFAULT '{}',
	sort_order INTEGER NOT NULL DEFAULT 0,
	UNIQUE (assistant_id, routing_rule_definition_id)
);

CREATE INDEX IF NOT EXISTS idx_assistant_rule_bindings_assistant ON assistant_rule_bindings (assistant_id);
```

</details>

### Version 3 — `000003_provider_model_availability.sql`

<details>
<summary>SQL</summary>

```sql
-- Operator SQLite: per-tenant provider model availability (operator-curated catalog filter).

CREATE TABLE IF NOT EXISTS provider_model_config (
	tenant_id TEXT NOT NULL,
	provider_id TEXT NOT NULL,
	updated_at TEXT NOT NULL,
	metadata_json TEXT NOT NULL DEFAULT '{}',
	PRIMARY KEY (tenant_id, provider_id)
);

CREATE TABLE IF NOT EXISTS provider_model_availability (
	tenant_id TEXT NOT NULL,
	provider_id TEXT NOT NULL,
	model_id TEXT NOT NULL,
	available INTEGER NOT NULL,
	metadata_json TEXT NOT NULL DEFAULT '{}',
	updated_at TEXT NOT NULL,
	PRIMARY KEY (tenant_id, provider_id, model_id)
);

CREATE INDEX IF NOT EXISTS idx_provider_model_avail_tenant ON provider_model_availability (tenant_id);
CREATE INDEX IF NOT EXISTS idx_provider_model_avail_provider ON provider_model_availability (tenant_id, provider_id);
```

</details>

### Version 4 — `000004_conversation_history.sql`

<details>
<summary>SQL</summary>

```sql
-- Operator conversation history (transcripts, not lifecycle logs).

CREATE TABLE IF NOT EXISTS conversations (
	conversation_id TEXT NOT NULL PRIMARY KEY,
	principal_id TEXT NOT NULL,
	title TEXT NULL,
	preview_text TEXT NOT NULL DEFAULT '',
	flagged INTEGER NOT NULL DEFAULT 0 CHECK (flagged IN (0, 1)),
	workspace_project_id TEXT NOT NULL DEFAULT '',
	workspace_flavor_id TEXT NOT NULL DEFAULT '',
	workspace_row_id INTEGER NULL,
	created_at TEXT NOT NULL,
	updated_at TEXT NOT NULL
);

CREATE TABLE IF NOT EXISTS conversation_turns (
	turn_id TEXT NOT NULL PRIMARY KEY,
	conversation_id TEXT NOT NULL REFERENCES conversations (conversation_id) ON DELETE CASCADE,
	turn_index INTEGER NOT NULL,
	role TEXT NOT NULL CHECK (role IN ('user', 'assistant', 'error')),
	content TEXT NOT NULL,
	selected_model TEXT NOT NULL DEFAULT '',
	resolved_model TEXT NOT NULL DEFAULT '',
	error_detail TEXT NOT NULL DEFAULT '',
	retry_user_text TEXT NOT NULL DEFAULT '',
	prompt_tokens INTEGER NULL,
	completion_tokens INTEGER NULL,
	total_tokens INTEGER NULL,
	created_at TEXT NOT NULL
);

CREATE TABLE IF NOT EXISTS conversation_retrievals (
	retrieval_id TEXT NOT NULL PRIMARY KEY,
	turn_id TEXT NOT NULL REFERENCES conversation_turns (turn_id) ON DELETE CASCADE,
	sort_order INTEGER NOT NULL,
	file_path TEXT NOT NULL DEFAULT '',
	score REAL NOT NULL DEFAULT 0,
	snippet_text TEXT NOT NULL DEFAULT '',
	language TEXT NOT NULL DEFAULT '',
	vector_point_id TEXT NOT NULL DEFAULT '',
	content_sha256 TEXT NOT NULL DEFAULT ''
);

CREATE INDEX IF NOT EXISTS idx_conversations_principal_updated
	ON conversations (principal_id, updated_at DESC);

CREATE INDEX IF NOT EXISTS idx_conversations_principal_flagged
	ON conversations (principal_id, flagged, updated_at DESC);

CREATE INDEX IF NOT EXISTS idx_conversation_turns_conversation
	ON conversation_turns (conversation_id, turn_index, role);

CREATE INDEX IF NOT EXISTS idx_conversation_retrievals_turn
	ON conversation_retrievals (turn_id, sort_order);
```

</details>

### Version 5 — `000005_drop_conversation_merge.sql`

<details>
<summary>SQL</summary>

```sql
-- Remove semantic conversation merge state (feature removed).

DROP INDEX IF EXISTS idx_conversation_dedup_created;
DROP TABLE IF EXISTS conversation_dedup_cache;
DROP INDEX IF EXISTS idx_conversation_context_scope_time;
DROP TABLE IF EXISTS conversation_context;
```

</details>

### Version 6 — `000006_reindex_and_corpus_segments.sql`

<details>
<summary>SQL</summary>

```sql
-- Operator SQLite: workspace re-index intent + corpus segment index for manifest ingest.

ALTER TABLE workspaces ADD COLUMN reindex_generation INTEGER NOT NULL DEFAULT 0;

CREATE TABLE IF NOT EXISTS corpus_segments (
	segment_id TEXT PRIMARY KEY,
	tenant_id TEXT NOT NULL,
	project_id TEXT NOT NULL,
	flavor_id TEXT NOT NULL,
	source TEXT NOT NULL,
	content_sha256 TEXT NOT NULL,
	chunk_index INTEGER NOT NULL,
	chunk_count INTEGER NOT NULL,
	start_line INTEGER NOT NULL,
	end_line INTEGER NOT NULL,
	start_byte INTEGER NOT NULL,
	end_byte INTEGER NOT NULL,
	start_ch INTEGER NOT NULL,
	end_ch INTEGER NOT NULL,
	starts_mid_line INTEGER NOT NULL DEFAULT 0,
	vector_point_id TEXT NOT NULL,
	language TEXT,
	created_at TEXT NOT NULL,
	UNIQUE (tenant_id, project_id, flavor_id, source, content_sha256, chunk_index)
);

CREATE INDEX IF NOT EXISTS idx_corpus_segments_lookup
	ON corpus_segments (tenant_id, project_id, flavor_id, source, content_sha256);

CREATE INDEX IF NOT EXISTS idx_corpus_segments_line
	ON corpus_segments (tenant_id, project_id, flavor_id, source, start_line);
```

</details>

### Version 7 — `000007_manifest_retrieval_lines.sql`

<details>
<summary>SQL</summary>

```sql
-- Manifest ingest: line ranges on saved RAG hits (no backfill for legacy rows).

ALTER TABLE conversation_retrievals ADD COLUMN start_line INTEGER NOT NULL DEFAULT 0;
ALTER TABLE conversation_retrievals ADD COLUMN end_line INTEGER NOT NULL DEFAULT 0;
ALTER TABLE conversation_retrievals ADD COLUMN starts_mid_line INTEGER NOT NULL DEFAULT 0 CHECK (starts_mid_line IN (0, 1));
```

</details>

### Version 8 — `000008_harness_turn_summary.sql`

<details>
<summary>SQL</summary>

```sql
-- Per-turn harness envelope summary (redacted JSON) on assistant/error rows.

ALTER TABLE conversation_turns ADD COLUMN harness_summary_json TEXT NOT NULL DEFAULT '';
```

</details>

### Version 9 — `000009_assistant_harness_modules.sql`

<details>
<summary>SQL</summary>

```sql
-- Per-assistant harness module toggles and config (v0.4 turn harness).

CREATE TABLE IF NOT EXISTS assistant_harness_modules (
	assistant_id INTEGER NOT NULL REFERENCES assistants (id) ON DELETE CASCADE,
	module_id TEXT NOT NULL,
	enabled INTEGER NOT NULL DEFAULT 0,
	config_json TEXT NOT NULL DEFAULT '{}',
	updated_at TEXT NOT NULL,
	PRIMARY KEY (assistant_id, module_id)
);

CREATE INDEX IF NOT EXISTS idx_assistant_harness_modules_assistant ON assistant_harness_modules (assistant_id);
```

</details>

### Version 10 — `000010_workspace_policy.sql`

<details>
<summary>SQL</summary>

```sql
-- Workspace policy used by the assistant harness meta-policy stage.
ALTER TABLE workspaces ADD COLUMN sensitivity TEXT NOT NULL DEFAULT 'internal';
ALTER TABLE workspaces ADD COLUMN allow_cloud INTEGER NOT NULL DEFAULT 1;
ALTER TABLE workspaces ADD COLUMN allow_cloud_summary_only INTEGER NOT NULL DEFAULT 0;
ALTER TABLE workspaces ADD COLUMN file_action_policy TEXT NOT NULL DEFAULT 'none';
```

</details>

### Version 11 — `000011_rename_virtual_models_to_assistants.sql`

**Runtime:** `ApplyMigrations` does **not** execute this file as raw SQL; it runs `applyRenameVirtualModelsToAssistants` when legacy `virtual_models` exists (see `migrate_rename.go`). The `.sql` file documents the upgrade path.

<details>
<summary>SQL</summary>

```sql
-- Upgrade path for operator DBs created before the virtual-model → assistant rename.
--
-- Fresh installs already have assistants/* tables from 000002_assistants.sql; SQLite
-- lacks IF NOT EXISTS for RENAME TABLE, so version 11 is applied conditionally in Go
-- (operatorstore.applyRenameVirtualModelsToAssistants) when virtual_models exists.
--
-- When the legacy table is present, the migration performs:
--   virtual_models → assistants
--   virtual_model_fallback → assistant_fallback (virtual_model_id → assistant_id)
--   virtual_model_routing_policy → assistant_routing_policy
--   virtual_model_tool_router → assistant_tool_router
--   virtual_model_rule_bindings → assistant_rule_bindings
--   virtual_model_harness_modules → assistant_harness_modules (when present)
-- and recreates indexes under assistant_* names where legacy idx_virtual_* / idx_vm_* exist.
```

</details>

### Version 12 — `000012_mcp_servers.sql`

<details>
<summary>SQL</summary>

```sql
-- Operator MCP server definitions and per-assistant bindings (v0.5 Phase 2c).

CREATE TABLE IF NOT EXISTS mcp_servers (
	server_id TEXT NOT NULL,
	tenant_id TEXT NOT NULL DEFAULT '',
	disabled INTEGER NOT NULL DEFAULT 0,
	auto_start INTEGER NOT NULL DEFAULT 0,
	transport TEXT NOT NULL DEFAULT 'stdio',
	command TEXT NOT NULL DEFAULT '',
	args_json TEXT NOT NULL DEFAULT '[]',
	env_allowlist_json TEXT NOT NULL DEFAULT '[]',
	url TEXT NOT NULL DEFAULT '',
	created_at TEXT NOT NULL,
	updated_at TEXT NOT NULL,
	PRIMARY KEY (tenant_id, server_id)
);

CREATE INDEX IF NOT EXISTS idx_mcp_servers_tenant ON mcp_servers (tenant_id);

CREATE TABLE IF NOT EXISTS mcp_assistant_server_bindings (
	assistant_id INTEGER NOT NULL REFERENCES assistants (id) ON DELETE CASCADE,
	server_id TEXT NOT NULL,
	tenant_id TEXT NOT NULL DEFAULT '',
	enabled INTEGER NOT NULL DEFAULT 1,
	updated_at TEXT NOT NULL,
	PRIMARY KEY (assistant_id, server_id)
);

CREATE INDEX IF NOT EXISTS idx_mcp_assistant_server_bindings_assistant ON mcp_assistant_server_bindings (assistant_id);

CREATE TABLE IF NOT EXISTS mcp_assistant_tool_permissions (
	assistant_id INTEGER NOT NULL REFERENCES assistants (id) ON DELETE CASCADE,
	server_id TEXT NOT NULL,
	tool_name TEXT NOT NULL,
	tenant_id TEXT NOT NULL DEFAULT '',
	enabled INTEGER NOT NULL DEFAULT 0,
	updated_at TEXT NOT NULL,
	PRIMARY KEY (assistant_id, server_id, tool_name)
);

CREATE INDEX IF NOT EXISTS idx_mcp_assistant_tool_permissions_assistant ON mcp_assistant_tool_permissions (assistant_id);
```

</details>

## Gateway metrics

- **SQL directory:** `migrations/chimera-gateway/metrics`
- **Applied versions table:** `gateway_migrations`
- **Go:** `chimera/chimera-gateway/internal/gatewaymetrics/migrate.go`

| Version | File | Summary |
|---------|------|--------|
| 1 | `000001_init.sql` | Gateway metrics schema (G6). Applied by internal/gatewaymetrics migrator; do not edit history in place — add new numbered files. |
| 2 | `000002_broker_metrics_rename.sql` | Rename upstream_* metrics tables to broker_* (hard-cut vocabulary). |

### Version 1 — `000001_init.sql`

<details>
<summary>SQL</summary>

```sql
-- Gateway metrics schema (G6). Applied by internal/gatewaymetrics migrator; do not edit history in place — add new numbered files.

CREATE TABLE IF NOT EXISTS upstream_rollup_minute (
  provider TEXT NOT NULL,
  model_id TEXT NOT NULL,
  minute_utc TEXT NOT NULL,
  status INTEGER NOT NULL,
  calls INTEGER NOT NULL DEFAULT 0,
  est_tokens INTEGER NOT NULL DEFAULT 0,
  PRIMARY KEY (provider, model_id, minute_utc, status)
);

CREATE TABLE IF NOT EXISTS upstream_rollup_day (
  provider TEXT NOT NULL,
  model_id TEXT NOT NULL,
  day_utc TEXT NOT NULL,
  status INTEGER NOT NULL,
  calls INTEGER NOT NULL DEFAULT 0,
  est_tokens INTEGER NOT NULL DEFAULT 0,
  PRIMARY KEY (provider, model_id, day_utc, status)
);

CREATE TABLE IF NOT EXISTS upstream_call_events (
  id INTEGER PRIMARY KEY AUTOINCREMENT,
  occurred_at TEXT NOT NULL,
  provider TEXT NOT NULL,
  model_id TEXT NOT NULL,
  status INTEGER NOT NULL,
  est_tokens INTEGER NOT NULL
);

CREATE INDEX IF NOT EXISTS idx_upstream_call_events_time ON upstream_call_events (occurred_at);
```

</details>

### Version 2 — `000002_broker_metrics_rename.sql`

<details>
<summary>SQL</summary>

```sql
-- Rename upstream_* metrics tables to broker_* (hard-cut vocabulary).

ALTER TABLE upstream_rollup_minute RENAME TO broker_rollup_minute;
ALTER TABLE upstream_rollup_day RENAME TO broker_rollup_day;
ALTER TABLE upstream_call_events RENAME TO broker_call_events;
DROP INDEX IF EXISTS idx_upstream_call_events_time;
CREATE INDEX IF NOT EXISTS idx_broker_call_events_time ON broker_call_events (occurred_at);
```

</details>

