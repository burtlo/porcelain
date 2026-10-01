package operatorstore_test

import (
	"context"
	"path/filepath"
	"testing"

	"github.com/lynn/porcelain/chimera/chimera-gateway/internal/operatorstore"
	"github.com/lynn/porcelain/chimera/chimera-gateway/internal/testsupport"
)

func TestResolveWorkspaceForHarness_byRowID(t *testing.T) {
	store, err := operatorstore.Open(filepath.Join(t.TempDir(), "operator.sqlite"), testsupport.GatewayOperatorMigrationsDir(t), nil)
	if err != nil {
		t.Fatal(err)
	}
	defer store.Close()
	ws, err := store.CreateWorkspace(context.Background(), "default", "porcelain", "", []string{t.TempDir()}, operatorstore.WorkspacePolicy{
		FileActionPolicy: "read",
	})
	if err != nil {
		t.Fatal(err)
	}
	got, ids, err := store.ResolveWorkspaceForHarness(context.Background(), "default", "default", "", ws.ID)
	if err != nil {
		t.Fatal(err)
	}
	if got == nil || got.ID != ws.ID || len(ids) != 1 {
		t.Fatalf("got=%+v ids=%v", got, ids)
	}
}

func TestResolveWorkspaceForHarness_projectFallback(t *testing.T) {
	store, err := operatorstore.Open(filepath.Join(t.TempDir(), "operator.sqlite"), testsupport.GatewayOperatorMigrationsDir(t), nil)
	if err != nil {
		t.Fatal(err)
	}
	defer store.Close()
	ws, err := store.CreateWorkspace(context.Background(), "default", "porcelain", "dev", []string{t.TempDir()}, operatorstore.WorkspacePolicy{
		FileActionPolicy: "read",
	})
	if err != nil {
		t.Fatal(err)
	}
	got, _, err := store.ResolveWorkspaceForHarness(context.Background(), "default", "porcelain", "other-flavor", 0)
	if err != nil {
		t.Fatal(err)
	}
	if got == nil || got.ID != ws.ID {
		t.Fatalf("got=%+v", got)
	}
}
