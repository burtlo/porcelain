// Package fake implements a minimal stdio MCP server for gateway transport tests.
package fake

import (
	"bufio"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"time"
)

// Config controls which tools the fake server advertises.
type Config struct {
	// CollisionTool adds a tool named read_file (native name collision test).
	CollisionTool bool
	// FailList makes tools/list return an MCP error (catalog soft-fail tests).
	FailList bool
	// EchoDelay blocks tools/call echo for the given duration.
	EchoDelay time.Duration
}

// Serve reads line-framed JSON-RPC from in and writes responses to out until EOF.
func Serve(in io.Reader, out io.Writer, cfg Config) error {
	br := bufio.NewReaderSize(in, 1<<20)
	for {
		line, err := br.ReadBytes('\n')
		if len(line) == 0 && err != nil {
			if errors.Is(err, io.EOF) {
				return nil
			}
			return err
		}
		var req struct {
			JSONRPC string          `json:"jsonrpc"`
			ID      json.RawMessage `json:"id"`
			Method  string          `json:"method"`
			Params  json.RawMessage `json:"params"`
		}
		if err := json.Unmarshal(line, &req); err != nil {
			return fmt.Errorf("fake mcp: decode: %w", err)
		}
		if req.Method == "notifications/initialized" {
			continue
		}
		if len(req.ID) == 0 {
			continue
		}
		resp, err := dispatch(req.Method, req.Params, req.ID, cfg)
		if err != nil {
			return err
		}
		if resp == nil {
			continue
		}
		raw, err := json.Marshal(resp)
		if err != nil {
			return err
		}
		raw = append(raw, '\n')
		if _, werr := out.Write(raw); werr != nil {
			return werr
		}
		if err != nil {
			if errors.Is(err, io.EOF) {
				return nil
			}
			return err
		}
	}
}

func dispatch(method string, params json.RawMessage, id json.RawMessage, cfg Config) (map[string]any, error) {
	switch method {
	case "initialize":
		return map[string]any{
			"jsonrpc": "2.0",
			"id":      json.RawMessage(id),
			"result": map[string]any{
				"protocolVersion": "2024-11-05",
				"capabilities":    map[string]any{"tools": map[string]any{}},
				"serverInfo":      map[string]string{"name": "chimera-mcp-fake", "version": "0.0.0"},
			},
		}, nil
	case "tools/list":
		if cfg.FailList {
			return rpcError(id, -32000, "list failed"), nil
		}
		tools := []map[string]any{
			{
				"name":        "echo",
				"description": "Echoes the message argument as text content.",
				"inputSchema": map[string]any{
					"type": "object",
					"properties": map[string]any{
						"message": map[string]any{"type": "string"},
					},
					"required": []string{"message"},
				},
			},
		}
		if cfg.CollisionTool {
			tools = append(tools, map[string]any{
				"name":        "read_file",
				"description": "Fake collision with native read_file.",
				"inputSchema": map[string]any{"type": "object"},
			})
		}
		return map[string]any{
			"jsonrpc": "2.0",
			"id":      json.RawMessage(id),
			"result":  map[string]any{"tools": tools},
		}, nil
	case "tools/call":
		var p struct {
			Name      string          `json:"name"`
			Arguments json.RawMessage `json:"arguments"`
		}
		if err := json.Unmarshal(params, &p); err != nil {
			return rpcError(id, -32602, "invalid params"), nil
		}
		switch p.Name {
		case "echo":
			var args struct {
				Message string `json:"message"`
			}
			_ = json.Unmarshal(p.Arguments, &args)
			if cfg.EchoDelay > 0 {
				time.Sleep(cfg.EchoDelay)
			}
			isErr := args.Message == "__force_error__"
			text := args.Message
			if isErr {
				text = "mcp reported error"
			}
			return map[string]any{
				"jsonrpc": "2.0",
				"id":      json.RawMessage(id),
				"result": map[string]any{
					"isError": isErr,
					"content": []map[string]any{
						{"type": "text", "text": text},
					},
				},
			}, nil
		case "read_file":
			return map[string]any{
				"jsonrpc": "2.0",
				"id":      json.RawMessage(id),
				"result": map[string]any{
					"content": []map[string]any{
						{"type": "text", "text": "mcp-read-file"},
					},
				},
			}, nil
		default:
			return rpcError(id, -32601, "tool not found"), nil
		}
	default:
		return rpcError(id, -32601, "method not found"), nil
	}
}

func rpcError(id json.RawMessage, code int, msg string) map[string]any {
	return map[string]any{
		"jsonrpc": "2.0",
		"id":      json.RawMessage(id),
		"error": map[string]any{
			"code":    code,
			"message": msg,
		},
	}
}
