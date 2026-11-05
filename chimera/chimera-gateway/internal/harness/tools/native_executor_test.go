package tools

import (
	"context"
	"os"
	"path/filepath"
	"testing"
)

func TestNativeExecutor_reservedNamesInDeclarations(t *testing.T) {
	for _, name := range []string{"workspace_context_around", "workspace_adjacent_chunks", "workspace_read_lines"} {
		if !IsReservedNativeName(name) {
			t.Fatalf("missing reserved name %q", name)
		}
	}
	decls := OpenAIToolDeclarations(true)
	if len(decls) != 7 {
		t.Fatalf("want 7 decls, got %d", len(decls))
	}
	declsBase := OpenAIToolDeclarations(false)
	if len(declsBase) != 4 {
		t.Fatalf("want 4 base decls, got %d", len(declsBase))
	}
}

func TestNativeExecutor_readLinesFallsBackToWorkspaceWithoutHash(t *testing.T) {
	root := t.TempDir()
	readme := filepath.Join(root, "README.md")
	if err := os.WriteFile(readme, []byte("line one\nline two\nline three\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	n := &NativeExecutor{Workspace: NewWorkspaceExecutor([]string{root}, PolicyRead)}
	got, err := n.Invoke(context.Background(), ToolCall{
		Name:      "workspace_read_lines",
		Arguments: []byte(`{"source":"README.md","start_line":1,"end_line":1}`),
	})
	if err != nil || got.IsError {
		t.Fatalf("err=%v result=%+v", err, got)
	}
	if got.Content != "line one" {
		t.Fatalf("content=%q", got.Content)
	}
}

func TestCompositeNativeWinsExpansionName(t *testing.T) {
	ws := NewWorkspaceExecutor(nil, PolicyRead)
	comp := &CompositeExecutor{Native: &NativeExecutor{Workspace: ws, Expansion: nil}}
	args := []byte(`{"point_id":"p1","radius":1}`)
	got, _ := comp.Invoke(context.Background(), ToolCall{Name: "workspace_adjacent_chunks", Arguments: args})
	if !got.IsError || got.Content != "rag expansion tools unavailable" {
		t.Fatalf("expected expansion unavailable, got %+v", got)
	}
}
