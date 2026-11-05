package config

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/lynn/porcelain/chimera/chimera-indexer/adapter"
)

func TestWriteChimeraIndexerTuning_preservesSuiteKeys(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	p := filepath.Join(dir, "chimera.yaml")
	raw := `gateway: { listen_port: 3000 }
indexer:
  enabled: true
  config_path: "indexer.yaml"
  bin: "./chimera-indexer"
  log_json: false
  log_level: info
  workers: 1
broker: { url: "http://127.0.0.1:8080" }
`
	if err := os.WriteFile(p, []byte(raw), 0o644); err != nil {
		t.Fatal(err)
	}
	fc := adapter.FileConfig{
		LogLevel:           "debug",
		JobSkipLog:         "debug",
		JobIngestLog:       "debug",
		ScopeStatusPollMS:  -1,
		StorageStatsPollMS: -1,
		SyncStatePath:      "data/indexer/sync-state.json",
		Workers:            4,
	}
	if err := WriteChimeraIndexerTuning(p, fc); err != nil {
		t.Fatal(err)
	}
	res, err := LoadChimeraYAML(p, nil)
	if err != nil {
		t.Fatal(err)
	}
	if !res.IndexerEnabled {
		t.Fatal("enabled lost")
	}
	if res.IndexerOverlayPath == "" || !strings.HasSuffix(res.IndexerOverlayPath, "indexer.yaml") {
		t.Fatalf("config_path lost: %q", res.IndexerOverlayPath)
	}
	if res.IndexerBin != "./chimera-indexer" {
		t.Fatalf("bin=%q", res.IndexerBin)
	}
	if res.IndexerLogJSON {
		t.Fatal("log_json should stay false")
	}
	if res.IndexerFileConfig.LogLevel != "debug" || res.IndexerFileConfig.Workers != 4 {
		t.Fatalf("tuning: %+v", res.IndexerFileConfig)
	}
	if res.IndexerFileConfig.SyncStatePath != "data/indexer/sync-state.json" {
		t.Fatalf("sync_state_path=%q", res.IndexerFileConfig.SyncStatePath)
	}
	body, _ := os.ReadFile(p)
	if strings.Contains(string(body), "roots:") {
		t.Fatalf("roots should not be written: %s", body)
	}
}
