package operatorstore

import (
	"context"
	"database/sql"
	"io"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"testing"
	"time"

	"github.com/lynn/porcelain/chimera/chimera-gateway/internal/testsupport"
	_ "modernc.org/sqlite"
)

var migrationVersionRE = regexp.MustCompile(`^(\d{6})_.+\.sql$`)

func copyOperatorMigrationsUpTo(t *testing.T, maxVersion int) string {
	t.Helper()
	src := testsupport.GatewayOperatorMigrationsDir(t)
	dst := filepath.Join(t.TempDir(), "migrations")
	if err := os.MkdirAll(dst, 0o755); err != nil {
		t.Fatal(err)
	}
	entries, err := os.ReadDir(src)
	if err != nil {
		t.Fatal(err)
	}
	for _, e := range entries {
		if e.IsDir() {
			continue
		}
		name := e.Name()
		sub := migrationVersionRE.FindStringSubmatch(name)
		if len(sub) != 2 {
			continue
		}
		v, err := strconv.Atoi(sub[1])
		if err != nil || v > maxVersion {
			continue
		}
		copyMigrationFile(t, filepath.Join(src, name), filepath.Join(dst, name))
	}
	return dst
}

func copyMigrationFile(t *testing.T, src, dst string) {
	t.Helper()
	in, err := os.Open(src)
	if err != nil {
		t.Fatal(err)
	}
	defer in.Close()
	out, err := os.Create(dst)
	if err != nil {
		t.Fatal(err)
	}
	defer out.Close()
	if _, err := io.Copy(out, in); err != nil {
		t.Fatal(err)
	}
}

func seedVirtualModelAtMigration010(t *testing.T, db *sql.DB) int64 {
	t.Helper()
	now := time.Now().UTC().Format(time.RFC3339Nano)
	res, err := db.Exec(`
INSERT INTO virtual_models (model_id, name, version, description, enabled, visibility,
	created_by_principal_id, tenant_id, created_at, updated_at)
VALUES (?,?,?,?,?,?,?,?,?,?)`,
		"Legacy-1.0.0", "Legacy", "1.0.0", "", 1, "public", "", "", now, now)
	if err != nil {
		t.Fatal(err)
	}
	id, err := res.LastInsertId()
	if err != nil {
		t.Fatal(err)
	}
	if _, err := db.Exec(`INSERT INTO virtual_model_fallback (virtual_model_id, chain_json, updated_at) VALUES (?,?,?)`,
		id, `["groq/a","groq/b"]`, now); err != nil {
		t.Fatal(err)
	}
	if _, err := db.Exec(`INSERT INTO virtual_model_routing_policy (virtual_model_id, enabled, policy_yaml, updated_at) VALUES (?,?,?,?)`,
		id, 0, "", now); err != nil {
		t.Fatal(err)
	}
	if _, err := db.Exec(`
INSERT INTO virtual_model_tool_router (virtual_model_id, enabled, router_models_json, confidence_threshold, updated_at)
VALUES (?,?,?,?,?)`, id, 0, "[]", 0.5, now); err != nil {
		t.Fatal(err)
	}
	if _, err := db.Exec(`
INSERT INTO virtual_model_harness_modules (virtual_model_id, module_id, enabled, config_json, updated_at)
VALUES (?,?,?,?,?)`, id, HarnessModuleRetrieval, 1, "{}", now); err != nil {
		t.Fatal(err)
	}
	return id
}

func TestMigration000011_AssistantsRename_UpgradeFrom000010(t *testing.T) {
	dir := t.TempDir()
	dbPath := filepath.Join(dir, "operator.sqlite")
	mig010 := copyOperatorMigrationsUpTo(t, 10)
	fullMig := testsupport.GatewayOperatorMigrationsDir(t)

	db, err := sql.Open("sqlite", dbPath)
	if err != nil {
		t.Fatal(err)
	}
	if err := ApplyMigrations(db, mig010, nil); err != nil {
		t.Fatal(err)
	}
	seedVirtualModelAtMigration010(t, db)

	body, err := os.ReadFile(filepath.Join(fullMig, "000011_assistants.sql"))
	if err != nil {
		t.Fatal(err)
	}
	tx, err := db.Begin()
	if err != nil {
		t.Fatal(err)
	}
	if _, err := tx.Exec(string(body)); err != nil {
		_ = tx.Rollback()
		t.Fatalf("apply 000011: %v", err)
	}
	if _, err := tx.Exec(`INSERT INTO operator_migrations (version) VALUES (?)`, 11); err != nil {
		_ = tx.Rollback()
		t.Fatal(err)
	}
	if err := tx.Commit(); err != nil {
		t.Fatal(err)
	}
	if err := db.Close(); err != nil {
		t.Fatal(err)
	}

	s, err := Open(dbPath, fullMig, nil)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = s.Close() })

	ctx := context.Background()
	got, err := s.GetVirtualModelByModelID(ctx, "Legacy-1.0.0")
	if err != nil || got == nil {
		t.Fatalf("get after upgrade: err=%v got=%+v", err, got)
	}
	if got.ModelID != "Legacy-1.0.0" || len(got.FallbackChain) != 2 {
		t.Fatalf("data not preserved: %+v", got)
	}
	if !got.HarnessModuleEnabled(HarnessModuleRetrieval) {
		t.Fatalf("harness modules not preserved: %+v", got.HarnessModules)
	}

	var oldTableCount int
	if err := s.db.QueryRowContext(ctx, `
SELECT COUNT(*) FROM sqlite_master WHERE type='table' AND name LIKE 'virtual_model%'`).Scan(&oldTableCount); err != nil {
		t.Fatal(err)
	}
	if oldTableCount != 0 {
		t.Fatalf("legacy virtual_model* tables remain: count=%d", oldTableCount)
	}

	assertAssistantsSchema(t, s)

	var assistantID int64
	if err := s.db.QueryRowContext(ctx, `SELECT assistant_id FROM assistant_fallback LIMIT 1`).Scan(&assistantID); err != nil {
		t.Fatal(err)
	}
	if assistantID <= 0 {
		t.Fatalf("assistant_id column not populated: %d", assistantID)
	}
	var rowCount int
	if err := s.db.QueryRowContext(ctx, `SELECT COUNT(*) FROM assistants WHERE model_id = ?`, "Legacy-1.0.0").Scan(&rowCount); err != nil {
		t.Fatal(err)
	}
	if rowCount != 1 {
		t.Fatalf("assistants row count=%d", rowCount)
	}
}
