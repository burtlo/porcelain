package tools

import (
	"context"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func call(t *testing.T, name string, args any) ToolCall {
	t.Helper()
	raw, err := json.Marshal(args)
	if err != nil {
		t.Fatal(err)
	}
	return ToolCall{ID: "call_1", Name: name, Arguments: raw}
}

func TestWorkspaceExecutorReadWriteAndRelativePaths(t *testing.T) {
	root := t.TempDir()
	exec := NewWorkspaceExecutor([]string{root}, PolicyReadWrite)
	got, err := exec.Invoke(context.Background(), call(t, "write_file", map[string]any{"path": "notes/todo.txt", "content": "ship tools"}))
	if err != nil || got.IsError {
		t.Fatalf("write: result=%+v err=%v", got, err)
	}
	data, err := os.ReadFile(filepath.Join(root, "notes", "todo.txt"))
	if err != nil || string(data) != "ship tools" {
		t.Fatalf("written file = %q, %v", data, err)
	}
	got, err = exec.Invoke(context.Background(), call(t, "list_dir", map[string]any{"path": "notes"}))
	if err != nil || got.IsError || got.Content != "notes/todo.txt" {
		t.Fatalf("list: result=%+v err=%v", got, err)
	}
}

func TestWorkspaceExecutorRejectsTraversalAndReadOnlyWrite(t *testing.T) {
	root := t.TempDir()
	exec := NewWorkspaceExecutor([]string{root}, PolicyRead)
	got, err := exec.Invoke(context.Background(), call(t, "write_file", map[string]any{"path": "x.txt", "content": "no"}))
	if !errors.Is(err, ErrPermission) || !got.IsError {
		t.Fatalf("read-only write: result=%+v err=%v", got, err)
	}
	got, err = exec.Invoke(context.Background(), call(t, "read_file", map[string]any{"path": "../secret.txt"}))
	if !errors.Is(err, ErrPath) || !got.IsError {
		t.Fatalf("traversal: result=%+v err=%v", got, err)
	}
}

func TestWorkspaceExecutorRejectsEscapingSymlink(t *testing.T) {
	root, outside := t.TempDir(), t.TempDir()
	if err := os.WriteFile(filepath.Join(outside, "secret.txt"), []byte("secret"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(outside, filepath.Join(root, "escape")); err != nil {
		t.Skipf("symlink unavailable: %v", err)
	}
	got, err := NewWorkspaceExecutor([]string{root}, PolicyRead).Invoke(context.Background(), call(t, "read_file", map[string]any{"path": "escape/secret.txt"}))
	if !errors.Is(err, ErrPath) || !got.IsError || strings.Contains(got.Content, "secret") {
		t.Fatalf("symlink escape: result=%+v err=%v", got, err)
	}
}
