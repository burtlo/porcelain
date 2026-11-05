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
