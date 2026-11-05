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
