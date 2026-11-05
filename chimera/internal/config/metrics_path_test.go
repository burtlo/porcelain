package config

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestLoadChimeraYAML_metricsPaths(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	gw := filepath.Join(dir, "config", "chimera.yaml")
	if err := os.MkdirAll(filepath.Dir(gw), 0o755); err != nil {
		t.Fatal(err)
	}
	raw := strings.TrimSpace(`
gateway:
  listen_port: 3000
  auth:
    api_keys: "./api-keys.yaml"
`)
	if err := os.WriteFile(gw, []byte(raw+"\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	res, err := LoadChimeraYAML(gw, nil)
	if err != nil {
		t.Fatal(err)
	}
	if !res.MetricsEnabled {
		t.Fatal("metrics should default enabled")
	}
	wantDB := filepath.Join(dir, "data", "gateway", "metrics.sqlite")
	if res.MetricsSQLitePath != wantDB {
		t.Fatalf("MetricsSQLitePath=%q want %q", res.MetricsSQLitePath, wantDB)
	}
	wantMig := filepath.Join(dir, "migrations", "chimera-gateway", "metrics")
	if res.MetricsMigrationsDir != wantMig {
		t.Fatalf("MetricsMigrationsDir=%q want %q", res.MetricsMigrationsDir, wantMig)
	}
}

func TestLoadChimeraYAML_metricsDisabled(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	gw := filepath.Join(dir, "chimera.yaml")
	raw := `gateway:
  listen_port: 3000
  auth: { api_keys: "./t.yaml" }
  metrics:
    enabled: false
`
	if err := os.WriteFile(gw, []byte(raw), 0o644); err != nil {
		t.Fatal(err)
	}
	res, err := LoadChimeraYAML(gw, nil)
	if err != nil {
		t.Fatal(err)
	}
	if res.MetricsEnabled {
		t.Fatal("expected metrics disabled")
	}
}
