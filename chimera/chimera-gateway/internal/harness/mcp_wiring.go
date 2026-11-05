package harness

import (
	"context"
	"encoding/json"

	"github.com/lynn/porcelain/chimera/chimera-gateway/internal/harness/tools"
	"github.com/lynn/porcelain/chimera/chimera-gateway/internal/mcptools/catalog"
	"github.com/lynn/porcelain/chimera/chimera-gateway/internal/operatorstore"
)

// wireToolExecutor builds native and optional MCP composite executors plus injected tool declarations.
func wireToolExecutor(ctx context.Context, tc *TurnContext, native tools.ToolExecutor) {
	if tc == nil || native == nil {
		return
	}
	decls := tools.OpenAIToolDeclarations(nativeExpansionEnabled(tc))
	tc.ToolExecutor = native
	tc.ToolDeclarations = decls

	if tc.MCPManager == nil || tc.Resolved == nil || !tc.Resolved.MCP.MCPConfigured() {
		return
	}
	if tc.Stack.VM == nil || !tc.Stack.VM.HarnessEnabled(operatorstore.HarnessModuleToolExecutor) {
		return
	}
	if !toolExecutorMCPEnabled(tc) {
		return
	}
	bind := tc.Resolved.MCP.AssistantBinding(tc.AssistantID())
	if bind == nil {
		return
	}
	cap := tc.Resolved.MCP.CatalogMaxTools
	built, err := catalog.Build(ctx, tc.MCPManager, catalog.Scope{
		AssistantID: tc.AssistantID(),
		MCP:         tc.Resolved.MCP,
		Binding:     bind,
		Cap:         cap,
		Log:         tc.RouteLog,
	})
	if err != nil {
		return
	}
	if len(built.MCPTools) == 0 {
		return
	}
	merged := append(append([]map[string]any(nil), decls...), built.MCPTools...)
	tc.ToolDeclarations = merged
	tc.ToolExecutor = &tools.CompositeExecutor{
		Native:         native,
		Routes:         built.Routes,
		Manager:        tc.MCPManager,
		Log:            tc.RouteLog,
		CallTimeout:    tc.Resolved.MCP.CallTimeout,
		MaxArgBytes:    0,
		MaxResultBytes: 0,
	}
}

func toolExecutorMCPEnabled(tc *TurnContext) bool {
	if tc == nil || tc.Stack.VM == nil {
		return false
	}
	module, ok := tc.Stack.VM.HarnessModules[operatorstore.HarnessModuleToolExecutor]
	if !ok || !module.Enabled {
		return false
	}
	var cfg struct {
		MCP *struct {
			Enabled *bool `json:"enabled"`
		} `json:"mcp"`
	}
	if json.Unmarshal([]byte(module.ConfigJSON), &cfg) != nil || cfg.MCP == nil || cfg.MCP.Enabled == nil {
		return true
	}
	return *cfg.MCP.Enabled
}
