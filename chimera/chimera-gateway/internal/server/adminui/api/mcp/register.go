package mcp

import (
	"net/http"

	"github.com/lynn/porcelain/chimera/chimera-gateway/internal/server/adminui/handler"
)

// Register mounts operator MCP settings API routes.
func Register(mux *http.ServeMux, h *handler.Handler) {
	if h == nil {
		return
	}
	mux.HandleFunc("GET /api/ui/mcp/servers", h.RequireAuthJSON(func(w http.ResponseWriter, r *http.Request) {
		handleServersGET(h, w, r)
	}))
	mux.HandleFunc("PUT /api/ui/mcp/servers/{server_id}", h.RequireAuthJSON(func(w http.ResponseWriter, r *http.Request) {
		handleServerPUT(h, w, r)
	}))
	mux.HandleFunc("GET /api/ui/mcp/servers/{server_id}/tools", h.RequireAuthJSON(func(w http.ResponseWriter, r *http.Request) {
		handleServerToolsGET(h, w, r)
	}))
	mux.HandleFunc("GET /api/ui/assistants/{id}/mcp", h.RequireAuthJSON(func(w http.ResponseWriter, r *http.Request) {
		handleAssistantMCPGET(h, w, r)
	}))
	mux.HandleFunc("PUT /api/ui/assistants/{id}/mcp", h.RequireAuthJSON(func(w http.ResponseWriter, r *http.Request) {
		handleAssistantMCPPUT(h, w, r)
	}))
}
