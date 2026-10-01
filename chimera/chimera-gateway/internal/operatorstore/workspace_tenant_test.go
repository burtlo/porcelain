package operatorstore_test

import (
	"context"
	"path/filepath"
	"testing"

	"github.com/lynn/porcelain/chimera/chimera-gateway/internal/operatorstore"
	"github.com/lynn/porcelain/chimera/chimera-gateway/internal/testsupport"
)

func TestListWorkspaces_legacyEmptyTenantVisibleForDefaultSession(t *testing.T) {
	store, err := operatorstore.Open(filepath.Join(t.TempDir(), "operator.sqlite"), testsupport.GatewayOperatorMigrationsDir(t), nil)
	if err != nil {
		t.Fatal(err)
	}
	defer store.Close()
	_, err = store.CreateWorkspace(context.Background(), "", "porcelain", "", []string{t.TempDir()}, operatorstore.WorkspacePolicy{
		FileActionPolicy: "read",
	})
	if err != nil {
		t.Fatal(err)
	}
	list, err := store.ListWorkspaces(context.Background(), "default")
	if err != nil {
		t.Fatal(err)
	}
	if len(list) != 1 || list[0].ProjectID != "porcelain" {
		t.Fatalf("list=%+v", list)
	}
}

func TestResolveWorkspaceScope_legacyEmptyTenantMatchesDefaultSession(t *testing.T) {
	store, err := operatorstore.Open(filepath.Join(t.TempDir(), "operator.sqlite"), testsupport.GatewayOperatorMigrationsDir(t), nil)
	if err != nil {
		t.Fatal(err)
	}
	defer store.Close()
	_, err = store.CreateWorkspace(context.Background(), "", "porcelain", "", []string{t.TempDir()}, operatorstore.WorkspacePolicy{
		FileActionPolicy: "read",
	})
	if err != nil {
		t.Fatal(err)
	}
	ws, ids, err := store.ResolveWorkspaceScope(context.Background(), "default", "porcelain", "")
	if err != nil {
		t.Fatal(err)
	}
	if ws == nil || len(ids) != 1 {
		t.Fatalf("ws=%+v ids=%v", ws, ids)
	}
}
