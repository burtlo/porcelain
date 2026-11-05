package mcpmgr

import (
	"github.com/lynn/porcelain/chimera/chimera-gateway/internal/mcpclient"
	"github.com/lynn/porcelain/chimera/internal/config"
)

// NewForTest wires pre-handshaken clients for catalog and harness tests.
func NewForTest(clients map[string]*mcpclient.Client) *Manager {
	m := &Manager{cfg: config.MCP{}, servers: make(map[string]*managed)}
	for id, cli := range clients {
		m.servers[id] = &managed{
			cfg:   config.MCPServer{ID: id, Transport: "stdio"},
			state: StateRunning,
			cli:   cli,
		}
	}
	return m
}
