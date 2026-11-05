package operatorcopy_test

import (
	"bytes"
	"os"
	"path/filepath"
	"testing"

	"github.com/lynn/porcelain/internal/operatorcopy"
	"github.com/lynn/porcelain/internal/operatorcopy/genslugindex"
)

func TestGeneratedLogSlugIndexMarkdownMatchesFile(t *testing.T) {
	reg, err := operatorcopy.LoadEmbedded()
	if err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(repoRoot(t), filepath.FromSlash(genslugindex.DefaultLogSlugIndexPath))

	var buf bytes.Buffer
	if err := genslugindex.WriteLogSlugIndexMarkdown(&buf, reg); err != nil {
		t.Fatal(err)
	}
	want := buf.String()

	onDisk, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read %s: %v", path, err)
	}
	if string(onDisk) != want {
		t.Fatalf("%s is stale; run: go generate ./internal/operatorcopy/...", path)
	}
}
