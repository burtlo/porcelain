package routes

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
)

// Route is one registered HTTP surface entry extracted from source.
type Route struct {
	Method string
	Path   string
	Auth   string
	Source string // file:line
	Notes  string
}

var (
	reHandleMethodPath = regexp.MustCompile(`mux\.HandleFunc\("([A-Z]+) ([^"]+)"`)
	reHandlePathOnly   = regexp.MustCompile(`mux\.HandleFunc\("([^"]+)"`)
	reHandleConcat     = regexp.MustCompile(`mux\.HandleFunc\("([A-Z]+) ([^"]*)"\+\w+\+"([^"]*)"`)
	reHandleContract   = regexp.MustCompile(`mux\.HandleFunc\(contract\.(HealthPath|ReadyPath|MetricsPath)`)
	reMethodCase       = regexp.MustCompile(`case http\.Method([A-Za-z]+):`)
	reMethodNotR       = regexp.MustCompile(`r\.Method != http\.Method([A-Za-z]+)`)
	reMethodNotReq     = regexp.MustCompile(`req\.Method != http\.Method([A-Za-z]+)`)
)

var contractPaths = map[string]string{
	"HealthPath":  "/healthz",
	"ReadyPath":   "/readyz",
	"MetricsPath": "/metrics",
}

// Collect scans configured source files under repoRoot.
func Collect(repoRoot string) ([]Route, error) {
	var out []Route
	for _, rel := range SourceFiles() {
		path := filepath.Join(repoRoot, filepath.FromSlash(rel))
		routes, err := scanFile(rel, path)
		if err != nil {
			return nil, err
		}
		out = append(out, routes...)
	}
	sort.Slice(out, func(i, j int) bool {
		if out[i].Source != out[j].Source {
			return out[i].Source < out[j].Source
		}
		if out[i].Path != out[j].Path {
			return out[i].Path < out[j].Path
		}
		return out[i].Method < out[j].Method
	})
	return out, nil
}

func scanFile(rel, abs string) ([]Route, error) {
	f, err := os.Open(abs)
	if err != nil {
		return nil, fmt.Errorf("routes: open %s: %w", rel, err)
	}
	defer f.Close()

	var lines []string
	sc := bufio.NewScanner(f)
	for sc.Scan() {
		lines = append(lines, sc.Text())
	}
	if err := sc.Err(); err != nil {
		return nil, fmt.Errorf("routes: read %s: %w", rel, err)
	}

	var routes []Route
	for i, line := range lines {
		lineno := i + 1
		trim := strings.TrimSpace(line)
		if !strings.Contains(trim, "mux.HandleFunc(") {
			continue
		}

		if m := reHandleContract.FindStringSubmatch(trim); len(m) == 2 {
			p := contractPaths[m[1]]
			method := inferMethods(lines, i)
			routes = append(routes, Route{
				Method: method,
				Path:   p,
				Auth:   inferAuth(trim, p, rel),
				Source: fmt.Sprintf("%s:%d", rel, lineno),
				Notes:  inferNotes(rel, p),
			})
			continue
		}

		if m := reHandleConcat.FindStringSubmatch(trim); len(m) == 4 {
			path := m[2] + "{provider}" + m[3]
			routes = append(routes, Route{
				Method: m[1],
				Path:   path,
				Auth:   inferAuth(trim, path, rel),
				Source: fmt.Sprintf("%s:%d", rel, lineno),
				Notes:  inferNotes(rel, path),
			})
			continue
		}

		if m := reHandleMethodPath.FindStringSubmatch(trim); len(m) == 3 {
			routes = append(routes, Route{
				Method: m[1],
				Path:   m[2],
				Auth:   inferAuth(trim, m[2], rel),
				Source: fmt.Sprintf("%s:%d", rel, lineno),
				Notes:  inferNotes(rel, m[2]),
			})
			continue
		}

		if m := reHandlePathOnly.FindStringSubmatch(trim); len(m) == 2 {
			p := m[1]
			if strings.Contains(p, " ") {
				continue
			}
			method := inferMethods(lines, i)
			routes = append(routes, Route{
				Method: method,
				Path:   p,
				Auth:   inferAuth(trim, p, rel),
				Source: fmt.Sprintf("%s:%d", rel, lineno),
				Notes:  inferNotes(rel, p),
			})
		}
	}
	return routes, nil
}

func inferMethods(lines []string, start int) string {
	var methods []string
	limit := start + 40
	if limit > len(lines) {
		limit = len(lines)
	}
	inSwitch := false
	for i := start; i < limit; i++ {
		line := lines[i]
		if strings.Contains(line, "switch r.Method") || strings.Contains(line, "switch req.Method") {
			inSwitch = true
			continue
		}
		if inSwitch {
			if strings.Contains(line, "default:") {
				break
			}
			if m := reMethodCase.FindStringSubmatch(line); len(m) == 2 {
				methods = append(methods, strings.ToUpper(m[1]))
			}
			continue
		}
		if m := reMethodNotR.FindStringSubmatch(line); len(m) == 2 {
			return strings.ToUpper(m[1])
		}
		if m := reMethodNotReq.FindStringSubmatch(line); len(m) == 2 {
			return strings.ToUpper(m[1])
		}
	}
	if len(methods) > 0 {
		return strings.Join(methods, ",")
	}
	return "*"
}

func inferAuth(line, path, rel string) string {
	switch {
	case strings.Contains(line, "RequireAuthTenantJSON"):
		return "tenant"
	case strings.Contains(line, "RequireAuthJSON"):
		return "session"
	case strings.Contains(line, "RequireAuthPage"):
		return "session"
	}

	if strings.Contains(rel, "chimera-supervisor") {
		return "public"
	}

	if strings.HasPrefix(path, "/v1/") {
		return "bearer"
	}

	if strings.HasPrefix(path, "/api/ui/setup") {
		return "public"
	}
	if path == "/api/ui/login" || path == "/ui/login" {
		return "public"
	}
	if strings.HasSuffix(path, "/api/ui/logout") {
		return "session"
	}

	if strings.HasPrefix(path, "/api/ui/") {
		return "session"
	}

	if strings.HasPrefix(path, "/ui") {
		if strings.HasPrefix(path, "/ui/assets/") && !strings.Contains(line, "RequireAuth") {
			return "public"
		}
		if strings.Contains(line, "RequireAuth") {
			return "session"
		}
	}

	return "public"
}

func inferNotes(rel, path string) string {
	if strings.Contains(rel, "ui_bootstrap.go") {
		return "UI bootstrap mux (pre-operator SQLite)"
	}
	if strings.Contains(rel, "chimera-supervisor") {
		return "chimera-supervisor control plane"
	}
	if strings.Contains(rel, "adminui") {
		return "operator UI (ui != nil)"
	}
	if path == "/" {
		return "redirects to /ui when operator UI enabled"
	}
	if path == "/ui/models" {
		return "merged models without bearer (browser)"
	}
	if path == "/shutdown" {
		return "POST only; graceful shutdown hook"
	}
	return ""
}
