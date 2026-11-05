package catalog

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"sort"
	"strings"

	"github.com/lynn/porcelain/chimera/chimera-gateway/internal/mcpclient"
	"github.com/lynn/porcelain/chimera/chimera-gateway/internal/mcpmgr"
	"github.com/lynn/porcelain/chimera/internal/config"
	"github.com/lynn/porcelain/internal/naming"
)

// ErrCatalogTruncated is returned when native + MCP tools exceed the configured cap (fail closed for MCP).
var ErrCatalogTruncated = errors.New("mcp catalog truncated")

// Route maps an OpenAI function.name to an MCP backend.
type Route struct {
	ToolKey     string
	ServerID    string
	MCPToolName string
	OpenAIName  string
}

// Scope selects MCP tools for one assistant turn.
type Scope struct {
	AssistantID string
	MCP         config.MCP
	Binding     *config.MCPAssistantBinding
	Cap         int
	Log         *slog.Logger
}

// Result is a successful catalog build.
type Result struct {
	MCPTools []map[string]any
	Routes   map[string]Route
}

type candidate struct {
	serverID    string
	mcpToolName string
	tool        mcpclient.Tool
	openai      string
	toolKey     string
}

// Build merges allowed MCP tools for scope. Native tools are counted toward cap but not returned here.
func Build(ctx context.Context, mgr *mcpmgr.Manager, scope Scope) (Result, error) {
	if mgr == nil || scope.Binding == nil {
		return Result{}, nil
	}
	cap := scope.Cap
	if cap <= 0 {
		cap = 32
	}
	const nativeCount = 4 // read_file, write_file, list_dir, search
	if nativeCount > cap {
		emitTruncated(scope.Log, cap, nativeCount)
		return Result{}, ErrCatalogTruncated
	}

	var candidates []candidate
	for _, bindSrv := range scope.Binding.Servers {
		serverID := strings.TrimSpace(bindSrv.ServerID)
		rec := scope.MCP.ServerByID(serverID)
		if rec == nil || rec.Disabled {
			continue
		}
		cli, err := mgr.Client(ctx, serverID)
		if err != nil {
			if scope.Log != nil {
				scope.Log.Warn("mcp tools/list skipped for server",
					"msg", naming.MsgToolsMcpError, "server_id", serverID, "err", err)
			}
			continue
		}
		listed, err := cli.ListTools(ctx)
		if err != nil {
			if scope.Log != nil {
				scope.Log.Warn("mcp tools/list failed; soft-fail server",
					"msg", naming.MsgToolsMcpError, "server_id", serverID, "err", err)
			}
			continue
		}
		for _, tool := range listed {
			name := strings.TrimSpace(tool.Name)
			if name == "" {
				continue
			}
			allowed, ok := bindSrv.Tools[name]
			if !ok || !allowed {
				continue
			}
			openai := mcpclient.OpenAIName(serverID, name)
			candidates = append(candidates, candidate{
				serverID:    serverID,
				mcpToolName: name,
				tool:        tool,
				openai:      openai,
				toolKey:     fmt.Sprintf("mcp:%s:%s", serverID, name),
			})
		}
	}

	sort.Slice(candidates, func(i, j int) bool {
		a, b := candidates[i], candidates[j]
		if a.serverID != b.serverID {
			return a.serverID < b.serverID
		}
		return a.mcpToolName < b.mcpToolName
	})
	candidates = dedupeOpenAINames(candidates)

	if nativeCount+len(candidates) > cap {
		emitTruncated(scope.Log, cap, nativeCount+len(candidates))
		return Result{}, ErrCatalogTruncated
	}

	out := Result{Routes: make(map[string]Route, len(candidates))}
	for _, c := range candidates {
		decl := mcpToolDecl(c.openai, c.tool)
		out.MCPTools = append(out.MCPTools, decl)
		out.Routes[c.openai] = Route{
			ToolKey:     c.toolKey,
			ServerID:    c.serverID,
			MCPToolName: c.mcpToolName,
			OpenAIName:  c.openai,
		}
	}
	return out, nil
}

func dedupeOpenAINames(in []candidate) []candidate {
	assigned := map[string]bool{}
	for i := range in {
		name := in[i].openai
		if !assigned[name] {
			assigned[name] = true
			continue
		}
		for n := 2; ; n++ {
			try := openaiWithSuffix(in[i].openai, n)
			if !assigned[try] {
				in[i].openai = try
				assigned[try] = true
				break
			}
		}
	}
	return in
}

func openaiWithSuffix(base string, n int) string {
	suffix := fmt.Sprintf("_%d", n)
	if len(base)+len(suffix) <= 64 {
		return base + suffix
	}
	trim := 64 - len(suffix)
	if trim < 1 {
		return suffix[1:]
	}
	return base[:trim] + suffix
}

func mcpToolDecl(openaiName string, tool mcpclient.Tool) map[string]any {
	params := map[string]any{"type": "object", "additionalProperties": true}
	if len(tool.InputSchema) > 0 {
		_ = json.Unmarshal(tool.InputSchema, &params)
	}
	desc := strings.TrimSpace(tool.Description)
	if desc == "" {
		desc = "MCP tool " + tool.Name
	}
	return map[string]any{
		"type": "function",
		"function": map[string]any{
			"name":        openaiName,
			"description": desc,
			"parameters":  params,
		},
	}
}

func emitTruncated(log *slog.Logger, cap, count int) {
	if log == nil {
		return
	}
	log.Warn("merged tool catalog exceeded cap; MCP tools not injected",
		"msg", naming.MsgToolsMcpCatalogTruncated, "cap", cap, "count", count)
}
