package supervise

import (
	"context"
	"encoding/json"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"sync"
	"testing"

	svconfig "github.com/lynn/porcelain/chimera/chimera-supervisor/internal/config"
	gwconfig "github.com/lynn/porcelain/chimera/internal/config"
)

func testModel(id, endpoint string) gwconfig.LocalModel {
	return gwconfig.LocalModel{ID: id, Enabled: true, Endpoint: endpoint, Listen: "127.0.0.1:7770", ModelPath: "m.gguf", Alias: id + "-alias",
		CtxSize: 65536, NGPULayers: 99, NCPUMoE: 2, Parallel: 1, StartupSec: 300, CacheDir: "cache", ExtraArgs: []string{"-fa", "on"}}
}

func TestMaterializeBrokerConfig_AddsLocalProvidersKeepsSeed(t *testing.T) {
	dir := t.TempDir()
	seed := filepath.Join(dir, "seed.json")
	if err := os.WriteFile(seed, []byte(`{"client":{"drop_excess_requests":false},"providers":{"groq":{"keys":[]},"local-mine":{"keys":[],"note":"operator"}}}`), 0o600); err != nil {
		t.Fatal(err)
	}
	path, err := materializeBrokerConfig(seed, filepath.Join(dir, "broker"), []gwconfig.LocalModel{testModel("qwen35b", "127.0.0.1:8181"), testModel("mine", "127.0.0.1:8182")})
	if err != nil {
		t.Fatal(err)
	}
	raw, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	var doc struct {
		Client    map[string]any            `json:"client"`
		Providers map[string]map[string]any `json:"providers"`
	}
	if err := json.Unmarshal(raw, &doc); err != nil {
		t.Fatal(err)
	}
	if doc.Client == nil || doc.Providers["groq"] == nil {
		t.Fatalf("seed content lost: %s", raw)
	}
	p := doc.Providers["local-qwen35b"]
	if p == nil {
		t.Fatalf("local provider missing: %s", raw)
	}
	nc := p["network_config"].(map[string]any)
	cp := p["custom_provider_config"].(map[string]any)
	if nc["base_url"] != "http://127.0.0.1:8181" || cp["base_provider_type"] != "openai" || cp["is_key_less"] != true {
		t.Fatalf("bad local provider: %v", p)
	}
	if doc.Providers["local-mine"]["note"] != "operator" {
		t.Fatalf("operator-defined provider was overwritten: %v", doc.Providers["local-mine"])
	}
}

func TestMaterializeBrokerConfig_MissingSeed(t *testing.T) {
	dir := t.TempDir()
	path, err := materializeBrokerConfig(filepath.Join(dir, "none.json"), dir, []gwconfig.LocalModel{testModel("a", "127.0.0.1:1")})
	if err != nil {
		t.Fatal(err)
	}
	raw, _ := os.ReadFile(path)
	if !strings.Contains(string(raw), `"local-a"`) {
		t.Fatalf("missing provider: %s", raw)
	}
}

func TestLocalModelArgs(t *testing.T) {
	m := testModel("qwen35b", "127.0.0.1:8181")
	m.MmprojPath = "mm.gguf"
	args, err := localModelArgs(m, "llama-server.exe")
	if err != nil {
		t.Fatal(err)
	}
	joined := strings.Join(args, " ")
	for _, want := range []string{"-bin llama-server.exe", "-endpoint 127.0.0.1:8181", "-model-id qwen35b", "-alias qwen35b-alias", "-ctx-size 65536", "-n-cpu-moe 2", "-thinking=false", "-mmproj-path mm.gguf", "-startup-timeout 5m0s"} {
		if !strings.Contains(joined, want) {
			t.Fatalf("args missing %q: %v", want, args)
		}
	}
	i := slices.Index(args, "-extra-args")
	if i < 0 || args[i+1] != `["-fa","on"]` {
		t.Fatalf("extra args not JSON-encoded: %v", args)
	}
}

func TestLlamaServerBinForChat_Precedence(t *testing.T) {
	res := &gwconfig.Resolved{LocalModels: gwconfig.LocalModels{LlamaServerBin: "yaml-llama"}}
	if got := llamaServerBinForChat(svconfig.Config{LLMBackendBin: "flag-llama", EmbedBackendBin: "embed-llama"}, res); got != "flag-llama" {
		t.Fatalf("flag should win, got %q", got)
	}
	if got := llamaServerBinForChat(svconfig.Config{EmbedBackendBin: "embed-llama"}, res); got != "yaml-llama" {
		t.Fatalf("yaml should win over embed default, got %q", got)
	}
	if got := llamaServerBinForChat(svconfig.Config{EmbedBackendBin: "embed-llama"}, &gwconfig.Resolved{}); got != "embed-llama" {
		t.Fatalf("embed default fallback, got %q", got)
	}
}

func TestPruneStaleLocalProviders(t *testing.T) {
	var mu sync.Mutex
	var deleted []string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.Method == http.MethodGet && r.URL.Path == "/api/providers":
			_, _ = io.WriteString(w, `{"providers":[{"name":"groq"},{"name":"local-qwen35b"},{"name":"local-old"},{"name":"ollama"}]}`)
		case r.Method == http.MethodDelete && strings.HasPrefix(r.URL.Path, "/api/providers/"):
			mu.Lock()
			deleted = append(deleted, strings.TrimPrefix(r.URL.Path, "/api/providers/"))
			mu.Unlock()
		default:
			http.NotFound(w, r)
		}
	}))
	defer srv.Close()
	log := slog.New(slog.NewTextHandler(io.Discard, nil))
	pruneStaleLocalProviders(context.Background(), strings.TrimPrefix(srv.URL, "http://"), []gwconfig.LocalModel{testModel("qwen35b", "x:1")}, log)
	if len(deleted) != 1 || deleted[0] != "local-old" {
		t.Fatalf("deleted = %v, want only local-old", deleted)
	}
}
