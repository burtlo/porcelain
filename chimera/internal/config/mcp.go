package config

import (
	"log/slog"
	"strings"
	"time"
)

const (
	defaultMCPCatalogMaxTools = 32
	defaultMCPCallTimeout     = 60 * time.Second
)

// MCP holds gateway-owned MCP server definitions and assistant bindings (Phase 1 YAML seed).
type MCP struct {
	CatalogMaxTools int
	CallTimeout     time.Duration
	Servers         []MCPServer
	Assistants      []MCPAssistantBinding
}

// MCPServer is one MCP upstream record from chimera.yaml.
type MCPServer struct {
	ID           string
	Disabled     bool
	AutoStart    bool
	Transport    string // stdio | http
	Command      string
	Args         []string
	EnvAllowlist []string
	URL          string
}

// MCPAssistantBinding binds MCP servers/tools to one assistant model id.
type MCPAssistantBinding struct {
	AssistantID string
	Servers     []MCPServerBinding
}

// MCPServerBinding is per-server tool allowlist for an assistant.
type MCPServerBinding struct {
	ServerID string
	Tools    map[string]bool // mcp tool name -> enabled
}

// MCPConfigured reports whether any server definitions exist.
func (m MCP) MCPConfigured() bool {
	return len(m.Servers) > 0
}

// AssistantBinding returns the binding for assistantID, if any.
func (m MCP) AssistantBinding(assistantID string) *MCPAssistantBinding {
	assistantID = strings.TrimSpace(assistantID)
	for _, b := range m.Assistants {
		if strings.TrimSpace(b.AssistantID) == assistantID {
			return &b
		}
	}
	return nil
}

// ServerByID returns the server record for id.
func (m MCP) ServerByID(id string) *MCPServer {
	id = strings.TrimSpace(id)
	for _, s := range m.Servers {
		if strings.TrimSpace(s.ID) == id {
			return &s
		}
	}
	return nil
}

func parseMCPDoc(doc chimeraDoc, log *slog.Logger) MCP {
	out := MCP{
		CatalogMaxTools: defaultMCPCatalogMaxTools,
		CallTimeout:     defaultMCPCallTimeout,
	}
	if doc.MCP.CatalogMaxTools > 0 {
		out.CatalogMaxTools = doc.MCP.CatalogMaxTools
	}
	if doc.MCP.CallTimeoutMS > 0 {
		out.CallTimeout = time.Duration(doc.MCP.CallTimeoutMS) * time.Millisecond
	}
	for _, s := range doc.MCP.Servers {
		rec := MCPServer{
			ID:           strings.TrimSpace(s.ID),
			Disabled:     boolOrDefault(s.Disabled, false),
			AutoStart:    boolOrDefault(s.AutoStart, false),
			Transport:    strings.TrimSpace(s.Transport),
			Command:      strings.TrimSpace(s.Command),
			Args:         append([]string(nil), s.Args...),
			EnvAllowlist: append([]string(nil), s.EnvAllowlist...),
			URL:          strings.TrimSpace(s.URL),
		}
		if rec.Transport == "" {
			rec.Transport = "stdio"
		}
		if rec.ID == "" {
			if log != nil {
				log.Warn("mcp server missing id; skipping", "msg", "gateway.mcp.config_invalid")
			}
			continue
		}
		out.Servers = append(out.Servers, rec)
	}
	for _, a := range doc.MCP.Assistants {
		bind := MCPAssistantBinding{AssistantID: strings.TrimSpace(a.AssistantID)}
		for _, srv := range a.Servers {
			tools := make(map[string]bool, len(srv.Tools))
			for name, en := range srv.Tools {
				tools[strings.TrimSpace(name)] = en
			}
			bind.Servers = append(bind.Servers, MCPServerBinding{
				ServerID: strings.TrimSpace(srv.ServerID),
				Tools:    tools,
			})
		}
		if bind.AssistantID == "" {
			continue
		}
		out.Assistants = append(out.Assistants, bind)
	}
	return out
}
