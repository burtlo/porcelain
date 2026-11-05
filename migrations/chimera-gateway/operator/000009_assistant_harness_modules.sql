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
