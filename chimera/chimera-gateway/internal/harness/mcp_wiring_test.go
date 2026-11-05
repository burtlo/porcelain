package harness_test

import (
	"context"
	"io"
	"path/filepath"
	"testing"

	"github.com/lynn/porcelain/chimera/chimera-gateway/internal/assistant"
	"github.com/lynn/porcelain/chimera/chimera-gateway/internal/harness"
	"github.com/lynn/porcelain/chimera/chimera-gateway/internal/mcpclient"
	"github.com/lynn/porcelain/chimera/chimera-gateway/internal/mcpclient/fake"
	"github.com/lynn/porcelain/chimera/chimera-gateway/internal/mcpmgr"
	"github.com/lynn/porcelain/chimera/chimera-gateway/internal/operatorstore"
	"github.com/lynn/porcelain/chimera/chimera-gateway/internal/testsupport"
	"github.com/lynn/porcelain/chimera/internal/config"
)

func TestMetaPolicyWiresMCPToolDeclarations(t *testing.T) {
	store, err := operatorstore.Open(filepath.Join(t.TempDir(), "operator.sqlite"), testsupport.GatewayOperatorMigrationsDir(t), nil)
	if err != nil {
		t.Fatal(err)
	}
	defer store.Close()
	workspace, err := store.CreateWorkspace(context.Background(), "tenant-a", "project-a", "flavor-a", []string{t.TempDir()}, operatorstore.WorkspacePolicy{
		FileActionPolicy: "read",
	})
	if err != nil {
		t.Fatal(err)
	}

	serverIn, clientOut := io.Pipe()
	clientIn, serverOut := io.Pipe()
	go func() { _ = fake.Serve(serverIn, serverOut, fake.Config{}) }()
	cli := mcpclient.NewPipeClient(clientIn, clientOut)
	if err := cli.Handshake(context.Background()); err != nil {
		t.Fatal(err)
	}
	mgr := mcpmgr.NewForTest(map[string]*mcpclient.Client{"fake_echo": cli})

	tc := &harness.TurnContext{
		OperatorStore: store,
		TenantID:      "tenant-a",
		ProjectID:     "project-a",
		FlavorID:      "flavor-a",
		MCPManager:    mgr,
		Resolved: &config.Resolved{
			MCP: config.MCP{
				CatalogMaxTools: 32,
				Servers:         []config.MCPServer{{ID: "fake_echo"}},
				Assistants: []config.MCPAssistantBinding{{
					AssistantID: "Policy-1.0",
					Servers: []config.MCPServerBinding{{
						ServerID: "fake_echo",
						Tools:    map[string]bool{"echo": true},
					}},
				}},
			},
		},
		Stack: harness.VMStack{
			VM: &assistant.Resolved{
				ModelID: "Policy-1.0",
				HarnessModules: map[string]assistant.HarnessModule{
					operatorstore.HarnessModuleToolExecutor: {Enabled: true, ConfigJSON: `{"mcp":{"enabled":true}}`},
				},
			},
		},
	}
	env := &harness.TurnEnvelope{}
	if err := (harness.MetaPolicyStage{}).Run(context.Background(), tc, env, harness.Body{}); err != nil {
		t.Fatal(err)
	}
	if tc.WorkspaceScope == nil || tc.WorkspaceScope.ID != workspace.ID {
		t.Fatalf("workspace=%+v", tc.WorkspaceScope)
	}
	if len(tc.ToolDeclarations) < 5 {
		t.Fatalf("expected native+mcp decls, got %d", len(tc.ToolDeclarations))
	}
	foundMCP := false
	for _, decl := range tc.ToolDeclarations {
		fn, _ := decl["function"].(map[string]any)
		if fn != nil && fn["name"] == "fake_echo__echo" {
			foundMCP = true
		}
	}
	if !foundMCP {
		t.Fatalf("declarations=%v", tc.ToolDeclarations)
	}
}
