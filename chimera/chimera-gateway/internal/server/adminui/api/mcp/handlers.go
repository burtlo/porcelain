package mcp

import (
	"context"
	"encoding/json"
	"net/http"
	"strings"

	"github.com/lynn/porcelain/chimera/chimera-gateway/internal/mcpmgr"
	"github.com/lynn/porcelain/chimera/chimera-gateway/internal/operatorstore"
	"github.com/lynn/porcelain/chimera/chimera-gateway/internal/server/adminui/handler"
	"github.com/lynn/porcelain/chimera/internal/config"
	"github.com/lynn/porcelain/internal/operatorapi"
)

const operatorTenantID = ""

func operatorStore(h *handler.Handler) *operatorstore.Store {
	if h == nil || h.RT == nil {
		return nil
	}
	return h.RT.OperatorStore()
}

func effectiveMCP(h *handler.Handler, ctx context.Context) config.MCP {
	if h == nil || h.RT == nil {
		return config.MCP{}
	}
	h.RT.Sync()
	return h.RT.EffectiveMCP(ctx)
}

func handleServersGET(h *handler.Handler, w http.ResponseWriter, r *http.Request) {
	st := operatorStore(h)
	ctx := r.Context()
	mcpCfg := effectiveMCP(h, ctx)
	source := "yaml"
	var rows []operatorstore.MCPServerRow
	if st != nil {
		has, err := st.HasMCPServerRows(ctx, operatorTenantID)
		if err == nil && has {
			source = "sqlite"
			rows, err = st.ListMCPServers(ctx, operatorTenantID)
			if err != nil {
				http.Error(w, err.Error(), http.StatusInternalServerError)
				return
			}
		}
	}
	mgr := h.RT.MCPManager()
	statusByID := map[string]mcpmgr.ServerStatus{}
	if mgr != nil {
		for _, st := range mgr.ListServerStatus() {
			statusByID[st.ServerID] = st
		}
	}
	var servers []operatorapi.MCPServerSummary
	if source == "sqlite" {
		for _, row := range rows {
			servers = append(servers, serverSummaryFromRow(row, statusByID[row.ServerID]))
		}
	} else {
		for _, s := range mcpCfg.Servers {
			servers = append(servers, serverSummaryFromConfig(s, statusByID[s.ID]))
		}
	}
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(operatorapi.MCPServersListResponse{
		Servers: servers,
		Source:  source,
	})
}

func serverSummaryFromRow(row operatorstore.MCPServerRow, st mcpmgr.ServerStatus) operatorapi.MCPServerSummary {
	state := string(st.State)
	if state == "" {
		state = "stopped"
	}
	return operatorapi.MCPServerSummary{
		ServerID:     row.ServerID,
		Disabled:     row.Disabled,
		AutoStart:    row.AutoStart,
		Transport:    row.Transport,
		Command:      row.Command,
		Args:         row.Args,
		EnvAllowlist: row.EnvAllowlist,
		URL:          row.URL,
		State:        state,
		Error:        st.Error,
		UpdatedAt:    row.UpdatedAt.UTC().Format("2006-01-02T15:04:05Z"),
	}
}

func serverSummaryFromConfig(s config.MCPServer, st mcpmgr.ServerStatus) operatorapi.MCPServerSummary {
	state := string(st.State)
	if state == "" {
		state = "stopped"
	}
	return operatorapi.MCPServerSummary{
		ServerID:     s.ID,
		Disabled:     s.Disabled,
		AutoStart:    s.AutoStart,
		Transport:    s.Transport,
		Command:      s.Command,
		Args:         s.Args,
		EnvAllowlist: s.EnvAllowlist,
		URL:          s.URL,
		State:        state,
		Error:        st.Error,
	}
}

func handleServerPUT(h *handler.Handler, w http.ResponseWriter, r *http.Request) {
	st := operatorStore(h)
	if st == nil {
		http.Error(w, "operator store unavailable", http.StatusServiceUnavailable)
		return
	}
	serverID := strings.TrimSpace(r.PathValue("server_id"))
	if serverID == "" {
		http.Error(w, "server_id required", http.StatusBadRequest)
		return
	}
	var body operatorapi.MCPServerUpsertRequest
	if err := json.NewDecoder(http.MaxBytesReader(w, r.Body, 1<<20)).Decode(&body); err != nil {
		http.Error(w, "invalid json", http.StatusBadRequest)
		return
	}
	if err := st.UpsertMCPServer(r.Context(), operatorTenantID, operatorstore.UpsertMCPServerInput{
		ServerID:     serverID,
		Disabled:     body.Disabled,
		AutoStart:    body.AutoStart,
		Transport:    body.Transport,
		Command:      body.Command,
		Args:         body.Args,
		EnvAllowlist: body.EnvAllowlist,
		URL:          body.URL,
	}); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	h.RT.ReloadMCP(r.Context())
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(map[string]any{"ok": true})
}

func handleServerToolsGET(h *handler.Handler, w http.ResponseWriter, r *http.Request) {
	serverID := strings.TrimSpace(r.PathValue("server_id"))
	if serverID == "" {
		http.Error(w, "server_id required", http.StatusBadRequest)
		return
	}
	mgr := h.RT.MCPManager()
	if mgr == nil {
		http.Error(w, "mcp not configured", http.StatusServiceUnavailable)
		return
	}
	st := mgr.Status(serverID)
	resp := operatorapi.MCPServerToolsResponse{
		ServerID: serverID,
		State:    string(st.State),
		Error:    st.Error,
		Tools:    []string{},
	}
	if st.State != mcpmgr.StateError && st.State != mcpmgr.StateStopped {
		cli, err := mgr.Client(r.Context(), serverID)
		if err != nil {
			resp.State = string(mcpmgr.StateError)
			resp.Error = err.Error()
		} else {
			listed, err := cli.ListTools(r.Context())
			if err != nil {
				resp.State = string(mcpmgr.StateError)
				resp.Error = err.Error()
			} else {
				for _, t := range listed {
					if n := strings.TrimSpace(t.Name); n != "" {
						resp.Tools = append(resp.Tools, n)
					}
				}
			}
		}
	} else if st.Error != "" {
		resp.Error = st.Error
	}
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(resp)
}

func handleAssistantMCPGET(h *handler.Handler, w http.ResponseWriter, r *http.Request) {
	st := operatorStore(h)
	if st == nil {
		http.Error(w, "operator store unavailable", http.StatusServiceUnavailable)
		return
	}
	id, ok := parseAssistantID(r)
	if !ok {
		http.Error(w, "invalid id", http.StatusBadRequest)
		return
	}
	rows, err := st.ListMCPAssistantBindings(r.Context(), operatorTenantID, id)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(operatorapi.MCPAssistantBindingsResponse{
		Bindings: bindingsAPI(rows),
	})
}

func handleAssistantMCPPUT(h *handler.Handler, w http.ResponseWriter, r *http.Request) {
	st := operatorStore(h)
	if st == nil {
		http.Error(w, "operator store unavailable", http.StatusServiceUnavailable)
		return
	}
	id, ok := parseAssistantID(r)
	if !ok {
		http.Error(w, "invalid id", http.StatusBadRequest)
		return
	}
	var body operatorapi.MCPAssistantBindingsSaveRequest
	if err := json.NewDecoder(http.MaxBytesReader(w, r.Body, 1<<20)).Decode(&body); err != nil {
		http.Error(w, "invalid json", http.StatusBadRequest)
		return
	}
	rows := make([]operatorstore.MCPAssistantBindingRow, 0, len(body.Bindings))
	for _, b := range body.Bindings {
		tools := make(map[string]bool)
		for _, t := range b.Tools {
			name := strings.TrimSpace(t.ToolName)
			if name == "" {
				continue
			}
			tools[name] = t.Enabled
		}
		rows = append(rows, operatorstore.MCPAssistantBindingRow{
			AssistantID: id,
			ServerID:    b.ServerID,
			Enabled:     b.Enabled,
			Tools:       tools,
		})
	}
	if err := st.SetMCPAssistantBindings(r.Context(), operatorTenantID, id, rows); err != nil {
		if strings.Contains(err.Error(), "not found") {
			http.NotFound(w, r)
			return
		}
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	h.RT.ReloadMCP(r.Context())
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(operatorapi.MCPAssistantBindingsResponse{
		Bindings: bindingsAPI(rows),
	})
}

func bindingsAPI(rows []operatorstore.MCPAssistantBindingRow) []operatorapi.MCPAssistantServerBinding {
	out := make([]operatorapi.MCPAssistantServerBinding, 0, len(rows))
	for _, r := range rows {
		b := operatorapi.MCPAssistantServerBinding{
			ServerID: r.ServerID,
			Enabled:  r.Enabled,
			Tools:    []operatorapi.MCPAssistantToolPermission{},
		}
		for name, en := range r.Tools {
			b.Tools = append(b.Tools, operatorapi.MCPAssistantToolPermission{
				ToolName: name,
				Enabled:  en,
			})
		}
		out = append(out, b)
	}
	return out
}

func parseAssistantID(r *http.Request) (int64, bool) {
	raw := strings.TrimSpace(r.PathValue("id"))
	if raw == "" {
		return 0, false
	}
	var id int64
	for _, c := range raw {
		if c < '0' || c > '9' {
			return 0, false
		}
		id = id*10 + int64(c-'0')
	}
	return id, id > 0
}
