package harness

import (
	"encoding/json"
	"strings"

	"github.com/lynn/porcelain/chimera/chimera-gateway/internal/harness/tools"
)

// inferToolCallsFromShorthand maps qwen-style argument blobs (without a "name"
// field) onto gateway-native workspace tools.
func inferToolCallsFromShorthand(block string) []tools.ToolCall {
	block = strings.TrimSpace(block)
	if block == "" {
		return nil
	}
	var fields map[string]json.RawMessage
	if json.Unmarshal([]byte(block), &fields) != nil {
		return nil
	}
	if _, ok := fields["name"]; ok {
		return nil
	}
	query := jsonStringField(fields, "query")
	path := jsonStringField(fields, "path")
	content := jsonStringField(fields, "content")
	if query != "" {
		args := map[string]string{"query": query}
		if path != "" {
			args["path"] = path
		}
		return []tools.ToolCall{toolCall("search", args)}
	}
	if path != "" && content != "" {
		return []tools.ToolCall{toolCall("write_file", map[string]string{"path": path, "content": content})}
	}
	if path != "" {
		return []tools.ToolCall{toolCall("read_file", map[string]string{"path": path})}
	}
	return nil
}

func toolCall(name string, args map[string]string) tools.ToolCall {
	raw, _ := json.Marshal(args)
	return tools.ToolCall{Name: name, Arguments: raw}
}

func jsonStringField(fields map[string]json.RawMessage, key string) string {
	raw, ok := fields[key]
	if !ok || len(raw) == 0 {
		return ""
	}
	var s string
	if json.Unmarshal(raw, &s) != nil {
		return ""
	}
	return strings.TrimSpace(s)
}
