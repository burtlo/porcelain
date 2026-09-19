-- Rename virtual model operator tables/columns to assistants (ASST-002 work item 1).
-- Preserves model_id values (Name-Version OpenAI-compatible identifiers).

ALTER TABLE virtual_models RENAME TO assistants;

DROP INDEX IF EXISTS idx_virtual_models_tenant;
DROP INDEX IF EXISTS idx_virtual_models_enabled;
CREATE INDEX IF NOT EXISTS idx_assistants_tenant ON assistants (tenant_id);
CREATE INDEX IF NOT EXISTS idx_assistants_enabled ON assistants (enabled);

ALTER TABLE virtual_model_fallback RENAME TO assistant_fallback;
ALTER TABLE assistant_fallback RENAME COLUMN virtual_model_id TO assistant_id;

ALTER TABLE virtual_model_routing_policy RENAME TO assistant_routing_policy;
ALTER TABLE assistant_routing_policy RENAME COLUMN virtual_model_id TO assistant_id;

ALTER TABLE virtual_model_tool_router RENAME TO assistant_tool_router;
ALTER TABLE assistant_tool_router RENAME COLUMN virtual_model_id TO assistant_id;

ALTER TABLE virtual_model_rule_bindings RENAME TO assistant_rule_bindings;
ALTER TABLE assistant_rule_bindings RENAME COLUMN virtual_model_id TO assistant_id;
DROP INDEX IF EXISTS idx_vm_rule_bindings_vm;
CREATE INDEX IF NOT EXISTS idx_assistant_rule_bindings_assistant ON assistant_rule_bindings (assistant_id);

ALTER TABLE virtual_model_harness_modules RENAME TO assistant_harness_modules;
ALTER TABLE assistant_harness_modules RENAME COLUMN virtual_model_id TO assistant_id;
DROP INDEX IF EXISTS idx_vm_harness_modules_vm;
CREATE INDEX IF NOT EXISTS idx_assistant_harness_modules_assistant ON assistant_harness_modules (assistant_id);
