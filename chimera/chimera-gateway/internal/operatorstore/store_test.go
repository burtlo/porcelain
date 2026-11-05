package operatorstore

import (
	"context"
	"path/filepath"
	"testing"

	"github.com/lynn/porcelain/chimera/chimera-gateway/internal/testsupport"
)

func testMigrationsDir(t *testing.T) string {
	t.Helper()
	return testsupport.GatewayOperatorMigrationsDir(t)
}

func TestStore_CreateListWorkspace(t *testing.T) {
	dir := t.TempDir()
	dbPath := filepath.Join(dir, "operator.sqlite")
	s, err := Open(dbPath, testMigrationsDir(t), nil)
	if err != nil {
		t.Fatal(err)
	}
	defer s.Close()

	ctx := context.Background()
	w, err := s.CreateWorkspace(ctx, "", "myproj", "f1", []string{"/tmp/a", "/tmp/b"})
	if err != nil {
		t.Fatal(err)
	}
	if w.ID < 1 || len(w.Paths) != 2 {
		t.Fatalf("workspace=%+v", w)
	}

	all, err := s.ListWorkspaces(ctx, "")
	if err != nil {
		t.Fatal(err)
	}
	if len(all) != 1 || len(all[0].Paths) != 2 {
		t.Fatalf("list=%+v", all)
	}

	if err := s.DeletePath(ctx, "", w.Paths[0].ID); err != nil {
		t.Fatal(err)
	}
	all2, err := s.ListWorkspaces(ctx, "")
	if err != nil {
		t.Fatal(err)
	}
	if len(all2[0].Paths) != 1 {
		t.Fatalf("after delete path: %+v", all2)
	}

	if err := s.DeleteWorkspace(ctx, "", w.ID); err != nil {
		t.Fatal(err)
	}
	all3, err := s.ListWorkspaces(ctx, "")
	if err != nil {
		t.Fatal(err)
	}
	if len(all3) != 0 {
		t.Fatalf("want empty, got %+v", all3)
	}
}

func TestStore_ResolveWorkspaceScopeUsesLowestIDAndPolicy(t *testing.T) {
	s, err := Open(filepath.Join(t.TempDir(), "operator.sqlite"), testMigrationsDir(t), nil)
	if err != nil {
		t.Fatal(err)
	}
	defer s.Close()

	ctx := context.Background()
	first, err := s.CreateWorkspace(ctx, "tenant-a", "project-a", "flavor-a", []string{"/tmp/a"}, WorkspacePolicy{
		Sensitivity: "private", AllowCloud: false, FileActionPolicy: "read",
	})
	if err != nil {
		t.Fatal(err)
	}
	second, err := s.CreateWorkspace(ctx, "tenant-a", "project-a", "flavor-a", []string{"/tmp/b"}, WorkspacePolicy{
		Sensitivity: "public", AllowCloud: true, FileActionPolicy: "read_write",
	})
	if err != nil {
		t.Fatal(err)
	}
	got, matched, err := s.ResolveWorkspaceScope(ctx, "tenant-a", "project-a", "flavor-a")
	if err != nil {
		t.Fatal(err)
	}
	if got == nil || got.ID != first.ID || got.Sensitivity != "private" || got.AllowCloud || got.FileActionPolicy != "read" {
		t.Fatalf("resolved=%+v", got)
	}
	if len(matched) != 2 || matched[0] != first.ID || matched[1] != second.ID {
		t.Fatalf("matched=%v", matched)
	}
}
