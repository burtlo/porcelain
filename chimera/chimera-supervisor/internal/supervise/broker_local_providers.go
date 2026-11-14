package supervise

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"time"

	gwconfig "github.com/lynn/porcelain/chimera/internal/config"
	"github.com/lynn/porcelain/internal/naming"
)

// materializedBrokerConfigName is written into the broker data dir and handed to chimera-broker
// (BROKER__CHIMERA_BROKER_CONFIG). BiFrost merges providers from this file into its config store
// on startup, so local models are registered before the first request.
const materializedBrokerConfigName = "chimera-broker.materialized.json"

// localModelProviderConfig is the BiFrost provider entry for one local llama-server:
// a keyless OpenAI-compatible custom provider.
func localModelProviderConfig(m gwconfig.LocalModel) map[string]any {
	return map[string]any{
		"keys": []any{},
		"network_config": map[string]any{
			"base_url":                           "http://" + m.Endpoint,
			"default_request_timeout_in_seconds": 600,
		},
		"custom_provider_config": map[string]any{
			"base_provider_type": "openai",
			"is_key_less":        true,
			"allowed_requests": map[string]any{
				"chat_completion":        true,
				"chat_completion_stream": true,
				"text_completion":        true,
				"list_models":            true,
			},
		},
	}
}

// brokerSeedConfigPath is the operator broker config the materialized copy starts from.
func brokerSeedConfigPath() string {
	if p := strings.TrimSpace(os.Getenv(naming.EnvBrokerChimeraBrokerConfig)); p != "" {
		return p
	}
	return naming.DefaultBrokerConfigPath
}

// materializeBrokerConfig writes seed + one "local-<id>" provider per enabled model to
// <dataDir>/chimera-broker.materialized.json and returns that path. Operator-defined
// providers with the same name win (they are never overwritten).
func materializeBrokerConfig(seedPath, dataDir string, models []gwconfig.LocalModel) (string, error) {
	doc := map[string]any{}
	if raw, err := os.ReadFile(seedPath); err == nil {
		if err := json.Unmarshal(raw, &doc); err != nil {
			return "", fmt.Errorf("parse broker seed config %s: %w", seedPath, err)
		}
	} else if !os.IsNotExist(err) {
		return "", fmt.Errorf("read broker seed config %s: %w", seedPath, err)
	}
	providers, _ := doc["providers"].(map[string]any)
	if providers == nil {
		providers = map[string]any{}
	}
	for _, m := range models {
		if _, exists := providers[m.ProviderName()]; exists {
			continue
		}
		providers[m.ProviderName()] = localModelProviderConfig(m)
	}
	doc["providers"] = providers
	out, err := json.MarshalIndent(doc, "", "  ")
	if err != nil {
		return "", err
	}
	if err := os.MkdirAll(dataDir, 0o755); err != nil {
		return "", err
	}
	path, err := filepath.Abs(filepath.Join(dataDir, materializedBrokerConfigName))
	if err != nil {
		return "", err
	}
	if err := os.WriteFile(path, append(out, '\n'), 0o600); err != nil {
		return "", err
	}
	return path, nil
}

// pruneStaleLocalProviders deletes "local-*" providers from the running broker that are not
// enabled local models. A configured-but-unreachable provider makes BiFrost's /v1/models fail
// for every provider, so disabled or removed local models must not linger in its config store.
func pruneStaleLocalProviders(ctx context.Context, brokerEndpoint string, enabled []gwconfig.LocalModel, log *slog.Logger) {
	keep := map[string]bool{}
	for _, m := range enabled {
		keep[m.ProviderName()] = true
	}
	base := "http://" + strings.TrimSpace(brokerEndpoint)
	client := &http.Client{Timeout: 5 * time.Second}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, base+"/api/providers", nil)
	if err != nil {
		return
	}
	res, err := client.Do(req)
	if err != nil {
		log.Warn("list broker providers failed", "msg", "chimera-supervisor.local_model.prune_list_failed", "err", err)
		return
	}
	defer res.Body.Close()
	var body struct {
		Providers []struct {
			Name string `json:"name"`
		} `json:"providers"`
	}
	if err := json.NewDecoder(res.Body).Decode(&body); err != nil {
		return
	}
	for _, p := range body.Providers {
		if !strings.HasPrefix(p.Name, naming.LocalModelProviderPrefix) || keep[p.Name] {
			continue
		}
		del, err := http.NewRequestWithContext(ctx, http.MethodDelete, base+"/api/providers/"+url.PathEscape(p.Name), nil)
		if err != nil {
			continue
		}
		dres, err := client.Do(del)
		if err != nil {
			log.Warn("remove stale local provider failed", "msg", "chimera-supervisor.local_model.prune_failed", "provider", p.Name, "err", err)
			continue
		}
		dres.Body.Close()
		log.Info("removed stale local model provider", "msg", "chimera-supervisor.local_model.pruned", "provider", p.Name, "status", dres.StatusCode)
	}
}
