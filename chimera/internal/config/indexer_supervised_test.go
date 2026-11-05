package config

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestLoadChimeraYAML_IndexerOverlay(t *testing.T) {
	dir := t.TempDir()
	gw := filepath.Join(dir, "chimera.yaml")
	raw := []byte(`gateway:
  listen_port: 3000
  auth: { api_keys: "./t.yaml" }
vectorstore:
  url: "http://127.0.0.1:6333"
search:
  enabled: true
  embedding:
    model: "text-embedding-3-small"
    dim: 1536
indexer:
  enabled: true
  config_path: "./idx/custom.yaml"
`)
	if err := os.WriteFile(gw, raw, 0o644); err != nil {
		t.Fatal(err)
	}
	res, err := LoadChimeraYAML(gw, nil)
	if err != nil {
		t.Fatal(err)
	}
	if !res.IndexerEnabled || !res.IndexerSupervisedEnabled {
		t.Fatal("expected indexer enabled")
	}
	if !res.IndexerSupervisedLogJSON {
		t.Fatal("expected log_json default true")
	}
	if res.IndexerSupervisedStartWhenRAGDisabled {
		t.Fatal("start_when_rag_disabled is ignored (always false)")
	}
	wantOverlay := filepath.Join(dir, "idx", "custom.yaml")
	if res.IndexerOverlayPath != wantOverlay {
		t.Fatalf("overlay path: got %q want %q", res.IndexerOverlayPath, wantOverlay)
	}
	if !strings.HasSuffix(res.IndexerMaterializedPath, "indexer.materialized.yaml") {
		t.Fatalf("materialized path: %q", res.IndexerMaterializedPath)
	}
	if res.IndexerSupervisedConfigPath != res.IndexerMaterializedPath {
		t.Fatalf("supervised config path should be materialized: %q", res.IndexerSupervisedConfigPath)
	}
}

func TestLoadChimeraYAML_IndexerInline(t *testing.T) {
	dir := t.TempDir()
	gw := filepath.Join(dir, "chimera.yaml")
	raw := []byte(`gateway: { listen_port: 3000, log_level: info }
broker: { url: "http://127.0.0.1:8080" }
vectorstore: { url: "http://127.0.0.1:6333", log_level: debug }
indexer:
  enabled: true
  log_level: debug
  job_skip_log: debug
  workers: 2
search:
  enabled: true
  embedding: { model: "text-embedding-3-small", dim: 1536 }
supervisor:
  log_level: info
  services: [vectorstore, broker, gateway, indexer]
`)
	if err := os.WriteFile(gw, raw, 0o644); err != nil {
		t.Fatal(err)
	}
	res, err := LoadChimeraYAML(gw, nil)
	if err != nil {
		t.Fatal(err)
	}
	if res.IndexerFileConfig.Workers != 2 {
		t.Fatalf("workers: %d", res.IndexerFileConfig.Workers)
	}
	if res.IndexerFileConfig.LogLevel != "debug" {
		t.Fatalf("log_level: %q", res.IndexerFileConfig.LogLevel)
	}
	if !res.RAG.Enabled {
		t.Fatal("expected search enabled")
	}
	if res.SupervisorLogLevel != "info" {
		t.Fatalf("supervisor log: %q", res.SupervisorLogLevel)
	}
	if res.RAG.QdrantLogLevel != "debug" {
		t.Fatalf("vectorstore log: %q", res.RAG.QdrantLogLevel)
	}
	list, err := res.ResolveSupervisorServices()
	if err != nil {
		t.Fatal(err)
	}
	if len(list) != 4 {
		t.Fatalf("services: %v", list)
	}
}

func TestLoadChimeraYAML_Indexer_logJSONDefaultTrue(t *testing.T) {
	dir := t.TempDir()
	gw := filepath.Join(dir, "chimera.yaml")
	raw := []byte(`gateway:
  listen_port: 3000
  auth: { api_keys: "./t.yaml" }
indexer:
  enabled: true
`)
	if err := os.WriteFile(gw, raw, 0o644); err != nil {
		t.Fatal(err)
	}
	res, err := LoadChimeraYAML(gw, nil)
	if err != nil {
		t.Fatal(err)
	}
	if !res.IndexerSupervisedLogJSON {
		t.Fatal("expected log_json default true when omitted")
	}
}

func TestLoadChimeraYAML_Indexer_logJSONOptOut(t *testing.T) {
	dir := t.TempDir()
	gw := filepath.Join(dir, "chimera.yaml")
	raw := []byte(`gateway:
  listen_port: 3000
  auth: { api_keys: "./t.yaml" }
indexer:
  enabled: true
  log_json: false
`)
	if err := os.WriteFile(gw, raw, 0o644); err != nil {
		t.Fatal(err)
	}
	res, err := LoadChimeraYAML(gw, nil)
	if err != nil {
		t.Fatal(err)
	}
	if res.IndexerSupervisedLogJSON {
		t.Fatal("expected log_json false when set")
	}
}

func TestResolveSupervisorServices_rejectsDisabled(t *testing.T) {
	dir := t.TempDir()
	gw := filepath.Join(dir, "chimera.yaml")
	raw := []byte(`gateway: { enabled: true, listen_port: 3000 }
broker: { enabled: true, url: "http://127.0.0.1:8080" }
vectorstore: { enabled: false, url: "http://127.0.0.1:6333" }
indexer: { enabled: true }
supervisor:
  services: [vectorstore, broker, gateway]
`)
	if err := os.WriteFile(gw, raw, 0o644); err != nil {
		t.Fatal(err)
	}
	_, err := LoadChimeraYAML(gw, nil)
	if err == nil {
		t.Fatal("expected error for disabled vectorstore in services list")
	}
}
