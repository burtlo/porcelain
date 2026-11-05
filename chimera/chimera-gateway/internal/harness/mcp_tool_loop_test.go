package harness

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/lynn/porcelain/chimera/chimera-gateway/internal/assistant"
	"github.com/lynn/porcelain/chimera/chimera-gateway/internal/chat"
	"github.com/lynn/porcelain/chimera/chimera-gateway/internal/harness/tools"
	"github.com/lynn/porcelain/chimera/chimera-gateway/internal/mcpclient"
	"github.com/lynn/porcelain/chimera/chimera-gateway/internal/mcpclient/fake"
	"github.com/lynn/porcelain/chimera/chimera-gateway/internal/mcpmgr"
	"github.com/lynn/porcelain/chimera/chimera-gateway/internal/mcptools/catalog"
	"github.com/lynn/porcelain/chimera/chimera-gateway/internal/operatorstore"
	"github.com/lynn/porcelain/chimera/internal/config"
)

// Integration: fake MCP echo via composite executor inside the bounded tool loop.
func TestRunWorkspaceToolLoop_MCPRoundTrip(t *testing.T) {
	serverIn, clientOut := io.Pipe()
	clientIn, serverOut := io.Pipe()
	go func() { _ = fake.Serve(serverIn, serverOut, fake.Config{}) }()
	cli := mcpclient.NewPipeClient(clientIn, clientOut)
	if err := cli.Handshake(context.Background()); err != nil {
		t.Fatal(err)
	}
	mgr := mcpmgr.NewForTest(map[string]*mcpclient.Client{"fake_echo": cli})
	routes := map[string]catalog.Route{
		"fake_echo__echo": {
			ServerID: "fake_echo", MCPToolName: "echo", OpenAIName: "fake_echo__echo",
		},
	}
	comp := &tools.CompositeExecutor{Native: tools.NewWorkspaceExecutor(nil, tools.PolicyRead), Routes: routes, Manager: mgr}

	var upstreamCalls atomic.Int32
	up := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch upstreamCalls.Add(1) {
		case 1:
			_ = json.NewEncoder(w).Encode(map[string]any{
				"choices": []map[string]any{{
					"message": map[string]any{
						"role": "assistant",
						"tool_calls": []map[string]any{{
							"id": "call_mcp_1", "type": "function",
							"function": map[string]string{
								"name":      "fake_echo__echo",
								"arguments": `{"message":"from-model"}`,
							},
						}},
					},
				}},
			})
		default:
			_ = json.NewEncoder(w).Encode(map[string]any{
				"choices": []map[string]any{{
					"message": map[string]string{"role": "assistant", "content": "turn complete"},
				}},
			})
		}
	}))
	defer up.Close()

	rec := httptest.NewRecorder()
	tc := &TurnContext{
		W:            rec,
		InitialModel: "Dev-1.0",
		Timeout:      time.Minute,
		Resolved:     &config.Resolved{UpstreamBaseURL: up.URL},
		Stack: VMStack{
			VM: &assistant.Resolved{
				ModelID: "Dev-1.0",
				HarnessModules: map[string]assistant.HarnessModule{
					operatorstore.HarnessModuleToolExecutor: {
						Enabled:    true,
						ConfigJSON: `{"max_tool_rounds":3,"mcp":{"enabled":true}}`,
					},
				},
			},
		},
		ToolExecutor: comp,
	}
	msgs, _ := json.Marshal([]map[string]string{{"role": "user", "content": "use echo"}})
	body := Body{"model": json.RawMessage(`"Dev-1.0"`), "messages": msgs}
	env := &TurnEnvelope{SchemaVersion: 1}

	runWorkspaceToolLoop(context.Background(), tc, env, body, &chat.ProxyOpts{})

	if rec.Code != http.StatusOK {
		t.Fatalf("status=%d body=%s", rec.Code, rec.Body.String())
	}
	if !strings.Contains(rec.Body.String(), "turn complete") {
		t.Fatalf("final body=%s", rec.Body.String())
	}
	var messages []map[string]json.RawMessage
	if err := json.Unmarshal(body["messages"], &messages); err != nil {
		t.Fatal(err)
	}
	foundToolResult := false
	for _, msg := range messages {
		if string(msg["role"]) != `"tool"` {
			continue
		}
		if strings.Contains(string(msg["content"]), "from-model") {
			foundToolResult = true
		}
	}
	if !foundToolResult {
		t.Fatalf("missing MCP echo in tool messages: %v", messages)
	}
	if upstreamCalls.Load() != 2 {
		t.Fatalf("upstream calls=%d", upstreamCalls.Load())
	}
}
