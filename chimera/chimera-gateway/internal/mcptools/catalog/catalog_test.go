package catalog

import (
	"context"
	"io"
	"regexp"
	"testing"

	"github.com/lynn/porcelain/chimera/chimera-gateway/internal/mcpclient"
	"github.com/lynn/porcelain/chimera/chimera-gateway/internal/mcpclient/fake"
	"github.com/lynn/porcelain/chimera/chimera-gateway/internal/mcpmgr"
	"github.com/lynn/porcelain/chimera/internal/config"
)

func startFakeClient(t *testing.T, cfg fake.Config) (*mcpclient.Client, func()) {
	serverIn, clientOut := io.Pipe()
	clientIn, serverOut := io.Pipe()
	done := make(chan struct{})
	go func() {
		_ = fake.Serve(serverIn, serverOut, cfg)
		close(done)
	}()
	cli := mcpclient.NewPipeClient(clientIn, clientOut)
	if err := cli.Handshake(context.Background()); err != nil {
		t.Fatalf("handshake: %v", err)
	}
	return cli, func() {
		_ = clientOut.Close()
		_ = serverOut.Close()
		_ = cli.Close()
		<-done
	}
}

func TestBuildCollisionAndReservedNative(t *testing.T) {
	cli, cleanup := startFakeClient(t, fake.Config{CollisionTool: true})
	defer cleanup()
	mgr := mcpmgr.NewForTest(map[string]*mcpclient.Client{"extra": cli})

	mcpCfg := config.MCP{
		CatalogMaxTools: 32,
		Servers:         []config.MCPServer{{ID: "extra"}},
		Assistants: []config.MCPAssistantBinding{{
			AssistantID: "asst",
			Servers: []config.MCPServerBinding{{
				ServerID: "extra",
				Tools:    map[string]bool{"read_file": true, "echo": true},
			}},
		}},
	}
	bind := mcpCfg.AssistantBinding("asst")
	res, err := Build(context.Background(), mgr, Scope{AssistantID: "asst", MCP: mcpCfg, Binding: bind, Cap: 32})
	if err != nil {
		t.Fatal(err)
	}
	if res.Routes["extra__read_file"].MCPToolName != "read_file" {
		t.Fatalf("routes=%v", res.Routes)
	}
	if _, ok := res.Routes["read_file"]; ok {
		t.Fatal("native read_file slug must not be claimed by MCP route key")
	}
}

func TestBuildTwoServersSameToolName(t *testing.T) {
	cliA, cleanupA := startFakeClient(t, fake.Config{})
	defer cleanupA()
	cliB, cleanupB := startFakeClient(t, fake.Config{})
	defer cleanupB()
	mgr := mcpmgr.NewForTest(map[string]*mcpclient.Client{"srv_a": cliA, "srv_b": cliB})

	mcpCfg := config.MCP{
		CatalogMaxTools: 32,
		Servers: []config.MCPServer{
			{ID: "srv_a"}, {ID: "srv_b"},
		},
		Assistants: []config.MCPAssistantBinding{{
			AssistantID: "asst",
			Servers: []config.MCPServerBinding{
				{ServerID: "srv_a", Tools: map[string]bool{"echo": true}},
				{ServerID: "srv_b", Tools: map[string]bool{"echo": true}},
			},
		}},
	}
	bind := mcpCfg.AssistantBinding("asst")
	res, err := Build(context.Background(), mgr, Scope{MCP: mcpCfg, Binding: bind, Cap: 32})
	if err != nil {
		t.Fatal(err)
	}
	if len(res.Routes) != 2 {
		t.Fatalf("routes=%v", res.Routes)
	}
	names := map[string]bool{}
	for k := range res.Routes {
		if names[k] {
			t.Fatalf("duplicate slug %q", k)
		}
		names[k] = true
	}
}

func TestBuildCatalogTruncatedFailClosed(t *testing.T) {
	cli, cleanup := startFakeClient(t, fake.Config{})
	defer cleanup()
	mgr := mcpmgr.NewForTest(map[string]*mcpclient.Client{"srv": cli})
	mcpCfg := config.MCP{
		Servers: []config.MCPServer{{ID: "srv"}},
		Assistants: []config.MCPAssistantBinding{{
			AssistantID: "asst",
			Servers:     []config.MCPServerBinding{{ServerID: "srv", Tools: map[string]bool{"echo": true}}},
		}},
	}
	bind := mcpCfg.AssistantBinding("asst")
	_, err := Build(context.Background(), mgr, Scope{MCP: mcpCfg, Binding: bind, Cap: 4})
	if err != ErrCatalogTruncated {
		t.Fatalf("err=%v", err)
	}
}

var openAINamePattern = regexp.MustCompile(`^[a-zA-Z0-9_]{1,64}$`)

func TestBuildOpenAINamesMatchContract(t *testing.T) {
	cli, cleanup := startFakeClient(t, fake.Config{CollisionTool: true})
	defer cleanup()
	mgr := mcpmgr.NewForTest(map[string]*mcpclient.Client{"extra": cli})
	mcpCfg := config.MCP{
		Servers: []config.MCPServer{{ID: "extra"}},
		Assistants: []config.MCPAssistantBinding{{
			AssistantID: "asst",
			Servers: []config.MCPServerBinding{{
				ServerID: "extra",
				Tools:    map[string]bool{"read_file": true, "echo": true},
			}},
		}},
	}
	bind := mcpCfg.AssistantBinding("asst")
	res, err := Build(context.Background(), mgr, Scope{MCP: mcpCfg, Binding: bind, Cap: 32})
	if err != nil {
		t.Fatal(err)
	}
	for _, decl := range res.MCPTools {
		fn, _ := decl["function"].(map[string]any)
		name, _ := fn["name"].(string)
		if !openAINamePattern.MatchString(name) {
			t.Fatalf("bad openai name %q", name)
		}
	}
}

func TestBuildDenyWithoutToolAllowlist(t *testing.T) {
	cli, cleanup := startFakeClient(t, fake.Config{})
	defer cleanup()
	mgr := mcpmgr.NewForTest(map[string]*mcpclient.Client{"srv": cli})
	mcpCfg := config.MCP{
		Servers: []config.MCPServer{{ID: "srv"}},
		Assistants: []config.MCPAssistantBinding{{
			AssistantID: "asst",
			Servers:     []config.MCPServerBinding{{ServerID: "srv", Tools: map[string]bool{}}},
		}},
	}
	bind := mcpCfg.AssistantBinding("asst")
	res, err := Build(context.Background(), mgr, Scope{MCP: mcpCfg, Binding: bind, Cap: 32})
	if err != nil {
		t.Fatal(err)
	}
	if len(res.MCPTools) != 0 {
		t.Fatalf("expected no MCP tools without allow entries, got %d", len(res.MCPTools))
	}
}

func TestBuildDisabledToolNotListed(t *testing.T) {
	cli, cleanup := startFakeClient(t, fake.Config{})
	defer cleanup()
	mgr := mcpmgr.NewForTest(map[string]*mcpclient.Client{"srv": cli})
	mcpCfg := config.MCP{
		Servers: []config.MCPServer{{ID: "srv"}},
		Assistants: []config.MCPAssistantBinding{{
			AssistantID: "asst",
			Servers:     []config.MCPServerBinding{{ServerID: "srv", Tools: map[string]bool{"echo": false}}},
		}},
	}
	bind := mcpCfg.AssistantBinding("asst")
	res, err := Build(context.Background(), mgr, Scope{MCP: mcpCfg, Binding: bind, Cap: 32})
	if err != nil {
		t.Fatal(err)
	}
	if len(res.MCPTools) != 0 {
		t.Fatalf("disabled tool should not inject, got %+v", res.MCPTools)
	}
}

func TestBuildSoftFailOneServerList(t *testing.T) {
	okCli, cleanupOK := startFakeClient(t, fake.Config{})
	defer cleanupOK()
	failCli, cleanupFail := startFakeClient(t, fake.Config{FailList: true})
	defer cleanupFail()
	mgr := mcpmgr.NewForTest(map[string]*mcpclient.Client{"good": okCli, "bad": failCli})
	mcpCfg := config.MCP{
		Servers: []config.MCPServer{{ID: "good"}, {ID: "bad"}},
		Assistants: []config.MCPAssistantBinding{{
			AssistantID: "asst",
			Servers: []config.MCPServerBinding{
				{ServerID: "good", Tools: map[string]bool{"echo": true}},
				{ServerID: "bad", Tools: map[string]bool{"echo": true}},
			},
		}},
	}
	bind := mcpCfg.AssistantBinding("asst")
	res, err := Build(context.Background(), mgr, Scope{MCP: mcpCfg, Binding: bind, Cap: 32})
	if err != nil {
		t.Fatal(err)
	}
	if len(res.MCPTools) != 1 {
		t.Fatalf("expected tools from good server only, got %+v", res.MCPTools)
	}
	if res.Routes["good__echo"].ServerID != "good" {
		t.Fatalf("routes=%v", res.Routes)
	}
}
