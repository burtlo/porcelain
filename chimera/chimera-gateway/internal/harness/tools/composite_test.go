package tools

import (
	"context"
	"encoding/json"
	"io"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/lynn/porcelain/chimera/chimera-gateway/internal/mcpclient"
	"github.com/lynn/porcelain/chimera/chimera-gateway/internal/mcpclient/fake"
	"github.com/lynn/porcelain/chimera/chimera-gateway/internal/mcpmgr"
	"github.com/lynn/porcelain/chimera/chimera-gateway/internal/mcptools/catalog"
)

func TestCompositeNativeWinsReadFile(t *testing.T) {
	root := t.TempDir()
	native := NewWorkspaceExecutor([]string{root}, PolicyReadWrite)
	serverIn, clientOut := io.Pipe()
	clientIn, serverOut := io.Pipe()
	go func() { _ = fake.Serve(serverIn, serverOut, fake.Config{CollisionTool: true}) }()
	cli := mcpclient.NewPipeClient(clientIn, clientOut)
	_ = cli.Handshake(context.Background())
	mgr := mcpmgr.NewForTest(map[string]*mcpclient.Client{"extra": cli})
	routes := map[string]catalog.Route{
		"extra__read_file": {ServerID: "extra", MCPToolName: "read_file", OpenAIName: "extra__read_file"},
	}
	comp := &CompositeExecutor{Native: native, Routes: routes, Manager: mgr}
	if err := os.WriteFile(filepath.Join(root, "notes.txt"), []byte("native"), 0o644); err != nil {
		t.Fatal(err)
	}
	args, _ := json.Marshal(map[string]string{"path": "notes.txt"})
	got, err := comp.Invoke(context.Background(), ToolCall{Name: "read_file", Arguments: args})
	if err != nil || got.IsError || got.Content != "native" {
		t.Fatalf("native read_file: %+v err=%v", got, err)
	}
}

func TestCompositeMCPDispatch(t *testing.T) {
	native := NewWorkspaceExecutor(nil, PolicyNone)
	serverIn, clientOut := io.Pipe()
	clientIn, serverOut := io.Pipe()
	go func() { _ = fake.Serve(serverIn, serverOut, fake.Config{}) }()
	cli := mcpclient.NewPipeClient(clientIn, clientOut)
	_ = cli.Handshake(context.Background())
	mgr := mcpmgr.NewForTest(map[string]*mcpclient.Client{"fake": cli})
	routes := map[string]catalog.Route{
		"fake__echo": {ServerID: "fake", MCPToolName: "echo", OpenAIName: "fake__echo"},
	}
	comp := &CompositeExecutor{Native: native, Routes: routes, Manager: mgr}
	args, _ := json.Marshal(map[string]string{"message": "hi"})
	got, err := comp.Invoke(context.Background(), ToolCall{Name: "fake__echo", Arguments: args})
	if err != nil || got.IsError || got.Content != "hi" {
		t.Fatalf("mcp echo: %+v err=%v", got, err)
	}
}

func TestCompositeCallTimeoutReturnsToolError(t *testing.T) {
	native := NewWorkspaceExecutor(nil, PolicyNone)
	serverIn, clientOut := io.Pipe()
	clientIn, serverOut := io.Pipe()
	go func() {
		_ = fake.Serve(serverIn, serverOut, fake.Config{EchoDelay: 500 * time.Millisecond})
	}()
	cli := mcpclient.NewPipeClient(clientIn, clientOut)
	_ = cli.Handshake(context.Background())
	mgr := mcpmgr.NewForTest(map[string]*mcpclient.Client{"fake": cli})
	routes := map[string]catalog.Route{
		"fake__echo": {ServerID: "fake", MCPToolName: "echo", OpenAIName: "fake__echo"},
	}
	comp := &CompositeExecutor{
		Native:      native,
		Routes:      routes,
		Manager:     mgr,
		CallTimeout: 50 * time.Millisecond,
	}
	args, _ := json.Marshal(map[string]string{"message": "slow"})
	got, err := comp.Invoke(context.Background(), ToolCall{Name: "fake__echo", Arguments: args})
	if err == nil {
		t.Fatal("expected timeout error")
	}
	if !got.IsError {
		t.Fatalf("expected tool error envelope, got %+v", got)
	}
}

func TestCompositeMCPResultIsError(t *testing.T) {
	native := NewWorkspaceExecutor(nil, PolicyNone)
	serverIn, clientOut := io.Pipe()
	clientIn, serverOut := io.Pipe()
	go func() { _ = fake.Serve(serverIn, serverOut, fake.Config{}) }()
	cli := mcpclient.NewPipeClient(clientIn, clientOut)
	_ = cli.Handshake(context.Background())
	mgr := mcpmgr.NewForTest(map[string]*mcpclient.Client{"fake": cli})
	routes := map[string]catalog.Route{
		"fake__echo": {ServerID: "fake", MCPToolName: "echo", OpenAIName: "fake__echo"},
	}
	comp := &CompositeExecutor{Native: native, Routes: routes, Manager: mgr}
	args, _ := json.Marshal(map[string]string{"message": "__force_error__"})
	got, err := comp.Invoke(context.Background(), ToolCall{Name: "fake__echo", Arguments: args})
	if err != nil {
		t.Fatal(err)
	}
	if !got.IsError {
		t.Fatalf("expected MCP isError in tool result, got %+v", got)
	}
}
