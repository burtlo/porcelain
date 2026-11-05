package tools

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"strings"
	"time"

	"github.com/lynn/porcelain/chimera/chimera-gateway/internal/mcpmgr"
	"github.com/lynn/porcelain/chimera/chimera-gateway/internal/mcptools/catalog"
	"github.com/lynn/porcelain/internal/naming"
)

const (
	defaultMaxMCPArgBytes    = 64 << 10
	defaultMaxMCPResultBytes = 256 << 10
)

// CompositeExecutor dispatches tool calls to native workspace tools first, then MCP routes.
type CompositeExecutor struct {
	Native         ToolExecutor
	Routes         map[string]catalog.Route
	Manager        *mcpmgr.Manager
	Log            *slog.Logger
	CallTimeout    time.Duration
	MaxArgBytes    int
	MaxResultBytes int
}

// Invoke implements ToolExecutor.
func (c *CompositeExecutor) Invoke(ctx context.Context, call ToolCall) (ToolResult, error) {
	if c == nil {
		return ToolResult{Content: "tool executor unavailable", IsError: true}, fmt.Errorf("nil composite executor")
	}
	name := strings.TrimSpace(call.Name)
	if IsReservedNativeName(name) {
		return c.Native.Invoke(ctx, call)
	}
	route, ok := c.Routes[name]
	if !ok {
		return c.Native.Invoke(ctx, call)
	}
	return c.invokeMCP(ctx, call, route)
}

func (c *CompositeExecutor) invokeMCP(ctx context.Context, call ToolCall, route catalog.Route) (ToolResult, error) {
	if c.Manager == nil {
		return ToolResult{Content: "mcp backend unavailable", IsError: true}, fmt.Errorf("mcp manager nil")
	}
	maxArgs := c.MaxArgBytes
	if maxArgs <= 0 {
		maxArgs = defaultMaxMCPArgBytes
	}
	if len(call.Arguments) > maxArgs {
		if c.Log != nil {
			c.Log.Warn("mcp tool arguments over limit",
				"msg", naming.MsgToolsPolicyDenied, "tool", route.OpenAIName, "server_id", route.ServerID)
		}
		return ToolResult{Content: "tool arguments exceed size limit", IsError: true}, fmt.Errorf("arguments too large")
	}
	callCtx := ctx
	if c.CallTimeout > 0 {
		var cancel context.CancelFunc
		callCtx, cancel = context.WithTimeout(ctx, c.CallTimeout)
		defer cancel()
	}
	cli, err := c.Manager.Client(callCtx, route.ServerID)
	if err != nil {
		if c.Log != nil {
			c.Log.Warn("mcp tools/call transport error",
				"msg", naming.MsgToolsMcpError, "server_id", route.ServerID, "tool", route.MCPToolName, "err", err)
		}
		return ToolResult{Content: "mcp transport error", IsError: true}, err
	}
	raw, err := cli.CallTool(callCtx, route.MCPToolName, call.Arguments)
	if err != nil {
		if c.Log != nil {
			c.Log.Warn("mcp tools/call failed",
				"msg", naming.MsgToolsMcpError, "server_id", route.ServerID, "tool", route.MCPToolName, "err", err)
		}
		return ToolResult{Content: "mcp tool call failed", IsError: true}, err
	}
	content, isErr := mcpResultContent(raw, c.MaxResultBytes)
	if c.Log != nil && !isErr {
		c.Log.Info("mcp tools/call completed",
			"msg", naming.MsgToolsMcpInvoke, "server_id", route.ServerID, "tool", route.MCPToolName, "tool_key", route.ToolKey)
	}
	return ToolResult{Content: content, IsError: isErr}, nil
}

func mcpResultContent(raw json.RawMessage, maxBytes int) (string, bool) {
	if maxBytes <= 0 {
		maxBytes = defaultMaxMCPResultBytes
	}
	var parsed struct {
		Content []struct {
			Type string `json:"type"`
			Text string `json:"text"`
		} `json:"content"`
		IsError bool `json:"isError"`
	}
	if err := json.Unmarshal(raw, &parsed); err != nil {
		return strings.TrimSpace(string(raw)), false
	}
	var parts []string
	for _, block := range parsed.Content {
		if strings.TrimSpace(block.Text) != "" {
			parts = append(parts, block.Text)
		}
	}
	out := strings.Join(parts, "\n")
	if out == "" {
		out = strings.TrimSpace(string(raw))
	}
	if len(out) > maxBytes {
		out = out[:maxBytes] + "…"
	}
	return out, parsed.IsError
}
