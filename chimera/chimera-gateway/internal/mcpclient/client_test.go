package mcpclient

import (
	"context"
	"encoding/json"
	"io"
	"regexp"
	"strings"
	"testing"
	"time"

	"github.com/lynn/porcelain/chimera/chimera-gateway/internal/mcpclient/fake"
)

func startFake(cfg fake.Config) (*Client, func()) {
	serverIn, clientOut := io.Pipe()
	clientIn, serverOut := io.Pipe()
	done := make(chan struct{})
	go func() {
		_ = fake.Serve(serverIn, serverOut, cfg)
		close(done)
	}()
	cl := NewPipeClient(clientIn, clientOut)
	return cl, func() {
		_ = clientOut.Close()
		_ = serverOut.Close()
		_ = cl.Close()
		<-done
	}
}

func TestClientHandshakeListAndCall(t *testing.T) {
	cl, cleanup := startFake(fake.Config{})
	defer cleanup()

	ctx := context.Background()
	if err := cl.Handshake(ctx); err != nil {
		t.Fatalf("handshake: %v", err)
	}
	tools, err := cl.ListTools(ctx)
	if err != nil {
		t.Fatalf("list: %v", err)
	}
	if len(tools) != 1 || tools[0].Name != "echo" {
		t.Fatalf("tools: %+v", tools)
	}
	args, _ := json.Marshal(map[string]string{"message": "hello mcp"})
	raw, err := cl.CallTool(ctx, "echo", args)
	if err != nil {
		t.Fatalf("call: %v", err)
	}
	if !strings.Contains(string(raw), "hello mcp") {
		t.Fatalf("result: %s", raw)
	}
}

func TestClientCallTimeout(t *testing.T) {
	serverIn, clientOut := io.Pipe()
	clientIn, _ := io.Pipe()
	cl := NewPipeClient(clientIn, clientOut)
	defer cl.Close()

	go func() {
		_ = fake.Serve(serverIn, io.Discard, fake.Config{})
	}()

	ctx, cancel := context.WithTimeout(context.Background(), 50*time.Millisecond)
	defer cancel()
	_, err := cl.CallTool(ctx, "echo", nil)
	if err == nil {
		t.Fatal("expected timeout")
	}
}

var openAINamePattern = regexp.MustCompile(`^[a-zA-Z0-9_]{1,64}$`)

func TestOpenAINameLengthAndCollision(t *testing.T) {
	long := OpenAIName("server-with-a-very-long-identifier-name", "also_a_really_long_tool_name_for_truncation")
	if len(long) > 64 {
		t.Fatalf("len=%d name=%q", len(long), long)
	}
	if !openAINamePattern.MatchString(long) {
		t.Fatalf("pattern: %q", long)
	}
	dup := OpenAIName("git_local", "status")
	if dup != "git_local__status" {
		t.Fatalf("got %q", dup)
	}
	mcpRead := OpenAIName("extra", "read_file")
	if mcpRead != "extra__read_file" {
		t.Fatalf("got %q", mcpRead)
	}
	// Contract table row: long segments force digest suffix (≤ 64 chars).
	digestName := OpenAIName(strings.Repeat("a", 50), strings.Repeat("b", 20))
	if len(digestName) > 64 {
		t.Fatalf("len=%d name=%q", len(digestName), digestName)
	}
	if !strings.Contains(digestName, "__") {
		t.Fatalf("expected digest suffix: %q", digestName)
	}
}
