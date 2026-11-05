package routes_test

import (
	"bytes"
	"os"
	"path/filepath"
	"runtime"
	"testing"

	"github.com/lynn/porcelain/internal/docgen/routes"
)

func TestGeneratedGatewayHTTPRoutesMatchesFile(t *testing.T) {
	t.Helper()
	root := repoRoot(t)
	path := filepath.Join(root, filepath.FromSlash(routes.DefaultMarkdownPath))

	collected, err := routes.Collect(root)
	if err != nil {
		t.Fatal(err)
	}

	var buf bytes.Buffer
	if err := routes.WriteMarkdown(&buf, collected); err != nil {
		t.Fatal(err)
	}
	want := buf.String()

	onDisk, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read %s: %v (run: make contracts-generate)", path, err)
	}
	if string(onDisk) != want {
		t.Fatalf("%s is stale; run: make contracts-generate", routes.DefaultMarkdownPath)
	}
}

func repoRoot(t *testing.T) string {
	t.Helper()
	_, file, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("runtime.Caller failed")
	}
	dir := filepath.Dir(file)
	for {
		if _, err := os.Stat(filepath.Join(dir, "go.mod")); err == nil {
			return dir
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			t.Fatal("go.mod not found")
		}
		dir = parent
	}
}
