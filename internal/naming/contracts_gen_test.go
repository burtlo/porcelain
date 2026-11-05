package naming_test

import (
	"bytes"
	"os"
	"path/filepath"
	"runtime"
	"testing"

	"github.com/lynn/porcelain/internal/naming/gencontracts"
)

func TestGeneratedContractsJSMatchesFile(t *testing.T) {
	t.Helper()
	path := filepath.Join(repoRoot(t), filepath.FromSlash(gencontracts.DefaultContractsJSPath))

	var buf bytes.Buffer
	if err := gencontracts.WriteContractsJS(&buf); err != nil {
		t.Fatal(err)
	}
	want := buf.String()

	onDisk, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read %s: %v", path, err)
	}
	if string(onDisk) != want {
		t.Fatalf("%s is stale; run: make contracts-generate", path)
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
