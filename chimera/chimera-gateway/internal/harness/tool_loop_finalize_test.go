package harness

import (
	"encoding/json"
	"testing"
)

func TestMaybeReplaceStubFinalCompletion_readmeParagraph(t *testing.T) {
	readme := "# Chimera Gateway Runtime\n\n**Porcelain** is the local stack.\n\nMore text."
	toolPayload, _ := json.Marshal(map[string]any{"content": readme, "is_error": false})
	toolMsg, _ := json.Marshal(map[string]any{
		"role": "tool", "name": "read_file", "tool_call_id": "c1",
		"content": string(toolPayload),
	})
	assistant, _ := json.Marshal(map[string]any{
		"role": "assistant", "content": "",
		"tool_calls": []map[string]any{{"id": "c1", "type": "function", "function": map[string]string{"name": "read_file", "arguments": "{}"}}},
	})
	user, _ := json.Marshal(map[string]any{"role": "user", "content": "Read the first paragraph of the README.md"})
	messages, _ := json.Marshal([]json.RawMessage{user, assistant, toolMsg})
	body := Body{"messages": messages}

	stub, _ := json.Marshal(map[string]any{
		"choices": []map[string]any{{
			"message": map[string]string{
				"role":    "assistant",
				"content": "```json\n{\"response\": \"Read the file content successfully.\"}\n```",
			},
		}},
	})
	patched, reason := maybeReplaceStubFinalCompletion(body, "Read the first paragraph of the README.md", stub)
	if reason != "tool_result" {
		t.Fatalf("reason=%q", reason)
	}
	got := completionText(patched)
	if got != "# Chimera Gateway Runtime" {
		t.Fatalf("content=%q", got)
	}
}

func TestMaybeReplaceStubFinalCompletion_keepsGoodAnswer(t *testing.T) {
	toolMsg, _ := json.Marshal(map[string]any{
		"role": "tool", "name": "read_file",
		"content": `{"content":"line1\n\nline2","is_error":false}`,
	})
	messages, _ := json.Marshal([]json.RawMessage{toolMsg})
	body := Body{"messages": messages}
	good, _ := json.Marshal(map[string]any{
		"choices": []map[string]any{{"message": map[string]string{"content": "The README opens with: line1"}}},
	})
	patched, reason := maybeReplaceStubFinalCompletion(body, "summarize readme", good)
	if reason != "" {
		t.Fatalf("reason=%q", reason)
	}
	if string(patched) != string(good) {
		t.Fatalf("should not patch good answers")
	}
}
