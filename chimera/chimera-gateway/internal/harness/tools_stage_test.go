package harness

import (
	"encoding/json"
	"strings"
	"testing"
)

func TestCompletionToolCalls_openAIFormat(t *testing.T) {
	raw := []byte(`{"choices":[{"message":{"role":"assistant","content":"","tool_calls":[{"id":"c1","type":"function","function":{"name":"read_file","arguments":"{\"path\":\"a.md\"}"}}]}}]}`)
	calls, _ := completionToolCalls(raw)
	if len(calls) != 1 || calls[0].Name != "read_file" {
		t.Fatalf("calls=%+v", calls)
	}
}

func TestCompletionToolCalls_contentJSON(t *testing.T) {
	raw := []byte(`{"choices":[{"message":{"role":"assistant","content":"{\n  \"name\": \"read_file\",\n  \"arguments\": {\"path\": \"docs/x.md\"}\n}"}}]}`)
	calls, assistant := completionToolCalls(raw)
	if len(calls) != 1 || calls[0].Name != "read_file" {
		t.Fatalf("calls=%+v", calls)
	}
	if !strings.Contains(string(assistant), `"tool_calls"`) {
		t.Fatalf("assistant=%s", assistant)
	}
}

func TestCompletionToolCalls_markdownFence(t *testing.T) {
	content := "```json\n{\"name\": \"read_file\", \"arguments\": {\"path\": \"docs/y.md\"}}\n```"
	raw, _ := json.Marshal(map[string]any{
		"choices": []map[string]any{{"message": map[string]string{"role": "assistant", "content": content}}},
	})
	calls, _ := completionToolCalls(raw)
	if len(calls) != 1 {
		t.Fatalf("calls=%+v", calls)
	}
}

func TestCompletionToolCalls_proseMarkdownFence(t *testing.T) {
	content := "Certainly! Here is the call:\n\n```json\n{\"name\": \"write_file\", \"arguments\": {\"path\": \"LynnLovesIntros.md\", \"content\": \"hello world\\n\"}}\n```\n\n**File Written:** (fake)"
	raw, _ := json.Marshal(map[string]any{
		"choices": []map[string]any{{"message": map[string]string{"role": "assistant", "content": content}}},
	})
	calls, _ := completionToolCalls(raw)
	if len(calls) != 1 || calls[0].Name != "write_file" {
		t.Fatalf("calls=%+v", calls)
	}
}

func TestCompletionToolCalls_contentShorthand(t *testing.T) {
	cases := []struct {
		content string
		name    string
	}{
		{`{"query": "plan files"}`, "search"},
		{`{"path": "LynnLovesJUNK.md", "content": "hi"}`, "write_file"},
		{`{"path": "README.md"}`, "read_file"},
	}
	for _, tc := range cases {
		raw, _ := json.Marshal(map[string]any{
			"choices": []map[string]any{{"message": map[string]string{"role": "assistant", "content": tc.content}}},
		})
		calls, _ := completionToolCalls(raw)
		if len(calls) != 1 || calls[0].Name != tc.name {
			t.Fatalf("content=%q calls=%+v want %s", tc.content, calls, tc.name)
		}
	}
}

func TestCompletionToolCalls_toolCallTag(t *testing.T) {
	content := `<tool_call>
{"name": "read_file", "arguments": {"path": "docs/z.md"}}
</tool_call>`
	raw, _ := json.Marshal(map[string]any{
		"choices": []map[string]any{{"message": map[string]string{"role": "assistant", "content": content}}},
	})
	calls, _ := completionToolCalls(raw)
	if len(calls) != 1 {
		t.Fatalf("calls=%+v", calls)
	}
}
