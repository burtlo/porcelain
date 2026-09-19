-- Per-virtual-model harness module toggles and config (v0.4 turn harness).

CREATE TABLE IF NOT EXISTS virtual_model_harness_modules (
	virtual_model_id INTEGER NOT NULL REFERENCES virtual_models (id) ON DELETE CASCADE,
	module_id TEXT NOT NULL,
	enabled INTEGER NOT NULL DEFAULT 0,
	config_json TEXT NOT NULL DEFAULT '{}',
	updated_at TEXT NOT NULL,
	PRIMARY KEY (virtual_model_id, module_id)
);

CREATE INDEX IF NOT EXISTS idx_vm_harness_modules_vm ON virtual_model_harness_modules (virtual_model_id);
