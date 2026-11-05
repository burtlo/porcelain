package operatorapi

// MCPServerSummary is one row in GET /api/ui/mcp/servers.
type MCPServerSummary struct {
	ServerID     string   `json:"server_id"`
	Disabled     bool     `json:"disabled"`
	AutoStart    bool     `json:"auto_start"`
	Transport    string   `json:"transport"`
	Command      string   `json:"command,omitempty"`
	Args         []string `json:"args,omitempty"`
	EnvAllowlist []string `json:"env_allowlist,omitempty"`
	URL          string   `json:"url,omitempty"`
	State        string   `json:"state"`
	Error        string   `json:"error,omitempty"`
	UpdatedAt    string   `json:"updated_at,omitempty"`
}

// MCPServersListResponse is GET /api/ui/mcp/servers.
type MCPServersListResponse struct {
	Servers []MCPServerSummary `json:"servers"`
	Source  string             `json:"source"` // yaml | sqlite
}

// MCPServerUpsertRequest is PUT /api/ui/mcp/servers/{server_id}.
type MCPServerUpsertRequest struct {
	Disabled     bool     `json:"disabled"`
	AutoStart    bool     `json:"auto_start"`
	Transport    string   `json:"transport"`
	Command      string   `json:"command"`
	Args         []string `json:"args"`
	EnvAllowlist []string `json:"env_allowlist"`
	URL          string   `json:"url"`
}

// MCPAssistantToolPermission is one tool toggle for an assistant binding.
type MCPAssistantToolPermission struct {
	ToolName string `json:"tool_name"`
	Enabled  bool   `json:"enabled"`
}

// MCPAssistantServerBinding is one server bound to an assistant.
type MCPAssistantServerBinding struct {
	ServerID string                       `json:"server_id"`
	Enabled  bool                         `json:"enabled"`
	Tools    []MCPAssistantToolPermission `json:"tools"`
}

// MCPAssistantBindingsResponse is GET /api/ui/assistants/{id}/mcp.
type MCPAssistantBindingsResponse struct {
	Bindings []MCPAssistantServerBinding `json:"bindings"`
}

// MCPAssistantBindingsSaveRequest is PUT /api/ui/assistants/{id}/mcp.
type MCPAssistantBindingsSaveRequest struct {
	Bindings []MCPAssistantServerBinding `json:"bindings"`
}

// MCPServerToolsResponse is GET /api/ui/mcp/servers/{server_id}/tools.
type MCPServerToolsResponse struct {
	ServerID string   `json:"server_id"`
	State    string   `json:"state"`
	Error    string   `json:"error,omitempty"`
	Tools    []string `json:"tools"`
}
