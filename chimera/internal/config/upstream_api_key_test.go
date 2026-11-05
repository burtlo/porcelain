package config

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestEnsureGeneratedUpstreamAPIKey_generatesAndPersists(t *testing.T) {
	t.Setenv("CHIMERA_BROKER_API_KEY", "")
	dir := t.TempDir()
	p := filepath.Join(dir, "chimera.yaml")
	raw := `gateway:
  semver: "0.1.0"
  listen_port: 3000
  listen_host: "127.0.0.1"
  auth:
    api_keys: "./api-keys.yaml"
broker:
  url: "http://127.0.0.1:8080"
  api_key_env: "CHIMERA_BROKER_API_KEY"
`
	if err := os.WriteFile(p, []byte(raw), 0o644); err != nil {
		t.Fatal(err)
	}
	res, err := LoadChimeraYAML(p, nil)
	if err != nil {
		t.Fatal(err)
	}
	if res.UpstreamAPIKey != "" {
		t.Fatalf("expected empty before ensure, got %q", res.UpstreamAPIKey)
	}
	out, err := EnsureGeneratedUpstreamAPIKey(p, res, nil)
	if err != nil {
		t.Fatal(err)
	}
	if strings.TrimSpace(out.UpstreamAPIKey) == "" {
		t.Fatal("expected generated key")
	}
	res2, err := LoadChimeraYAML(p, nil)
	if err != nil {
		t.Fatal(err)
	}
	if res2.UpstreamAPIKey != out.UpstreamAPIKey {
		t.Fatalf("reload mismatch: %q vs %q", res2.UpstreamAPIKey, out.UpstreamAPIKey)
	}
	b, _ := os.ReadFile(p)
	if !strings.Contains(string(b), "api_key:") {
		t.Fatalf("expected api_key in file: %s", b)
	}
}

func TestEnsureGeneratedUpstreamAPIKey_envSkipsWrite(t *testing.T) {
	t.Setenv("CHIMERA_BROKER_API_KEY", "from-env")
	dir := t.TempDir()
	p := filepath.Join(dir, "chimera.yaml")
	raw := `gateway:
  semver: "0.1.0"
  auth:
    api_keys: "./api-keys.yaml"
broker:
  url: "http://127.0.0.1:8080"
  api_key_env: "CHIMERA_BROKER_API_KEY"
`
	if err := os.WriteFile(p, []byte(raw), 0o644); err != nil {
		t.Fatal(err)
	}
	res, err := LoadChimeraYAML(p, nil)
	if err != nil {
		t.Fatal(err)
	}
	out, err := EnsureGeneratedUpstreamAPIKey(p, res, nil)
	if err != nil {
		t.Fatal(err)
	}
	if out.UpstreamAPIKey != "" {
		t.Fatalf("yaml key should stay empty when env set, got %q", out.UpstreamAPIKey)
	}
	b, _ := os.ReadFile(p)
	if strings.Contains(string(b), "api_key:") {
		t.Fatalf("did not expect api_key in file: %s", b)
	}
}

func TestLoadChimeraYAML_upstreamAPIKey(t *testing.T) {
	dir := t.TempDir()
	p := filepath.Join(dir, "chimera.yaml")
	raw := `gateway:
  semver: "0.1.0"
  auth:
    api_keys: "./api-keys.yaml"
broker:
  url: "http://x"
  api_key: "yaml-secret"
`
	if err := os.WriteFile(p, []byte(raw), 0o644); err != nil {
		t.Fatal(err)
	}
	res, err := LoadChimeraYAML(p, nil)
	if err != nil {
		t.Fatal(err)
	}
	if res.UpstreamAPIKey != "yaml-secret" {
		t.Fatalf("got %q", res.UpstreamAPIKey)
	}
}
