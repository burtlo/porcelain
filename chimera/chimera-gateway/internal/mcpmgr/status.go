package mcpmgr

import (
	"strings"

	"github.com/lynn/porcelain/chimera/internal/config"
)

// ServerStatus is operator-visible health for one MCP server.
type ServerStatus struct {
	ServerID string
	State    State
	Error    string
}

// ListServerStatus returns lifecycle state for each configured server id.
func (m *Manager) ListServerStatus() []ServerStatus {
	if m == nil {
		return nil
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	var out []ServerStatus
	for _, s := range m.cfg.Servers {
		id := strings.TrimSpace(s.ID)
		if id == "" {
			continue
		}
		st := StateStopped
		errMsg := ""
		if ms := m.servers[id]; ms != nil {
			st = ms.state
			if ms.err != nil {
				errMsg = ms.err.Error()
			}
		}
		out = append(out, ServerStatus{ServerID: id, State: st, Error: errMsg})
	}
	return out
}

// Status returns lifecycle state for one server id.
func (m *Manager) Status(serverID string) ServerStatus {
	serverID = strings.TrimSpace(serverID)
	if m == nil || serverID == "" {
		return ServerStatus{ServerID: serverID, State: StateStopped}
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	ms := m.servers[serverID]
	if ms == nil {
		return ServerStatus{ServerID: serverID, State: StateStopped}
	}
	errMsg := ""
	if ms.err != nil {
		errMsg = ms.err.Error()
	}
	return ServerStatus{ServerID: serverID, State: ms.state, Error: errMsg}
}

// Config returns the manager's current MCP configuration snapshot.
func (m *Manager) Config() config.MCP {
	if m == nil {
		return config.MCP{}
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.cfg
}
