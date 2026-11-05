package config

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/lynn/porcelain/chimera/chimera-indexer/adapter"
	"gopkg.in/yaml.v3"
)

// MaterializeIndexerConfig merges inline IndexerFileConfig with optional overlay and
// writes the effective FileConfig YAML to IndexerMaterializedPath.
func MaterializeIndexerConfig(res *Resolved) (string, error) {
	if res == nil {
		return "", fmt.Errorf("nil resolved config")
	}
	base := res.IndexerFileConfig
	if overlay := strings.TrimSpace(res.IndexerOverlayPath); overlay != "" {
		if st, err := os.Stat(overlay); err == nil && !st.IsDir() {
			raw, err := os.ReadFile(overlay)
			if err != nil {
				return "", fmt.Errorf("indexer overlay %q: %w", overlay, err)
			}
			var fc adapter.FileConfig
			if err := yaml.Unmarshal(raw, &fc); err != nil {
				return "", fmt.Errorf("indexer overlay %q: %w", overlay, err)
			}
			base = adapter.MergeFileConfig(base, fc)
		}
	}
	// Supervised roots come from the workspaces API; clear YAML roots in materialized file.
	base.Roots = nil
	raw, err := yaml.Marshal(&base)
	if err != nil {
		return "", fmt.Errorf("marshal indexer config: %w", err)
	}
	path := strings.TrimSpace(res.IndexerMaterializedPath)
	if path == "" {
		return "", fmt.Errorf("indexer materialized path empty")
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return "", fmt.Errorf("indexer materialize dir: %w", err)
	}
	tmp := path + ".tmp"
	if err := os.WriteFile(tmp, raw, 0o644); err != nil {
		return "", fmt.Errorf("indexer materialize write: %w", err)
	}
	if err := os.Rename(tmp, path); err != nil {
		_ = os.Remove(tmp)
		return "", fmt.Errorf("indexer materialize rename: %w", err)
	}
	res.IndexerSupervisedConfigPath = path
	return path, nil
}

// EffectiveIndexerFileConfig returns inline config merged with overlay (no write).
func EffectiveIndexerFileConfig(res *Resolved) (adapter.FileConfig, error) {
	if res == nil {
		return adapter.FileConfig{}, fmt.Errorf("nil resolved config")
	}
	base := res.IndexerFileConfig
	if overlay := strings.TrimSpace(res.IndexerOverlayPath); overlay != "" {
		if st, err := os.Stat(overlay); err == nil && !st.IsDir() {
			raw, err := os.ReadFile(overlay)
			if err != nil {
				return adapter.FileConfig{}, err
			}
			var fc adapter.FileConfig
			if err := yaml.Unmarshal(raw, &fc); err != nil {
				return adapter.FileConfig{}, err
			}
			base = adapter.MergeFileConfig(base, fc)
		}
	}
	base.Roots = nil
	return base, nil
}
