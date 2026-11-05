package harness

import (
	"context"
	"encoding/json"
	"strings"
	"time"

	"github.com/lynn/porcelain/chimera/chimera-gateway/internal/harness/tools"
	"github.com/lynn/porcelain/internal/naming"
)

func logToolExecutorUnavailable(tc *TurnContext, env *TurnEnvelope) {
	if tc == nil || tc.RouteLog == nil {
		return
	}
	args := []any{
		"msg", naming.MsgHarnessToolExecutorUnavailable,
		"project", tc.ProjectID,
		"flavor", tc.FlavorID,
		"workspace_row_id", tc.WorkspaceRowID,
		"timeline_kind", naming.TimelineKindBroker,
	}
	args = appendToolLogIdentity(args, tc, env)
	tc.RouteLog.Warn("harness tool executor unavailable", args...)
}

func logToolLoopStarted(tc *TurnContext, env *TurnEnvelope, maxRounds int) {
	if tc == nil || tc.RouteLog == nil {
		return
	}
	args := []any{
		"msg", naming.MsgHarnessToolLoopStarted,
		"max_tool_rounds", maxRounds,
		"timeline_kind", naming.TimelineKindBroker,
	}
	args = appendToolLogIdentity(args, tc, env)
	tc.RouteLog.Info("harness tool loop started", args...)
}

func logToolModelRound(tc *TurnContext, env *TurnEnvelope, round int, parseSource string, callCount int) {
	if tc == nil || tc.RouteLog == nil {
		return
	}
	args := []any{
		"msg", naming.MsgHarnessToolModelRound,
		"round", round,
		"parse_source", parseSource,
		"call_count", callCount,
		"timeline_kind", naming.TimelineKindBroker,
	}
	args = appendToolLogIdentity(args, tc, env)
	tc.RouteLog.Info("harness tool model round", args...)
}

func logToolLoopFinished(tc *TurnContext, env *TurnEnvelope, roundsUsed int, reason string) {
	if tc == nil || tc.RouteLog == nil {
		return
	}
	args := []any{
		"msg", naming.MsgHarnessToolLoopFinished,
		"rounds_used", roundsUsed,
		"reason", reason,
		"timeline_kind", naming.TimelineKindBroker,
	}
	args = appendToolLogIdentity(args, tc, env)
	tc.RouteLog.Info("harness tool loop finished", args...)
}

func appendToolLogIdentity(args []any, tc *TurnContext, env *TurnEnvelope) []any {
	if env != nil && strings.TrimSpace(env.AssistantID) != "" {
		args = append(args, "assistant_id", env.AssistantID)
	}
	if tc != nil && strings.TrimSpace(tc.RequestID) != "" {
		args = append(args, "request_id", tc.RequestID)
	}
	if tc != nil && strings.TrimSpace(tc.ConversationID) != "" {
		args = append(args, "conversation_id", tc.ConversationID)
	}
	if tc != nil && tc.TurnIndex > 0 {
		args = append(args, "turn_index", tc.TurnIndex)
	}
	return args
}

func invokeAndLogTool(ctx context.Context, tc *TurnContext, env *TurnEnvelope, round int, call tools.ToolCall, executor tools.ToolExecutor) tools.ToolResult {
	dispatch := tools.DispatchFor(executor, call)
	path := redactedToolPath(call)
	argBytes := len(call.Arguments)
	toolCallID := strings.TrimSpace(call.ID)
	toolName := strings.TrimSpace(call.Name)

	if tc != nil && tc.RouteLog != nil {
		startArgs := []any{
			"msg", naming.MsgHarnessToolInvokeStarted,
			"round", round,
			"tool_name", toolName,
			"tool_backend", dispatch.Backend,
			"service", dispatch.Service,
			"arg_bytes", argBytes,
			"timeline_kind", naming.TimelineKindBroker,
		}
		if path != "" {
			startArgs = append(startArgs, "path", path)
		}
		if dispatch.MCPServer != "" {
			startArgs = append(startArgs, "mcp_server_id", dispatch.MCPServer)
		}
		if toolCallID != "" {
			startArgs = append(startArgs, "tool_call_id", toolCallID)
		}
		startArgs = appendToolLogIdentity(startArgs, tc, env)
		tc.RouteLog.Info("harness tool invoke started", startArgs...)
	}

	start := time.Now()
	result, err := executor.Invoke(ctx, call)
	latency := time.Since(start).Milliseconds()
	if err != nil {
		result.IsError = true
		if strings.TrimSpace(result.Content) == "" {
			result.Content = "tool failed"
		}
	}
	resultBytes := len(result.Content)
	failed := err != nil || result.IsError

	if tc != nil && tc.RouteLog != nil {
		doneArgs := []any{
			"round", round,
			"tool_name", toolName,
			"tool_backend", dispatch.Backend,
			"service", dispatch.Service,
			"latency_ms", latency,
			"result_bytes", resultBytes,
			"is_error", result.IsError,
			"timeline_kind", naming.TimelineKindBroker,
		}
		if path != "" {
			doneArgs = append(doneArgs, "path", path)
		}
		if dispatch.MCPServer != "" {
			doneArgs = append(doneArgs, "mcp_server_id", dispatch.MCPServer)
		}
		if toolCallID != "" {
			doneArgs = append(doneArgs, "tool_call_id", toolCallID)
		}
		if err != nil {
			doneArgs = append(doneArgs, "err", err.Error())
		}
		doneArgs = appendToolLogIdentity(doneArgs, tc, env)
		if failed {
			doneArgs = append([]any{"msg", naming.MsgHarnessToolInvokeFailed}, doneArgs...)
			tc.RouteLog.Warn("harness tool invoke failed", doneArgs...)
		} else {
			doneArgs = append([]any{"msg", naming.MsgHarnessToolInvokeCompleted}, doneArgs...)
			tc.RouteLog.Info("harness tool invoke completed", doneArgs...)
		}
	}
	return result
}

func redactedToolPath(call tools.ToolCall) string {
	var args struct {
		Path string `json:"path"`
	}
	if json.Unmarshal(call.Arguments, &args) != nil {
		return ""
	}
	path := strings.TrimSpace(args.Path)
	if path == "" {
		return ""
	}
	if len(path) > 200 {
		return path[:200] + "…"
	}
	return path
}
