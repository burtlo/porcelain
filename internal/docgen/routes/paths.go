package routes

import (
	"path/filepath"
)

// DefaultMarkdownPath is repo-relative output for the HTTP route catalog.
const DefaultMarkdownPath = "docs/generated/gateway-http-routes.md"

// SourceFiles returns repo-relative Go files scanned for mux.HandleFunc registrations.
func SourceFiles() []string {
	gw := filepath.Join("chimera", "chimera-gateway", "internal", "server")
	admin := filepath.Join(gw, "adminui")
	return []string{
		filepath.Join(gw, "server.go"),
		filepath.Join(gw, "ui_bootstrap.go"),
		filepath.Join(admin, "embed", "routes.go"),
		filepath.Join(admin, "api", "assistants", "register.go"),
		filepath.Join(admin, "api", "auth", "register.go"),
		filepath.Join(admin, "api", "conversations", "register.go"),
		filepath.Join(admin, "api", "indexer", "register.go"),
		filepath.Join(admin, "api", "logs", "register.go"),
		filepath.Join(admin, "api", "metrics", "register.go"),
		filepath.Join(admin, "api", "providers", "register.go"),
		filepath.Join(admin, "api", "rag", "register.go"),
		filepath.Join(admin, "api", "save", "register.go"),
		filepath.Join(admin, "api", "state", "register.go"),
		filepath.Join(admin, "api", "tokens", "register.go"),
		filepath.Join("chimera", "chimera-supervisor", "internal", "control", "http.go"),
	}
}
