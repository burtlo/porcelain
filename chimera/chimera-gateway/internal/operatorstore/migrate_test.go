package operatorstore

import (
	"database/sql"
	"path/filepath"
	"testing"

	"github.com/lynn/porcelain/chimera/chimera-gateway/internal/testsupport"
	_ "modernc.org/sqlite"
)

const legacyVirtualModelsDDL = `-- legacy 000002_virtual_models.sql (pre-rename)
CREATE TABLE IF NOT EXISTS virtual_models (
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
CREATE INDEX IF NOT EXISTS idx_virtual_models_tenant ON virtual_models (tenant_id);
CREATE INDEX IF NOT EXISTS idx_virtual_models_enabled ON virtual_models (enabled);
CREATE TABLE IF NOT EXISTS routing_rule_definitions (
	id INTEGER PRIMARY KEY AUTOINCREMENT,
	name TEXT NOT NULL,
	slug TEXT NOT NULL UNIQUE,
	default_config_json TEXT NOT NULL DEFAULT '{}',
	description TEXT NOT NULL DEFAULT '',
	created_at TEXT NOT NULL,
	updated_at TEXT NOT NULL
);
CREATE TABLE IF NOT EXISTS virtual_model_fallback (
	virtual_model_id INTEGER NOT NULL PRIMARY KEY REFERENCES virtual_models (id) ON DELETE CASCADE,
	chain_json TEXT NOT NULL,
	updated_at TEXT NOT NULL
);
CREATE TABLE IF NOT EXISTS virtual_model_routing_policy (
	virtual_model_id INTEGER NOT NULL PRIMARY KEY REFERENCES virtual_models (id) ON DELETE CASCADE,
	enabled INTEGER NOT NULL DEFAULT 1,
	policy_yaml TEXT NOT NULL DEFAULT '',
	updated_at TEXT NOT NULL
);
CREATE TABLE IF NOT EXISTS virtual_model_tool_router (
	virtual_model_id INTEGER NOT NULL PRIMARY KEY REFERENCES virtual_models (id) ON DELETE CASCADE,
	enabled INTEGER NOT NULL DEFAULT 0,
	router_models_json TEXT NOT NULL DEFAULT '[]',
	confidence_threshold REAL NOT NULL DEFAULT 0.5,
	updated_at TEXT NOT NULL
);
CREATE TABLE IF NOT EXISTS virtual_model_rule_bindings (
	id INTEGER PRIMARY KEY AUTOINCREMENT,
	virtual_model_id INTEGER NOT NULL REFERENCES virtual_models (id) ON DELETE CASCADE,
	routing_rule_definition_id INTEGER NOT NULL REFERENCES routing_rule_definitions (id) ON DELETE CASCADE,
	enabled INTEGER NOT NULL DEFAULT 1,
	override_config_json TEXT NOT NULL DEFAULT '{}',
	sort_order INTEGER NOT NULL DEFAULT 0,
	UNIQUE (virtual_model_id, routing_rule_definition_id)
);
CREATE INDEX IF NOT EXISTS idx_vm_rule_bindings_vm ON virtual_model_rule_bindings (virtual_model_id);
CREATE TABLE IF NOT EXISTS virtual_model_harness_modules (
	virtual_model_id INTEGER NOT NULL REFERENCES virtual_models (id) ON DELETE CASCADE,
	module_id TEXT NOT NULL,
	enabled INTEGER NOT NULL DEFAULT 0,
	config_json TEXT NOT NULL DEFAULT '{}',
	updated_at TEXT NOT NULL,
	PRIMARY KEY (virtual_model_id, module_id)
);
CREATE INDEX IF NOT EXISTS idx_vm_harness_modules_vm ON virtual_model_harness_modules (virtual_model_id);
`

func TestApplyMigrations_freshInstallUsesAssistantsSchema(t *testing.T) {
	dir := t.TempDir()
	dbPath := filepath.Join(dir, "operator.sqlite")
	migDir := testsupport.GatewayOperatorMigrationsDir(t)

	db, err := sql.Open("sqlite", dbPath)
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()

	if err := ApplyMigrations(db, migDir, nil); err != nil {
		t.Fatal(err)
	}
	assertTable(t, db, "assistants", true)
	assertTable(t, db, "virtual_models", false)
}

func TestApplyMigrations_renamesLegacyVirtualModels(t *testing.T) {
	dir := t.TempDir()
	dbPath := filepath.Join(dir, "operator.sqlite")
	migDir := testsupport.GatewayOperatorMigrationsDir(t)

	db, err := sql.Open("sqlite", dbPath)
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()

	if _, err := db.Exec(bootstrapDDL); err != nil {
		t.Fatal(err)
	}
	if _, err := db.Exec(legacyVirtualModelsDDL); err != nil {
		t.Fatal(err)
	}
	for _, v := range []int{1, 2, 3, 4, 5, 6, 7, 8, 9, 10} {
		if _, err := db.Exec(`INSERT INTO operator_migrations (version) VALUES (?)`, v); err != nil {
			t.Fatal(err)
		}
	}

	if err := ApplyMigrations(db, migDir, nil); err != nil {
		t.Fatal(err)
	}

	assertTable(t, db, "assistants", true)
	assertTable(t, db, "virtual_models", false)
	assertTable(t, db, "assistant_fallback", true)
	assertTable(t, db, "assistant_harness_modules", true)
	hasCol, err := columnExistsOnDB(db, "assistant_fallback", "assistant_id")
	if err != nil {
		t.Fatal(err)
	}
	if !hasCol {
		t.Fatal("assistant_fallback.assistant_id missing after rename migration")
	}
	hasLegacyCol, err := columnExistsOnDB(db, "assistant_fallback", "virtual_model_id")
	if err != nil {
		t.Fatal(err)
	}
	if hasLegacyCol {
		t.Fatal("assistant_fallback still has virtual_model_id column")
	}
	idx, err := indexExistsOnDB(db, "idx_assistants_tenant")
	if err != nil {
		t.Fatal(err)
	}
	if !idx {
		t.Fatal("expected idx_assistants_tenant after rename migration")
	}
}

func assertTable(t *testing.T, db *sql.DB, name string, want bool) {
	t.Helper()
	ok, err := tableExistsDB(db, name)
	if err != nil {
		t.Fatal(err)
	}
	if ok != want {
		t.Fatalf("table %q exists=%v want %v", name, ok, want)
	}
}

func tableExistsDB(db *sql.DB, name string) (bool, error) {
	var n int
	err := db.QueryRow(
		`SELECT COUNT(1) FROM sqlite_master WHERE type='table' AND name=?`,
		name,
	).Scan(&n)
	return n > 0, err
}

func columnExistsOnDB(db *sql.DB, table, column string) (bool, error) {
	rows, err := db.Query(`PRAGMA table_info(` + table + `)`)
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

func indexExistsOnDB(db *sql.DB, name string) (bool, error) {
	var n int
	err := db.QueryRow(
		`SELECT COUNT(1) FROM sqlite_master WHERE type='index' AND name=?`,
		name,
	).Scan(&n)
	return n > 0, err
}
