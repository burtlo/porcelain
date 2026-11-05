package config

import (
	"bytes"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"

	"gopkg.in/yaml.v3"
)

// PatchChimeraYAMLBytesWithListenHost sets gateway.listen_host in chimera.yaml bytes.
func PatchChimeraYAMLBytesWithListenHost(raw []byte, host string) ([]byte, error) {
	return patchChimeraYAMLApplyGateway(func(gwNode *yaml.Node) {
		setOrReplaceMappingString(gwNode, "listen_host", host)
	}, raw)
}

// WriteChimeraListenHost updates gateway.listen_host in chimera.yaml.
func WriteChimeraListenHost(chimeraPath, host string) error {
	raw, err := os.ReadFile(chimeraPath)
	if err != nil {
		return fmt.Errorf("read chimera yaml: %w", err)
	}
	out, err := PatchChimeraYAMLBytesWithListenHost(raw, host)
	if err != nil {
		return err
	}
	dir := filepath.Dir(chimeraPath)
	tmp, err := os.CreateTemp(dir, "chimera-listen-*.yaml")
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
		return fmt.Errorf("chimera yaml after patch failed to load: %w", err)
	}
	mode := fs.FileMode(0o644)
	if st, err := os.Stat(chimeraPath); err == nil {
		mode = st.Mode() & fs.ModePerm
	}
	return ReplaceFile(chimeraPath, out, mode)
}

func patchChimeraYAMLApplyGateway(fn func(*yaml.Node), raw []byte) ([]byte, error) {
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
	gwNode := mappingGetOrCreateChildMapping(docMap, "gateway")
	fn(gwNode)
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

func setOrReplaceMappingString(m *yaml.Node, key, v string) {
	if m.Kind != yaml.MappingNode {
		return
	}
	scalar := &yaml.Node{Kind: yaml.ScalarNode, Tag: "!!str", Value: v, Style: yaml.DoubleQuotedStyle}
	idx := mappingIndex(m, key)
	if idx >= 0 {
		m.Content[idx+1] = scalar
		return
	}
	kn := &yaml.Node{Kind: yaml.ScalarNode, Tag: "!!str", Value: key}
	m.Content = append(m.Content, kn, scalar)
}
