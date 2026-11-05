package config

import (
	"bytes"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"

	"github.com/lynn/porcelain/chimera/chimera-indexer/adapter"
	"gopkg.in/yaml.v3"
)

// Suite-only keys under indexer: (not part of standalone FileConfig).
var indexerSuiteKeys = map[string]bool{
	"enabled":     true,
	"config_path": true,
	"bin":         true,
	"log_json":    true,
}

// PatchChimeraYAMLBytesWithIndexerTuning replaces FileConfig keys under indexer:
// while preserving suite keys (enabled, config_path, bin, log_json). Clears roots
// (supervised watch paths come from the operator store).
func PatchChimeraYAMLBytesWithIndexerTuning(raw []byte, fc adapter.FileConfig) ([]byte, error) {
	fc.Roots = nil
	var root yaml.Node
	if err := yaml.Unmarshal(raw, &root); err != nil {
		return nil, fmt.Errorf("parse chimera yaml: %w", err)
	}
	if root.Kind != yaml.DocumentNode || len(root.Content) == 0 {
		return nil, fmt.Errorf("chimera yaml: expected document root")
	}
	docMap := root.Content[0]
	if docMap.Kind != yaml.MappingNode {
		return nil, fmt.Errorf("chimera yaml: expected mapping at document root")
	}
	idxNode := mappingGetOrCreateChildMapping(docMap, "indexer")

	fcRaw, err := yaml.Marshal(&fc)
	if err != nil {
		return nil, fmt.Errorf("marshal indexer tuning: %w", err)
	}
	var fcDoc yaml.Node
	if err := yaml.Unmarshal(fcRaw, &fcDoc); err != nil {
		return nil, fmt.Errorf("parse indexer tuning: %w", err)
	}
	if fcDoc.Kind != yaml.DocumentNode || len(fcDoc.Content) == 0 || fcDoc.Content[0].Kind != yaml.MappingNode {
		return nil, fmt.Errorf("indexer tuning: expected mapping")
	}
	fcMap := fcDoc.Content[0]

	// Drop previous FileConfig keys; keep suite keys.
	kept := make([]*yaml.Node, 0, len(idxNode.Content))
	for i := 0; i+1 < len(idxNode.Content); i += 2 {
		k := idxNode.Content[i]
		if indexerSuiteKeys[k.Value] {
			kept = append(kept, k, idxNode.Content[i+1])
		}
	}
	idxNode.Content = kept

	for i := 0; i+1 < len(fcMap.Content); i += 2 {
		k := fcMap.Content[i]
		if k.Value == "roots" || indexerSuiteKeys[k.Value] {
			continue
		}
		idxNode.Content = append(idxNode.Content, k, fcMap.Content[i+1])
	}

	var buf bytes.Buffer
	enc := yaml.NewEncoder(&buf)
	enc.SetIndent(2)
	if err := enc.Encode(&root); err != nil {
		_ = enc.Close()
		return nil, fmt.Errorf("encode chimera yaml: %w", err)
	}
	if err := enc.Close(); err != nil {
		return nil, fmt.Errorf("encode chimera yaml: %w", err)
	}
	return buf.Bytes(), nil
}

// WriteChimeraIndexerTuning updates FileConfig keys under indexer: in chimera.yaml
// when no separate overlay file is used.
func WriteChimeraIndexerTuning(chimeraPath string, fc adapter.FileConfig) error {
	raw, err := os.ReadFile(chimeraPath)
	if err != nil {
		return fmt.Errorf("read chimera yaml: %w", err)
	}
	out, err := PatchChimeraYAMLBytesWithIndexerTuning(raw, fc)
	if err != nil {
		return err
	}
	dir := filepath.Dir(chimeraPath)
	tmp, err := os.CreateTemp(dir, "chimera-indexer-*.yaml")
	if err != nil {
		return fmt.Errorf("temp file: %w", err)
	}
	tmpPath := tmp.Name()
	_ = tmp.Close()
	defer func() { _ = os.Remove(tmpPath) }()
	if err := os.WriteFile(tmpPath, out, 0o600); err != nil {
		return err
	}
	if _, err := LoadChimeraYAML(tmpPath, nil); err != nil {
		return fmt.Errorf("chimera yaml after indexer patch failed to load: %w", err)
	}
	mode := fs.FileMode(0o644)
	if st, err := os.Stat(chimeraPath); err == nil {
		mode = st.Mode() & fs.ModePerm
	}
	return ReplaceFile(chimeraPath, out, mode)
}
