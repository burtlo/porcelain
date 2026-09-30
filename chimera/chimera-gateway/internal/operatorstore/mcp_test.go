package operatorstore

import (
	"context"
	"path/filepath"
	"testing"

	"github.com/lynn/porcelain/chimera/chimera-gateway/internal/testsupport"
	"github.com/lynn/porcelain/chimera/internal/config"
)

func TestResolveMCPFromStore_SQLiteOverridesYAML(t *testing.T) {
	ctx := context.Background()
	s, err := Open(filepath.Join(t.TempDir(), "op.sqlite"), testsupport.GatewayOperatorMigrationsDir(t), nil)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = s.Close() })
	vm, err := s.CreateAssistant(ctx, CreateAssistantInput{
		ModelID: "Dev-1.0", Name: "Dev", Version: "1.0", TenantID: "", Enabled: true,
	})
	if err != nil {
		t.Fatal(err)
	}
	yaml := config.MCP{
		CatalogMaxTools: 32,
		Servers: []config.MCPServer{
			{ID: "yaml_only", Transport: "stdio", Command: "echo"},
		},
		Assistants: []config.MCPAssistantBinding{
			{AssistantID: "Dev-1.0", Servers: []config.MCPServerBinding{{ServerID: "yaml_only", Tools: map[string]bool{"x": true}}}},
		},
	}
	if got := ResolveMCPFromStore(ctx, s, "", yaml); len(got.Servers) != 1 || got.Servers[0].ID != "yaml_only" {
		t.Fatalf("expected yaml before sqlite rows, got %+v", got.Servers)
	}
	if err := s.UpsertMCPServer(ctx, "", UpsertMCPServerInput{
		ServerID: "fake_echo", Transport: "stdio", Command: "go", Args: []string{"run"},
	}); err != nil {
		t.Fatal(err)
	}
	if err := s.SetMCPAssistantBindings(ctx, "", vm.ID, []MCPAssistantBindingRow{
		{ServerID: "fake_echo", Enabled: true, Tools: map[string]bool{"echo": true}},
	}); err != nil {
		t.Fatal(err)
	}
	got := ResolveMCPFromStore(ctx, s, "", yaml)
	if len(got.Servers) != 1 || got.Servers[0].ID != "fake_echo" {
		t.Fatalf("servers=%+v", got.Servers)
	}
	if len(got.Assistants) != 1 || got.Assistants[0].AssistantID != "Dev-1.0" {
		t.Fatalf("assistants=%+v", got.Assistants)
	}
	if !got.Assistants[0].Servers[0].Tools["echo"] {
		t.Fatalf("tools=%+v", got.Assistants[0].Servers[0].Tools)
	}
	if got.CatalogMaxTools != 32 {
		t.Fatalf("catalog cap should inherit yaml, got %d", got.CatalogMaxTools)
	}
}
