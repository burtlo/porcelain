package operatorstore

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/lynn/porcelain/chimera/internal/config"
)

// MCPServerRow is one operator-managed MCP server definition.
type MCPServerRow struct {
	ServerID     string
	TenantID     string
	Disabled     bool
	AutoStart    bool
	Transport    string
	Command      string
	Args         []string
	EnvAllowlist []string
	URL          string
	CreatedAt    time.Time
	UpdatedAt    time.Time
}

// MCPAssistantBindingRow binds one server to an assistant with per-tool toggles.
type MCPAssistantBindingRow struct {
	AssistantID int64
	ServerID    string
	Enabled     bool
	Tools       map[string]bool
}

// UpsertMCPServerInput is create/update payload for one server.
type UpsertMCPServerInput struct {
	ServerID     string
	Disabled     bool
	AutoStart    bool
	Transport    string
	Command      string
	Args         []string
	EnvAllowlist []string
	URL          string
}

func normalizeMCPServerInput(in UpsertMCPServerInput) (UpsertMCPServerInput, error) {
	in.ServerID = strings.TrimSpace(in.ServerID)
	if in.ServerID == "" {
		return UpsertMCPServerInput{}, fmt.Errorf("server_id required")
	}
	in.Transport = strings.TrimSpace(strings.ToLower(in.Transport))
	if in.Transport == "" {
		in.Transport = "stdio"
	}
	switch in.Transport {
	case "stdio", "http":
	default:
		return UpsertMCPServerInput{}, fmt.Errorf("invalid transport %q", in.Transport)
	}
	if in.Transport == "stdio" && strings.TrimSpace(in.Command) == "" {
		return UpsertMCPServerInput{}, fmt.Errorf("command required for stdio transport")
	}
	if in.Transport == "http" && strings.TrimSpace(in.URL) == "" {
		return UpsertMCPServerInput{}, fmt.Errorf("url required for http transport")
	}
	return in, nil
}

// HasMCPServerRows reports whether operator SQLite defines any MCP servers.
func (s *Store) HasMCPServerRows(ctx context.Context, tenantID string) (bool, error) {
	if s == nil {
		return false, nil
	}
	var n int
	err := s.db.QueryRowContext(ctx, `SELECT COUNT(*) FROM mcp_servers WHERE tenant_id = ?`, tenantID).Scan(&n)
	return n > 0, err
}

// ListMCPServers returns all MCP server rows for tenantID.
func (s *Store) ListMCPServers(ctx context.Context, tenantID string) ([]MCPServerRow, error) {
	if s == nil {
		return nil, fmt.Errorf("operator store nil")
	}
	rows, err := s.db.QueryContext(ctx, `
SELECT server_id, tenant_id, disabled, auto_start, transport, command, args_json, env_allowlist_json, url, created_at, updated_at
FROM mcp_servers WHERE tenant_id = ? ORDER BY server_id`, tenantID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []MCPServerRow
	for rows.Next() {
		var r MCPServerRow
		var disabled, autoStart int
		var argsJSON, envJSON, created, updated string
		if err := rows.Scan(&r.ServerID, &r.TenantID, &disabled, &autoStart, &r.Transport, &r.Command, &argsJSON, &envJSON, &r.URL, &created, &updated); err != nil {
			return nil, err
		}
		r.Disabled = disabled != 0
		r.AutoStart = autoStart != 0
		_ = json.Unmarshal([]byte(argsJSON), &r.Args)
		_ = json.Unmarshal([]byte(envJSON), &r.EnvAllowlist)
		r.CreatedAt, _ = time.Parse(time.RFC3339Nano, created)
		r.UpdatedAt, _ = time.Parse(time.RFC3339Nano, updated)
		out = append(out, r)
	}
	return out, rows.Err()
}

// UpsertMCPServer inserts or updates one MCP server row.
func (s *Store) UpsertMCPServer(ctx context.Context, tenantID string, in UpsertMCPServerInput) error {
	in, err := normalizeMCPServerInput(in)
	if err != nil {
		return err
	}
	now := time.Now().UTC().Format(time.RFC3339Nano)
	argsJSON, _ := json.Marshal(in.Args)
	envJSON, _ := json.Marshal(in.EnvAllowlist)
	_, err = s.db.ExecContext(ctx, `
INSERT INTO mcp_servers (server_id, tenant_id, disabled, auto_start, transport, command, args_json, env_allowlist_json, url, created_at, updated_at)
VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
ON CONFLICT(tenant_id, server_id) DO UPDATE SET
	disabled = excluded.disabled,
	auto_start = excluded.auto_start,
	transport = excluded.transport,
	command = excluded.command,
	args_json = excluded.args_json,
	env_allowlist_json = excluded.env_allowlist_json,
	url = excluded.url,
	updated_at = excluded.updated_at`,
		in.ServerID, tenantID, boolToInt(in.Disabled), boolToInt(in.AutoStart), in.Transport, in.Command, string(argsJSON), string(envJSON), in.URL, now, now)
	return err
}

// DeleteMCPServer removes a server and dependent bindings.
func (s *Store) DeleteMCPServer(ctx context.Context, tenantID, serverID string) error {
	serverID = strings.TrimSpace(serverID)
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback() }()
	if _, err := tx.ExecContext(ctx, `DELETE FROM mcp_assistant_tool_permissions WHERE tenant_id = ? AND server_id = ?`, tenantID, serverID); err != nil {
		return err
	}
	if _, err := tx.ExecContext(ctx, `DELETE FROM mcp_assistant_server_bindings WHERE tenant_id = ? AND server_id = ?`, tenantID, serverID); err != nil {
		return err
	}
	res, err := tx.ExecContext(ctx, `DELETE FROM mcp_servers WHERE tenant_id = ? AND server_id = ?`, tenantID, serverID)
	if err != nil {
		return err
	}
	n, _ := res.RowsAffected()
	if n == 0 {
		return fmt.Errorf("mcp server not found")
	}
	return tx.Commit()
}

// ListMCPAssistantBindings returns server bindings and tool permissions for one assistant.
func (s *Store) ListMCPAssistantBindings(ctx context.Context, tenantID string, assistantID int64) ([]MCPAssistantBindingRow, error) {
	if s == nil {
		return nil, fmt.Errorf("operator store nil")
	}
	rows, err := s.db.QueryContext(ctx, `
SELECT server_id, enabled FROM mcp_assistant_server_bindings
WHERE tenant_id = ? AND assistant_id = ? ORDER BY server_id`, tenantID, assistantID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var binds []MCPAssistantBindingRow
	for rows.Next() {
		var b MCPAssistantBindingRow
		var en int
		if err := rows.Scan(&b.ServerID, &en); err != nil {
			return nil, err
		}
		b.AssistantID = assistantID
		b.Enabled = en != 0
		b.Tools = map[string]bool{}
		binds = append(binds, b)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	toolRows, err := s.db.QueryContext(ctx, `
SELECT server_id, tool_name, enabled FROM mcp_assistant_tool_permissions
WHERE tenant_id = ? AND assistant_id = ?`, tenantID, assistantID)
	if err != nil {
		return nil, err
	}
	defer toolRows.Close()
	byServer := make(map[string]int)
	for i, b := range binds {
		byServer[b.ServerID] = i
	}
	for toolRows.Next() {
		var serverID, toolName string
		var en int
		if err := toolRows.Scan(&serverID, &toolName, &en); err != nil {
			return nil, err
		}
		idx, ok := byServer[serverID]
		if !ok {
			continue
		}
		binds[idx].Tools[strings.TrimSpace(toolName)] = en != 0
	}
	return binds, toolRows.Err()
}

// SetMCPAssistantBindings replaces MCP bindings and tool permissions for one assistant.
func (s *Store) SetMCPAssistantBindings(ctx context.Context, tenantID string, assistantID int64, bindings []MCPAssistantBindingRow) error {
	vm, err := s.GetAssistantByID(ctx, tenantID, assistantID)
	if err != nil {
		return err
	}
	if vm == nil {
		return fmt.Errorf("assistant not found")
	}
	now := time.Now().UTC().Format(time.RFC3339Nano)
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback() }()
	if _, err := tx.ExecContext(ctx, `DELETE FROM mcp_assistant_tool_permissions WHERE tenant_id = ? AND assistant_id = ?`, tenantID, assistantID); err != nil {
		return err
	}
	if _, err := tx.ExecContext(ctx, `DELETE FROM mcp_assistant_server_bindings WHERE tenant_id = ? AND assistant_id = ?`, tenantID, assistantID); err != nil {
		return err
	}
	for _, b := range bindings {
		serverID := strings.TrimSpace(b.ServerID)
		if serverID == "" {
			continue
		}
		if _, err := tx.ExecContext(ctx, `
INSERT INTO mcp_assistant_server_bindings (assistant_id, server_id, tenant_id, enabled, updated_at)
VALUES (?, ?, ?, ?, ?)`, assistantID, serverID, tenantID, boolToInt(b.Enabled), now); err != nil {
			return err
		}
		for name, en := range b.Tools {
			name = strings.TrimSpace(name)
			if name == "" {
				continue
			}
			if _, err := tx.ExecContext(ctx, `
INSERT INTO mcp_assistant_tool_permissions (assistant_id, server_id, tool_name, tenant_id, enabled, updated_at)
VALUES (?, ?, ?, ?, ?, ?)`, assistantID, serverID, name, tenantID, boolToInt(en), now); err != nil {
				return err
			}
		}
	}
	return tx.Commit()
}

// ResolveMCPFromStore merges operator MCP rows into YAML MCP config.
// When any mcp_servers rows exist for the tenant, SQLite is the source of truth for
// server definitions and assistant bindings; YAML servers[] and assistants[] are ignored.
// Global limits (catalog_max_tools, call timeout) always come from YAML.
func ResolveMCPFromStore(ctx context.Context, store *Store, tenantID string, yaml config.MCP) config.MCP {
	if store == nil {
		return yaml
	}
	has, err := store.HasMCPServerRows(ctx, tenantID)
	if err != nil || !has {
		return yaml
	}
	rows, err := store.ListMCPServers(ctx, tenantID)
	if err != nil {
		return yaml
	}
	out := yaml
	out.Servers = make([]config.MCPServer, 0, len(rows))
	for _, r := range rows {
		out.Servers = append(out.Servers, config.MCPServer{
			ID:           r.ServerID,
			Disabled:     r.Disabled,
			AutoStart:    r.AutoStart,
			Transport:    r.Transport,
			Command:      r.Command,
			Args:         append([]string(nil), r.Args...),
			EnvAllowlist: append([]string(nil), r.EnvAllowlist...),
			URL:          r.URL,
		})
	}
	assistants, err := store.ListAssistants(ctx, tenantID, tenantID)
	if err != nil {
		out.Assistants = nil
		return out
	}
	var bindings []config.MCPAssistantBinding
	for _, a := range assistants {
		bindRows, err := store.ListMCPAssistantBindings(ctx, tenantID, a.ID)
		if err != nil || len(bindRows) == 0 {
			continue
		}
		bind := config.MCPAssistantBinding{AssistantID: a.ModelID}
		for _, br := range bindRows {
			if !br.Enabled {
				continue
			}
			bind.Servers = append(bind.Servers, config.MCPServerBinding{
				ServerID: br.ServerID,
				Tools:    br.Tools,
			})
		}
		if len(bind.Servers) > 0 {
			bindings = append(bindings, bind)
		}
	}
	out.Assistants = bindings
	return out
}
