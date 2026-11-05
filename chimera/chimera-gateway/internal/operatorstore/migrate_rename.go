package operatorstore

import (
	"database/sql"
	"fmt"
)

// applyRenameVirtualModelsToAssistants upgrades legacy virtual_models schema in place.
// No-op when virtual_models is absent (fresh installs use 000002_assistants.sql).
func applyRenameVirtualModelsToAssistants(tx *sql.Tx) error {
	ok, err := tableExists(tx, "virtual_models")
	if err != nil {
		return err
	}
	if !ok {
		return nil
	}

	renames := []struct{ from, to string }{
		{"virtual_models", "assistants"},
		{"virtual_model_fallback", "assistant_fallback"},
		{"virtual_model_routing_policy", "assistant_routing_policy"},
		{"virtual_model_tool_router", "assistant_tool_router"},
		{"virtual_model_rule_bindings", "assistant_rule_bindings"},
	}
	for _, r := range renames {
		if err := renameTableIfExists(tx, r.from, r.to); err != nil {
			return err
		}
	}

	columnRenames := []struct{ table, from, to string }{
		{"assistant_fallback", "virtual_model_id", "assistant_id"},
		{"assistant_routing_policy", "virtual_model_id", "assistant_id"},
		{"assistant_tool_router", "virtual_model_id", "assistant_id"},
		{"assistant_rule_bindings", "virtual_model_id", "assistant_id"},
	}
	for _, c := range columnRenames {
		if err := renameColumnIfExists(tx, c.table, c.from, c.to); err != nil {
			return err
		}
	}

	if err := renameTableIfExists(tx, "virtual_model_harness_modules", "assistant_harness_modules"); err != nil {
		return err
	}
	if err := renameColumnIfExists(tx, "assistant_harness_modules", "virtual_model_id", "assistant_id"); err != nil {
		return err
	}

	indexMigrations := []struct {
		oldName, createSQL string
	}{
		{
			"idx_virtual_models_tenant",
			`CREATE INDEX IF NOT EXISTS idx_assistants_tenant ON assistants (tenant_id)`,
		},
		{
			"idx_virtual_models_enabled",
			`CREATE INDEX IF NOT EXISTS idx_assistants_enabled ON assistants (enabled)`,
		},
		{
			"idx_vm_rule_bindings_vm",
			`CREATE INDEX IF NOT EXISTS idx_assistant_rule_bindings_assistant ON assistant_rule_bindings (assistant_id)`,
		},
		{
			"idx_vm_harness_modules_vm",
			`CREATE INDEX IF NOT EXISTS idx_assistant_harness_modules_assistant ON assistant_harness_modules (assistant_id)`,
		},
	}
	for _, idx := range indexMigrations {
		if err := replaceIndexIfExists(tx, idx.oldName, idx.createSQL); err != nil {
			return err
		}
	}
	return nil
}

func tableExists(tx *sql.Tx, name string) (bool, error) {
	var n int
	err := tx.QueryRow(
		`SELECT COUNT(1) FROM sqlite_master WHERE type='table' AND name=?`,
		name,
	).Scan(&n)
	return n > 0, err
}

func renameTableIfExists(tx *sql.Tx, from, to string) error {
	ok, err := tableExists(tx, from)
	if err != nil {
		return err
	}
	if !ok {
		return nil
	}
	if _, err := tx.Exec(fmt.Sprintf(`ALTER TABLE %s RENAME TO %s`, from, to)); err != nil {
		return fmt.Errorf("rename table %s to %s: %w", from, to, err)
	}
	return nil
}

func columnExists(tx *sql.Tx, table, column string) (bool, error) {
	rows, err := tx.Query(`PRAGMA table_info(` + table + `)`)
	if err != nil {
		return false, err
	}
	defer rows.Close()
	for rows.Next() {
		var cid int
		var name, ctype string
		var notnull, pk int
		var dflt sql.NullString
		if err := rows.Scan(&cid, &name, &ctype, &notnull, &dflt, &pk); err != nil {
			return false, err
		}
		if name == column {
			return true, nil
		}
	}
	return false, rows.Err()
}

func renameColumnIfExists(tx *sql.Tx, table, from, to string) error {
	ok, err := tableExists(tx, table)
	if err != nil {
		return err
	}
	if !ok {
		return nil
	}
	hasCol, err := columnExists(tx, table, from)
	if err != nil {
		return err
	}
	if !hasCol {
		return nil
	}
	q := fmt.Sprintf(`ALTER TABLE %s RENAME COLUMN %s TO %s`, table, from, to)
	if _, err := tx.Exec(q); err != nil {
		return fmt.Errorf("rename column %s.%s to %s: %w", table, from, to, err)
	}
	return nil
}

func indexExists(tx *sql.Tx, name string) (bool, error) {
	var n int
	err := tx.QueryRow(
		`SELECT COUNT(1) FROM sqlite_master WHERE type='index' AND name=?`,
		name,
	).Scan(&n)
	return n > 0, err
}

func replaceIndexIfExists(tx *sql.Tx, oldName, createSQL string) error {
	ok, err := indexExists(tx, oldName)
	if err != nil {
		return err
	}
	if !ok {
		return nil
	}
	if _, err := tx.Exec(`DROP INDEX ` + oldName); err != nil {
		return fmt.Errorf("drop index %s: %w", oldName, err)
	}
	if _, err := tx.Exec(createSQL); err != nil {
		return fmt.Errorf("create replacement index for %s: %w", oldName, err)
	}
	return nil
}
