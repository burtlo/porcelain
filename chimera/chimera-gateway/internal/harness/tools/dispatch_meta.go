package tools

import "strings"

// DispatchMeta describes where a harness tool call runs for operator logs.
type DispatchMeta struct {
	Backend   string // native | mcp | expansion
	Service   string // chimera-gateway or MCP server id
	MCPServer string // set when Backend is mcp
}

const serviceChimeraGateway = "chimera-gateway"

// DispatchFor returns logging metadata for a tool call and executor implementation.
func DispatchFor(executor ToolExecutor, call ToolCall) DispatchMeta {
	name := strings.TrimSpace(call.Name)
	if isExpansionToolName(name) {
		return DispatchMeta{Backend: "expansion", Service: serviceChimeraGateway}
	}
	comp, ok := executor.(*CompositeExecutor)
	if !ok {
		return DispatchMeta{Backend: "native", Service: serviceChimeraGateway}
	}
	if IsReservedNativeName(name) {
		return DispatchMeta{Backend: "native", Service: serviceChimeraGateway}
	}
	route, routed := comp.Routes[name]
	if !routed {
		return DispatchMeta{Backend: "native", Service: serviceChimeraGateway}
	}
	sid := strings.TrimSpace(route.ServerID)
	if sid == "" {
		sid = "mcp"
	}
	return DispatchMeta{Backend: "mcp", Service: sid, MCPServer: sid}
}

func isExpansionToolName(name string) bool {
	switch strings.TrimSpace(name) {
	case "workspace_context_around", "workspace_adjacent_chunks", "workspace_read_lines":
		return true
	default:
		return false
	}
}
