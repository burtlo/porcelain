package runtime

import (
	"context"

	"github.com/lynn/porcelain/chimera/chimera-gateway/internal/mcpmgr"
	"github.com/lynn/porcelain/chimera/chimera-gateway/internal/operatorstore"
	"github.com/lynn/porcelain/chimera/internal/config"
)

// EffectiveMCP returns MCP config after applying operator SQLite overrides.
func (rt *Runtime) EffectiveMCP(ctx context.Context) config.MCP {
	if rt == nil {
		return config.MCP{}
	}
	rt.mu.RLock()
	base := rt.resolved
	store := rt.operator
	rt.mu.RUnlock()
	if base == nil {
		return config.MCP{}
	}
	return operatorstore.ResolveMCPFromStore(ctx, store, "", base.MCP)
}

// ReloadMCP rebuilds the gateway MCP manager from YAML + operator store.
func (rt *Runtime) ReloadMCP(ctx context.Context) {
	if rt == nil {
		return
	}
	rt.mu.Lock()
	defer rt.mu.Unlock()
	rt.reloadMCPLocked(ctx)
}

func (rt *Runtime) reloadMCPLocked(ctx context.Context) {
	if rt.mcpMgr != nil {
		rt.mcpMgr.Stop()
		rt.mcpMgr = nil
	}
	if rt.resolved == nil {
		return
	}
	mcp := operatorstore.ResolveMCPFromStore(ctx, rt.operator, "", rt.resolved.MCP)
	if !mcp.MCPConfigured() {
		return
	}
	rt.mcpMgr = mcpmgr.New(mcp, rt.log)
	rt.mcpMgr.Start(ctx)
}

// ChatResolved clones gateway resolved config with operator MCP merged for harness turns.
func (rt *Runtime) ChatResolved(ctx context.Context) *config.Resolved {
	rt.mu.RLock()
	base := rt.resolved
	store := rt.operator
	rt.mu.RUnlock()
	if base == nil {
		return nil
	}
	if store == nil {
		return base
	}
	has, err := store.HasMCPServerRows(ctx, "")
	if err != nil || !has {
		return base
	}
	cp := config.CloneResolved(base)
	cp.MCP = operatorstore.ResolveMCPFromStore(ctx, store, "", base.MCP)
	return cp
}
