package harness

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"testing"
	"time"

	"github.com/lynn/porcelain/chimera/chimera-gateway/internal/harness/tools"
	"github.com/lynn/porcelain/chimera/chimera-gateway/internal/mcpmgr"
	"github.com/lynn/porcelain/chimera/chimera-gateway/internal/mcptools/catalog"
	"github.com/lynn/porcelain/chimera/internal/config"
)

// Integration: chimera.yaml MCP section → manager subprocess → catalog → composite tools/call.
func TestBlessedMCPSidecarFromYAML_CompositeCall(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("stdio subprocess test skipped on windows")
	}
	fakeMain := blessedFakeMainPath(t)
	dir := t.TempDir()
	yamlPath := filepath.Join(dir, "chimera.yaml")
	raw := fmt.Sprintf(`gateway:
  listen_port: 3000
mcp:
  servers:
    - id: fake_echo
      transport: stdio
      command: go
      args:
        - run
        - %s
      env_allowlist: [PATH, HOME, GOROOT, GOPATH, GOMOD, GOWORK, CGO_ENABLED]
  assistants:
    - assistant_id: Dev-1.0
      servers:
        - server_id: fake_echo
          tools:
            echo: true
`, fakeMain)
	if err := os.WriteFile(yamlPath, []byte(raw), 0o644); err != nil {
		t.Fatal(err)
	}
	resolved, err := config.LoadChimeraYAML(yamlPath, nil)
	if err != nil {
		t.Fatal(err)
	}
	mgr := mcpmgr.New(resolved.MCP, nil)
	defer mgr.Stop()

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	bind := resolved.MCP.AssistantBinding("Dev-1.0")
	if bind == nil {
		t.Fatal("missing assistant binding")
	}
	built, err := catalog.Build(ctx, mgr, catalog.Scope{
		AssistantID: "Dev-1.0",
		MCP:         resolved.MCP,
		Binding:     bind,
		Cap:         resolved.MCP.CatalogMaxTools,
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(built.Routes) == 0 {
		t.Fatal("expected MCP routes from YAML-defined server")
	}
	route := built.Routes["fake_echo__echo"]
	if route.ServerID != "fake_echo" || route.MCPToolName != "echo" {
		t.Fatalf("route=%+v", route)
	}
	native := tools.NewWorkspaceExecutor(nil, tools.PolicyNone)
	comp := &tools.CompositeExecutor{
		Native:  native,
		Routes:  built.Routes,
		Manager: mgr,
	}
	args, _ := json.Marshal(map[string]string{"message": "yaml-blessed-echo"})
	got, err := comp.Invoke(ctx, tools.ToolCall{Name: "fake_echo__echo", Arguments: args})
	if err != nil || got.IsError || got.Content != "yaml-blessed-echo" {
		t.Fatalf("invoke: %+v err=%v", got, err)
	}
}

func blessedFakeMainPath(t *testing.T) string {
	t.Helper()
	_, filename, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("runtime.Caller failed")
	}
	root := filepath.Clean(filepath.Join(filepath.Dir(filename), "..", "..", "..", ".."))
	return filepath.Join(root, "chimera", "chimera-gateway", "internal", "mcpclient", "fake", "cmd", "main.go")
}
