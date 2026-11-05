package config

import (
	"os"
	"path/filepath"
	"testing"
)

func TestLoadChimeraYAML_mcpSection(t *testing.T) {
	dir := t.TempDir()
	p := filepath.Join(dir, "chimera.yaml")
	raw := `
gateway:
  listen_port: 3000
mcp:
  catalog_max_tools: 16
  call_timeout_ms: 45000
  servers:
    - id: fake_echo
      transport: stdio
      command: echo
      args: ["mcp"]
      auto_start: true
  assistants:
    - assistant_id: Dev-1.0
      servers:
        - server_id: fake_echo
          tools:
            echo: true
`
	if err := os.WriteFile(p, []byte(raw), 0o644); err != nil {
		t.Fatal(err)
	}
	res, err := LoadChimeraYAML(p, nil)
	if err != nil {
		t.Fatal(err)
	}
	if !res.MCP.MCPConfigured() || res.MCP.CatalogMaxTools != 16 {
		t.Fatalf("mcp: %+v", res.MCP)
	}
	if res.MCP.CallTimeout.Milliseconds() != 45000 {
		t.Fatalf("timeout %v", res.MCP.CallTimeout)
	}
	bind := res.MCP.AssistantBinding("Dev-1.0")
	if bind == nil || len(bind.Servers) != 1 || !bind.Servers[0].Tools["echo"] {
		t.Fatalf("binding %+v", bind)
	}
}
