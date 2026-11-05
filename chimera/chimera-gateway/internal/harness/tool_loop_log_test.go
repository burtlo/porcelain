package harness

import (
	"bytes"
	"context"
	"encoding/json"
	"log/slog"
	"strings"
	"testing"

	"github.com/lynn/porcelain/chimera/chimera-gateway/internal/harness/tools"
	"github.com/lynn/porcelain/internal/naming"
)

type stubExecutor struct {
	content string
}

func (s stubExecutor) Invoke(context.Context, tools.ToolCall) (tools.ToolResult, error) {
	return tools.ToolResult{Content: s.content}, nil
}

func TestInvokeAndLogTool_emitsHarnessSlugs(t *testing.T) {
	var buf bytes.Buffer
	log := slog.New(slog.NewJSONHandler(&buf, &slog.HandlerOptions{Level: slog.LevelInfo}))
	tc := &TurnContext{
		RouteLog:       log,
		RequestID:      "req-1",
		ConversationID: "conv-1",
		TurnIndex:      2,
	}
	env := &TurnEnvelope{AssistantID: "asst-1"}
	args, _ := json.Marshal(map[string]string{"path": "docs/x.md"})
	call := tools.ToolCall{ID: "c1", Name: "read_file", Arguments: args}

	_ = invokeAndLogTool(context.Background(), tc, env, 1, call, stubExecutor{content: "hello"})

	out := buf.String()
	for _, slug := range []string{
		naming.MsgHarnessToolInvokeStarted,
		naming.MsgHarnessToolInvokeCompleted,
	} {
		if !strings.Contains(out, slug) {
			t.Fatalf("missing %s in log:\n%s", slug, out)
		}
	}
	if !strings.Contains(out, `"tool_backend":"native"`) || !strings.Contains(out, `"service":"chimera-gateway"`) {
		t.Fatalf("missing backend/service in log:\n%s", out)
	}
	if !strings.Contains(out, `"path":"docs/x.md"`) {
		t.Fatalf("missing path in log:\n%s", out)
	}
}
