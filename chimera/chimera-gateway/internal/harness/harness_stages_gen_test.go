package harness_test

import (
	"bytes"
	"os"
	"path/filepath"
	"runtime"
	"testing"

	"github.com/lynn/porcelain/chimera/chimera-gateway/internal/harness"
	"github.com/lynn/porcelain/chimera/chimera-gateway/internal/harness/gendocs"
)

func TestGeneratedHarnessStagesMarkdownMatchesFile(t *testing.T) {
	t.Helper()
	path := filepath.Join(repoRoot(t), filepath.FromSlash(gendocs.DefaultHarnessStagesPath))

	var buf bytes.Buffer
	if err := gendocs.WriteHarnessStagesMarkdown(&buf); err != nil {
		t.Fatal(err)
	}
	want := buf.String()

	onDisk, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read %s: %v", path, err)
	}
	if string(onDisk) != want {
		t.Fatalf("%s is stale; run: make harness-docs-generate", path)
	}
}

func TestDefaultRunnerStageOrder(t *testing.T) {
	got := stageNames(harness.DefaultRunner())
	want := []string{
		"stack_resolve",
		"meta_policy",
		"intent",
		"tool_router",
		"tool_executor",
		"retrieval",
		"request_witness",
		"initial_pick",
		"fallback_proxy",
	}
	if len(got) != len(want) {
		t.Fatalf("stage count: got %d want %d (%v)", len(got), len(want), got)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("stage[%d]: got %q want %q (full=%v)", i, got[i], want[i], got)
		}
	}
}

func TestPrePrimaryRunnerStageOrder(t *testing.T) {
	got := stageNames(harness.PrePrimaryRunner())
	want := []string{"stack_resolve", "meta_policy", "intent"}
	if len(got) != len(want) {
		t.Fatalf("stage count: got %d want %d (%v)", len(got), len(want), got)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("stage[%d]: got %q want %q", i, got[i], want[i])
		}
	}
}

func stageNames(r *harness.Runner) []string {
	meta := r.Stages()
	out := make([]string, len(meta))
	for i, m := range meta {
		out[i] = m.Name
	}
	return out
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
