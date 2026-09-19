package operatorstore

import (
	"context"
	"testing"
)

func assertAssistantsSchema(t *testing.T, s *Store) {
	t.Helper()
	ctx := context.Background()
	expectedTables := []string{
		"assistants",
		"assistant_fallback",
		"assistant_routing_policy",
		"assistant_tool_router",
		"assistant_harness_modules",
	}
	for _, name := range expectedTables {
		var n int
		err := s.db.QueryRowContext(ctx, `
SELECT COUNT(*) FROM sqlite_master WHERE type='table' AND name = ?`, name).Scan(&n)
		if err != nil {
			t.Fatal(err)
		}
		if n != 1 {
			t.Fatalf("missing table %s", name)
		}
	}
	var legacy int
	if err := s.db.QueryRowContext(ctx, `
SELECT COUNT(*) FROM sqlite_master WHERE type='table' AND name LIKE 'virtual_model%'`).Scan(&legacy); err != nil {
		t.Fatal(err)
	}
	if legacy != 0 {
		t.Fatalf("legacy virtual_model* tables remain: count=%d", legacy)
	}
}

func TestAssistantsSchema_FreshMigrations_UseAssistantTables(t *testing.T) {
	s := openTestStore(t)
	assertAssistantsSchema(t, s)
}
