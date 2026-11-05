package config

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/lynn/porcelain/chimera/chimera-indexer/adapter"
)

func TestMaterializeIndexerConfig_mergesOverlay(t *testing.T) {
	dir := t.TempDir()
	overlay := filepath.Join(dir, "overlay.yaml")
	if err := os.WriteFile(overlay, []byte("workers: 8\nlog_level: debug\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	mat := filepath.Join(dir, "out", "indexer.materialized.yaml")
	res := &Resolved{
		IndexerFileConfig:       adapter.FileConfig{Workers: 2, LogLevel: "info"},
		IndexerOverlayPath:      overlay,
		IndexerMaterializedPath: mat,
	}
	path, err := MaterializeIndexerConfig(res)
	if err != nil {
		t.Fatal(err)
	}
	if path != mat {
		t.Fatalf("path %q", path)
	}
	fc, err := EffectiveIndexerFileConfig(res)
	if err != nil {
		t.Fatal(err)
	}
	if fc.Workers != 8 {
		t.Fatalf("workers want 8 got %d", fc.Workers)
	}
	if fc.LogLevel != "debug" {
		t.Fatalf("log_level %q", fc.LogLevel)
	}
	if _, err := os.Stat(mat); err != nil {
		t.Fatal(err)
	}
}
