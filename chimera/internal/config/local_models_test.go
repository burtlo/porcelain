package config

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func writeChimeraYAML(t *testing.T, body string) string {
	t.Helper()
	dir := t.TempDir()
	p := filepath.Join(dir, "chimera.yaml")
	if err := os.WriteFile(p, []byte(body), 0o600); err != nil {
		t.Fatal(err)
	}
	return p
}

func TestLocalModels_ResolveDefaultsAndLaunchList(t *testing.T) {
	p := writeChimeraYAML(t, `
local_models:
  models_dir: "../models"
  models:
    - id: qwen35b
      endpoint: "127.0.0.1:8181"
      model: "chat/Qwen3.6-35B-A3B-UD-IQ3_XXS.gguf"
      alias: "qwen3.6-35b-a3b"
      ctx_size: 65536
      extra_args: ["-fa", "on"]
    - id: qwen9b
      enabled: false
      endpoint: "127.0.0.1:8180"
      model: "C:/abs/Qwen3.5-9B.gguf"
`)
	res, err := LoadChimeraYAML(p, nil)
	if err != nil {
		t.Fatal(err)
	}
	ms := res.LocalModels.Models
	if len(ms) != 2 {
		t.Fatalf("models = %d", len(ms))
	}
	m := ms[0]
	if m.ProviderName() != "local-qwen35b" || m.ModelRef() != "local-qwen35b/qwen3.6-35b-a3b" {
		t.Fatalf("provider/ref = %q %q", m.ProviderName(), m.ModelRef())
	}
	if m.Listen != "127.0.0.1:7770" || ms[1].Listen != "127.0.0.1:7771" {
		t.Fatalf("listen defaults = %q %q", m.Listen, ms[1].Listen)
	}
	wantModel := filepath.Clean(filepath.Join(filepath.Dir(p), "..", "models", "chat", "Qwen3.6-35B-A3B-UD-IQ3_XXS.gguf"))
	if m.ModelPath != wantModel {
		t.Fatalf("model path = %q want %q", m.ModelPath, wantModel)
	}
	if m.NGPULayers != 99 || m.Parallel != 1 || m.Thinking || !m.WaitReady || !m.Enabled {
		t.Fatalf("defaults wrong: %+v", m)
	}
	if ms[1].Enabled {
		t.Fatal("second model should be disabled")
	}
	if got := len(res.LocalModels.Enabled()); got != 1 {
		t.Fatalf("enabled = %d", got)
	}
	launch, err := res.ResolveSupervisorServices()
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(strings.Join(launch, ","), ServiceLocalModels) {
		t.Fatalf("default launch list %v lacks %s", launch, ServiceLocalModels)
	}
}

func TestLocalModels_Validate(t *testing.T) {
	cases := map[string]string{
		"bad id":         `[{id: "Qwen 35", endpoint: "127.0.0.1:8181", model: "a.gguf"}]`,
		"dup id":         `[{id: a, endpoint: "127.0.0.1:8181", model: "a.gguf"}, {id: a, endpoint: "127.0.0.1:8182", model: "b.gguf"}]`,
		"no model":       `[{id: a, endpoint: "127.0.0.1:8181"}]`,
		"bad endpoint":   `[{id: a, endpoint: "8181", model: "a.gguf"}]`,
		"port collision": `[{id: a, endpoint: "127.0.0.1:8181", model: "a.gguf"}, {id: b, endpoint: "127.0.0.1:8181", model: "b.gguf"}]`,
	}
	for name, models := range cases {
		t.Run(name, func(t *testing.T) {
			p := writeChimeraYAML(t, "local_models:\n  models: "+models+"\n")
			if _, err := LoadChimeraYAML(p, nil); err == nil {
				t.Fatal("expected validation error")
			}
		})
	}
}

func TestLocalModels_ServicesListRequiresEnabledModel(t *testing.T) {
	p := writeChimeraYAML(t, "supervisor:\n  services: [local_models]\n")
	res, err := LoadChimeraYAML(p, nil)
	if err == nil {
		_, err = res.ResolveSupervisorServices()
	}
	if err == nil || !strings.Contains(err.Error(), "no local_models.models entry is enabled") {
		t.Fatalf("want local_models error, got %v", err)
	}
}
