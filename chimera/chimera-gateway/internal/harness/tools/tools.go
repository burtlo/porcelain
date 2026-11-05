// Package tools implements gateway-native workspace file tools.
package tools

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

const (
	PolicyNone      = "none"
	PolicyRead      = "read"
	PolicyReadWrite = "read_write"
)

var reservedNativeNames = map[string]bool{
	"read_file":                 true,
	"write_file":                true,
	"list_dir":                  true,
	"search":                    true,
	"workspace_context_around":  true,
	"workspace_adjacent_chunks": true,
	"workspace_read_lines":      true,
}

// IsReservedNativeName reports whether name is dispatched only by the native workspace executor.
func IsReservedNativeName(name string) bool {
	return reservedNativeNames[strings.TrimSpace(name)]
}

var (
	ErrPermission = errors.New("workspace file action is not permitted")
	ErrPath       = errors.New("path is outside configured workspace roots")
)

// ToolCall is the OpenAI-compatible function call requested by a completion.
type ToolCall struct {
	ID        string
	Name      string
	Arguments json.RawMessage
}

// ToolResult is safe to insert into a model-facing tool message. Paths are
// always relative to an approved root.
type ToolResult struct {
	Content string
	IsError bool
}

// ToolExecutor executes one gateway-native workspace tool.
type ToolExecutor interface {
	Invoke(context.Context, ToolCall) (ToolResult, error)
}

// WorkspaceExecutor constrains native tools to the resolved workspace roots.
type WorkspaceExecutor struct {
	Roots  []string
	Policy string
}

func NewWorkspaceExecutor(roots []string, policy string) *WorkspaceExecutor {
	clean := make([]string, 0, len(roots))
	for _, root := range roots {
		if abs, err := filepath.Abs(root); err == nil {
			clean = append(clean, filepath.Clean(abs))
		}
	}
	return &WorkspaceExecutor{Roots: clean, Policy: policy}
}

func (e *WorkspaceExecutor) Invoke(ctx context.Context, call ToolCall) (ToolResult, error) {
	if err := ctx.Err(); err != nil {
		return ToolResult{}, err
	}
	if e == nil || e.Policy == PolicyNone {
		return ToolResult{Content: "workspace file actions are disabled by policy", IsError: true}, ErrPermission
	}
	switch call.Name {
	case "read_file":
		return e.readFile(call.Arguments)
	case "list_dir":
		return e.listDir(call.Arguments)
	case "search":
		return e.search(call.Arguments)
	case "write_file":
		if e.Policy != PolicyReadWrite {
			return ToolResult{Content: "workspace policy permits reads only", IsError: true}, ErrPermission
		}
		return e.writeFile(call.Arguments)
	default:
		return ToolResult{Content: "unknown workspace tool", IsError: true}, fmt.Errorf("unknown tool %q", call.Name)
	}
}

type pathArgs struct {
	Path string `json:"path"`
}

func (e *WorkspaceExecutor) resolve(path string) (string, string, error) {
	path = strings.TrimSpace(path)
	if path == "" || filepath.IsAbs(path) {
		return "", "", ErrPath
	}
	for _, root := range e.Roots {
		candidate := filepath.Clean(filepath.Join(root, path))
		rel, err := filepath.Rel(root, candidate)
		if err != nil || rel == ".." || strings.HasPrefix(rel, ".."+string(filepath.Separator)) {
			continue
		}
		// Clean prevents lexical traversal; resolve existing symlinks as well so
		// a workspace-owned symlink cannot escape the approved root.
		realRoot, err := filepath.EvalSymlinks(root)
		if err != nil {
			continue
		}
		probe := candidate
		for {
			if _, err := os.Lstat(probe); err == nil {
				break
			}
			parent := filepath.Dir(probe)
			if parent == probe {
				break
			}
			probe = parent
		}
		realProbe, err := filepath.EvalSymlinks(probe)
		if err != nil {
			continue
		}
		remaining, err := filepath.Rel(probe, candidate)
		if err != nil {
			continue
		}
		realCandidate := filepath.Join(realProbe, remaining)
		realRel, err := filepath.Rel(realRoot, realCandidate)
		if err != nil || realRel == ".." || strings.HasPrefix(realRel, ".."+string(filepath.Separator)) {
			continue
		}
		return candidate, filepath.ToSlash(rel), nil
	}
	return "", "", ErrPath
}

func (e *WorkspaceExecutor) readFile(raw json.RawMessage) (ToolResult, error) {
	var args pathArgs
	if err := json.Unmarshal(raw, &args); err != nil {
		return ToolResult{Content: "invalid read_file arguments", IsError: true}, err
	}
	path, rel, err := e.resolve(args.Path)
	if err != nil {
		return ToolResult{Content: "path must be relative to a workspace root", IsError: true}, err
	}
	data, err := os.ReadFile(path)
	if err != nil {
		return ToolResult{Content: fmt.Sprintf("cannot read %s", rel), IsError: true}, err
	}
	return ToolResult{Content: string(data)}, nil
}

func (e *WorkspaceExecutor) listDir(raw json.RawMessage) (ToolResult, error) {
	var args pathArgs
	if err := json.Unmarshal(raw, &args); err != nil {
		return ToolResult{Content: "invalid list_dir arguments", IsError: true}, err
	}
	if strings.TrimSpace(args.Path) == "" {
		args.Path = "."
	}
	path, rel, err := e.resolve(args.Path)
	if err != nil {
		return ToolResult{Content: "path must be relative to a workspace root", IsError: true}, err
	}
	entries, err := os.ReadDir(path)
	if err != nil {
		return ToolResult{Content: fmt.Sprintf("cannot list %s", rel), IsError: true}, err
	}
	out := make([]string, 0, len(entries))
	for _, entry := range entries {
		name := filepath.ToSlash(filepath.Join(rel, entry.Name()))
		if entry.IsDir() {
			name += "/"
		}
		out = append(out, name)
	}
	sort.Strings(out)
	return ToolResult{Content: strings.Join(out, "\n")}, nil
}

func (e *WorkspaceExecutor) writeFile(raw json.RawMessage) (ToolResult, error) {
	var args struct {
		Path    string `json:"path"`
		Content string `json:"content"`
	}
	if err := json.Unmarshal(raw, &args); err != nil {
		return ToolResult{Content: "invalid write_file arguments", IsError: true}, err
	}
	path, rel, err := e.resolve(args.Path)
	if err != nil {
		return ToolResult{Content: "path must be relative to a workspace root", IsError: true}, err
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return ToolResult{Content: fmt.Sprintf("cannot create parent for %s", rel), IsError: true}, err
	}
	tmp, err := os.CreateTemp(filepath.Dir(path), ".chimera-write-*")
	if err != nil {
		return ToolResult{Content: fmt.Sprintf("cannot write %s", rel), IsError: true}, err
	}
	tmpName := tmp.Name()
	defer os.Remove(tmpName)
	if _, err := tmp.WriteString(args.Content); err != nil {
		_ = tmp.Close()
		return ToolResult{Content: fmt.Sprintf("cannot write %s", rel), IsError: true}, err
	}
	if err := tmp.Chmod(0o644); err != nil {
		_ = tmp.Close()
		return ToolResult{Content: fmt.Sprintf("cannot write %s", rel), IsError: true}, err
	}
	if err := tmp.Close(); err != nil {
		return ToolResult{Content: fmt.Sprintf("cannot write %s", rel), IsError: true}, err
	}
	if err := os.Rename(tmpName, path); err != nil {
		return ToolResult{Content: fmt.Sprintf("cannot write %s", rel), IsError: true}, err
	}
	return ToolResult{Content: fmt.Sprintf("wrote %s", rel)}, nil
}

func (e *WorkspaceExecutor) search(raw json.RawMessage) (ToolResult, error) {
	var args struct {
		Path  string `json:"path"`
		Query string `json:"query"`
	}
	if err := json.Unmarshal(raw, &args); err != nil {
		return ToolResult{Content: "invalid search arguments", IsError: true}, err
	}
	if strings.TrimSpace(args.Path) == "" {
		args.Path = "."
	}
	if strings.TrimSpace(args.Query) == "" {
		return ToolResult{Content: "search query is required", IsError: true}, errors.New("empty search query")
	}
	root, _, err := e.resolve(args.Path)
	if err != nil {
		return ToolResult{Content: "path must be relative to a workspace root", IsError: true}, err
	}
	var matches []string
	err = filepath.WalkDir(root, func(path string, d os.DirEntry, walkErr error) error {
		if walkErr != nil || d.IsDir() || len(matches) >= 100 {
			return nil
		}
		data, err := os.ReadFile(path)
		if err != nil {
			return nil
		}
		for i, line := range strings.Split(string(data), "\n") {
			if strings.Contains(line, args.Query) {
				for _, workspaceRoot := range e.Roots {
					if rel, err := filepath.Rel(workspaceRoot, path); err == nil && rel != ".." && !strings.HasPrefix(rel, ".."+string(filepath.Separator)) {
						matches = append(matches, fmt.Sprintf("%s:%d:%s", filepath.ToSlash(rel), i+1, line))
						break
					}
				}
				if len(matches) >= 100 {
					break
				}
			}
		}
		return nil
	})
	if err != nil {
		return ToolResult{Content: "search failed", IsError: true}, err
	}
	return ToolResult{Content: strings.Join(matches, "\n")}, nil
}

// OpenAITools returns fixed declarations, never client-controlled schemas.
func OpenAITools() []map[string]any {
	return []map[string]any{
		functionTool("read_file", "Read a file under the configured workspace.", map[string]any{"path": stringProperty("Workspace-relative file path")}, []string{"path"}),
		functionTool("write_file", "Atomically write a file under the configured workspace.", map[string]any{"path": stringProperty("Workspace-relative file path"), "content": stringProperty("Complete file contents")}, []string{"path", "content"}),
		functionTool("list_dir", "List a directory under the configured workspace.", map[string]any{"path": stringProperty("Workspace-relative directory path")}, nil),
		functionTool("search", "Search text under the configured workspace.", map[string]any{"path": stringProperty("Workspace-relative directory path"), "query": stringProperty("Text to search for")}, []string{"query"}),
	}
}

func functionTool(name, description string, properties map[string]any, required []string) map[string]any {
	params := map[string]any{"type": "object", "properties": properties, "additionalProperties": false}
	if len(required) > 0 {
		params["required"] = required
	}
	return map[string]any{"type": "function", "function": map[string]any{"name": name, "description": description, "parameters": params}}
}

func stringProperty(description string) map[string]any {
	return map[string]any{"type": "string", "description": description}
}
