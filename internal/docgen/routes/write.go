package routes

import (
	"fmt"
	"io"
	"strings"
)

// WriteMarkdown emits the committed HTTP route catalog.
func WriteMarkdown(w io.Writer, routes []Route) error {
	var b strings.Builder
	b.WriteString("# Gateway HTTP routes (generated)\n\n")
	b.WriteString("DO NOT EDIT; run: `make contracts-generate` or `go run ./internal/docgen/routes/cmd`.\n\n")
	b.WriteString("Auth values: **public** (no gateway token), **bearer** (gateway API token), **session** (operator UI cookie), **tenant** (session + tenant scope for RAG).\n\n")

	b.WriteString("| Method | Path | Auth | Source | Notes |\n")
	b.WriteString("|--------|------|------|--------|-------|\n")
	for _, r := range routes {
		notes := strings.ReplaceAll(r.Notes, "|", "\\|")
		b.WriteString(fmt.Sprintf("| %s | `%s` | %s | `%s` | %s |\n",
			r.Method, r.Path, r.Auth, r.Source, notes))
	}
	b.WriteString("\n")
	_, err := io.WriteString(w, b.String())
	return err
}
