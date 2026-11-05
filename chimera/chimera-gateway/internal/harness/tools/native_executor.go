package tools

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/lynn/porcelain/chimera/chimera-gateway/internal/rag"
	"github.com/lynn/porcelain/chimera/chimera-gateway/internal/vectorstore"
)

// NativeExecutor dispatches reserved native tools to workspace file actions and RAG expansion.
type NativeExecutor struct {
	Workspace *WorkspaceExecutor
	Expansion *rag.ExpansionService
	Coords    vectorstore.Coords
}

// Invoke implements ToolExecutor.
func (n *NativeExecutor) Invoke(ctx context.Context, call ToolCall) (ToolResult, error) {
	if n == nil {
		return ToolResult{Content: "native tool executor unavailable", IsError: true}, fmt.Errorf("nil native executor")
	}
	switch call.Name {
	case "workspace_context_around", "workspace_adjacent_chunks", "workspace_read_lines":
		return n.invokeExpansion(ctx, call)
	default:
		if n.Workspace == nil {
			return ToolResult{Content: "workspace tools unavailable", IsError: true}, fmt.Errorf("nil workspace executor")
		}
		return n.Workspace.Invoke(ctx, call)
	}
}

func (n *NativeExecutor) invokeExpansion(ctx context.Context, call ToolCall) (ToolResult, error) {
	if err := ctx.Err(); err != nil {
		return ToolResult{}, err
	}
	switch call.Name {
	case "workspace_context_around":
		return n.contextAround(ctx, call.Arguments)
	case "workspace_read_lines":
		return n.readLines(ctx, call.Arguments)
	case "workspace_adjacent_chunks":
		return n.adjacentChunks(ctx, call.Arguments)
	default:
		return ToolResult{Content: "unknown expansion tool", IsError: true}, fmt.Errorf("unknown tool %q", call.Name)
	}
}

func (n *NativeExecutor) contextAround(ctx context.Context, raw json.RawMessage) (ToolResult, error) {
	var args struct {
		Source        string `json:"source"`
		Line          int    `json:"line"`
		BeforeLines   int    `json:"before_lines"`
		AfterLines    int    `json:"after_lines"`
		ContentSHA256 string `json:"content_sha256"`
	}
	if err := json.Unmarshal(raw, &args); err != nil {
		return ToolResult{Content: "invalid workspace_context_around arguments", IsError: true}, err
	}
	source, err := n.validateSource(args.Source)
	if err != nil {
		return ToolResult{Content: "source must be relative to a workspace root", IsError: true}, err
	}
	hash := strings.TrimSpace(args.ContentSHA256)
	if hash == "" {
		start := args.Line - args.BeforeLines
		if start < 1 {
			start = 1
		}
		end := args.Line + args.AfterLines
		return n.readLinesFromWorkspace(ctx, source, start, end)
	}
	if args.Line < 1 {
		return ToolResult{Content: "line must be >= 1", IsError: true}, fmt.Errorf("invalid line")
	}
	if n.Expansion == nil {
		return ToolResult{Content: "rag expansion tools unavailable", IsError: true}, fmt.Errorf("expansion service nil")
	}
	text, err := n.Expansion.ContextAround(ctx, n.Coords, source, hash, args.Line, args.BeforeLines, args.AfterLines)
	if err != nil {
		return ToolResult{Content: err.Error(), IsError: true}, err
	}
	return ToolResult{Content: text}, nil
}

func (n *NativeExecutor) readLines(ctx context.Context, raw json.RawMessage) (ToolResult, error) {
	var args struct {
		Source        string `json:"source"`
		StartLine     int    `json:"start_line"`
		EndLine       int    `json:"end_line"`
		ContentSHA256 string `json:"content_sha256"`
	}
	if err := json.Unmarshal(raw, &args); err != nil {
		return ToolResult{Content: "invalid workspace_read_lines arguments", IsError: true}, err
	}
	source, err := n.validateSource(args.Source)
	if err != nil {
		return ToolResult{Content: "source must be relative to a workspace root", IsError: true}, err
	}
	hash := strings.TrimSpace(args.ContentSHA256)
	if args.StartLine < 1 || args.EndLine < args.StartLine {
		return ToolResult{Content: "start_line and end_line required", IsError: true}, fmt.Errorf("invalid line range")
	}
	if hash == "" {
		return n.readLinesFromWorkspace(ctx, source, args.StartLine, args.EndLine)
	}
	if n.Expansion == nil {
		return ToolResult{Content: "rag expansion tools unavailable", IsError: true}, fmt.Errorf("expansion service nil")
	}
	before := 0
	after := args.EndLine - args.StartLine
	text, err := n.Expansion.ContextAround(ctx, n.Coords, source, hash, args.StartLine, before, after)
	if err != nil {
		return ToolResult{Content: err.Error(), IsError: true}, err
	}
	return ToolResult{Content: text}, nil
}

func (n *NativeExecutor) adjacentChunks(ctx context.Context, raw json.RawMessage) (ToolResult, error) {
	if n.Expansion == nil {
		return ToolResult{Content: "rag expansion tools unavailable", IsError: true}, fmt.Errorf("expansion service nil")
	}
	var args struct {
		PointID string `json:"point_id"`
		Radius  int    `json:"radius"`
	}
	if err := json.Unmarshal(raw, &args); err != nil {
		return ToolResult{Content: "invalid workspace_adjacent_chunks arguments", IsError: true}, err
	}
	pointID := strings.TrimSpace(args.PointID)
	if pointID == "" {
		return ToolResult{Content: "point_id is required", IsError: true}, fmt.Errorf("point_id required")
	}
	radius := args.Radius
	if radius <= 0 {
		radius = 1
	}
	hits, err := n.Expansion.AdjacentChunks(ctx, n.Coords, pointID, radius)
	if err != nil {
		return ToolResult{Content: err.Error(), IsError: true}, err
	}
	out, err := json.Marshal(map[string]any{"hits": hits})
	if err != nil {
		return ToolResult{Content: "failed to encode adjacent chunks", IsError: true}, err
	}
	return ToolResult{Content: string(out)}, nil
}

func (n *NativeExecutor) readLinesFromWorkspace(ctx context.Context, source string, startLine, endLine int) (ToolResult, error) {
	if n.Workspace == nil {
		return ToolResult{Content: "workspace tools unavailable", IsError: true}, fmt.Errorf("nil workspace executor")
	}
	pathJSON, err := json.Marshal(pathArgs{Path: source})
	if err != nil {
		return ToolResult{Content: "invalid path", IsError: true}, err
	}
	full, err := n.Workspace.Invoke(ctx, ToolCall{Name: "read_file", Arguments: pathJSON})
	if err != nil {
		return full, err
	}
	if full.IsError {
		return full, nil
	}
	lines := strings.Split(full.Content, "\n")
	if startLine > len(lines) {
		return ToolResult{Content: "", IsError: false}, nil
	}
	if endLine > len(lines) {
		endLine = len(lines)
	}
	return ToolResult{Content: strings.Join(lines[startLine-1:endLine], "\n")}, nil
}

func (n *NativeExecutor) validateSource(source string) (string, error) {
	if n.Workspace == nil {
		return "", ErrPath
	}
	_, rel, err := n.Workspace.resolve(source)
	if err != nil {
		return "", err
	}
	return rel, nil
}

// OpenAIToolDeclarations returns gateway-owned tool schemas for the harness loop.
func OpenAIToolDeclarations(includeExpansion bool) []map[string]any {
	out := append([]map[string]any(nil), OpenAITools()...)
	if includeExpansion {
		out = append(out, rag.WorkspaceOpenAIFunctionTools()...)
	}
	return out
}
