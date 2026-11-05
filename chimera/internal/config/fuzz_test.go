package config

import (
	"os"
	"path/filepath"
	"testing"
)

func FuzzLoadChimeraYAML(f *testing.F) {
	seed := []byte(`gateway:
  semver: "0.1.0"
  auth:
    api_keys: "./api-keys.yaml"
broker:
  url: "http://127.0.0.1:8080"
`)
	f.Add(seed)
	f.Fuzz(func(t *testing.T, data []byte) {
		dir := t.TempDir()
		p := filepath.Join(dir, "chimera.yaml")
		if err := os.WriteFile(p, data, 0o644); err != nil {
			t.Fatal(err)
		}
		_, _ = LoadChimeraYAML(p, nil)
	})
}
