package tools

import (
	"testing"

	"github.com/lynn/porcelain/chimera/chimera-gateway/internal/mcptools/catalog"
)

func TestDispatchFor_native(t *testing.T) {
	ws := NewWorkspaceExecutor(nil, PolicyRead)
	meta := DispatchFor(ws, ToolCall{Name: "read_file"})
	if meta.Backend != "native" || meta.Service != "chimera-gateway" {
		t.Fatalf("meta=%+v", meta)
	}
}

func TestDispatchFor_mcp(t *testing.T) {
	comp := &CompositeExecutor{
		Native: NewWorkspaceExecutor(nil, PolicyRead),
		Routes: map[string]catalog.Route{
			"fake_echo__echo": {ServerID: "fake_echo", OpenAIName: "fake_echo__echo"},
		},
	}
	meta := DispatchFor(comp, ToolCall{Name: "fake_echo__echo"})
	if meta.Backend != "mcp" || meta.MCPServer != "fake_echo" || meta.Service != "fake_echo" {
		t.Fatalf("meta=%+v", meta)
	}
}

func TestDispatchFor_expansion(t *testing.T) {
	meta := DispatchFor(NewWorkspaceExecutor(nil, PolicyRead), ToolCall{Name: "workspace_read_lines"})
	if meta.Backend != "expansion" {
		t.Fatalf("meta=%+v", meta)
	}
}
