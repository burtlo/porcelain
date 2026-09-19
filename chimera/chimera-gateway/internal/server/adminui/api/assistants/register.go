package assistants

import (
	"net/http"

	"github.com/lynn/porcelain/chimera/chimera-gateway/internal/server/adminui/handler"
)

// Register mounts assistant operator API routes.
func Register(mux *http.ServeMux, h *handler.Handler) {
	if h == nil {
		return
	}
	mux.HandleFunc("GET /api/ui/assistants", h.RequireAuthJSON(func(w http.ResponseWriter, r *http.Request) {
		handleListGET(h, w, r)
	}))
	mux.HandleFunc("POST /api/ui/assistants", h.RequireAuthJSON(func(w http.ResponseWriter, r *http.Request) {
		handleCreatePOST(h, w, r)
	}))
	mux.HandleFunc("GET /api/ui/assistants/{id}", h.RequireAuthJSON(func(w http.ResponseWriter, r *http.Request) {
		handleGetGET(h, w, r)
	}))
	mux.HandleFunc("PUT /api/ui/assistants/{id}", h.RequireAuthJSON(func(w http.ResponseWriter, r *http.Request) {
		handleUpdatePUT(h, w, r)
	}))
	mux.HandleFunc("DELETE /api/ui/assistants/{id}", h.RequireAuthJSON(func(w http.ResponseWriter, r *http.Request) {
		handleDeleteDELETE(h, w, r)
	}))
	mux.HandleFunc("PUT /api/ui/assistants/{id}/fallback", h.RequireAuthJSON(func(w http.ResponseWriter, r *http.Request) {
		handleFallbackPUT(h, w, r)
	}))
	mux.HandleFunc("PUT /api/ui/assistants/{id}/routing-policy", h.RequireAuthJSON(func(w http.ResponseWriter, r *http.Request) {
		handleRoutingPolicyPUT(h, w, r)
	}))
	mux.HandleFunc("PUT /api/ui/assistants/{id}/tool-router", h.RequireAuthJSON(func(w http.ResponseWriter, r *http.Request) {
		handleToolRouterPUT(h, w, r)
	}))
	mux.HandleFunc("POST /api/ui/assistants/{id}/routing/generate", h.RequireAuthJSON(func(w http.ResponseWriter, r *http.Request) {
		handleGeneratePOST(h, w, r)
	}))
	mux.HandleFunc("POST /api/ui/assistants/{id}/routing/evaluate", h.RequireAuthJSON(func(w http.ResponseWriter, r *http.Request) {
		handleEvaluatePOST(h, w, r)
	}))
}
