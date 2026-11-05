package harness

import (
	"encoding/json"
	"strings"
)

// maybeReplaceStubFinalCompletion rewrites a tool-loop completion when the model
// acknowledges a tool result in JSON instead of answering the user.
func maybeReplaceStubFinalCompletion(body Body, userQuery string, raw []byte) ([]byte, string) {
	if len(raw) == 0 || !hadSuccessfulWorkspaceTool(body) {
		return raw, ""
	}
	content := completionText(raw)
	if !isLowValueFinalAfterTools(content) {
		return raw, ""
	}
	synth := synthesizeAnswerFromToolResults(body, userQuery)
	if strings.TrimSpace(synth) == "" {
		return raw, ""
	}
	patched, ok := patchCompletionContent(raw, synth)
	if !ok {
		return raw, ""
	}
	return patched, "tool_result"
}

func hadSuccessfulWorkspaceTool(body Body) bool {
	return strings.TrimSpace(lastSuccessfulToolContent(body, "")) != ""
}

func isLowValueFinalAfterTools(content string) bool {
	c := strings.TrimSpace(stripMarkdownJSONFence(content))
	if c == "" {
		return true
	}
	if len(inferToolCallsFromShorthand(c)) > 0 {
		return true
	}
	if looksLikeUnexecutedToolProse(content) {
		return true
	}
	if looksLikeStubToolJSON(c) {
		return true
	}
	if len(c) < 160 && !strings.Contains(c, "\n") {
		lower := strings.ToLower(c)
		if strings.Contains(lower, "success") || strings.Contains(lower, "read the file") {
			return true
		}
	}
	return false
}

func looksLikeUnexecutedToolProse(content string) bool {
	lower := strings.ToLower(content)
	if !strings.Contains(lower, "file written") &&
		!strings.Contains(lower, "function call") &&
		!strings.Contains(lower, "successfully written") {
		return false
	}
	for _, block := range extractMarkdownCodeBlocks(content) {
		if calls, _ := toolCallsFromBlock(block); len(calls) > 0 {
			return true
		}
	}
	return false
}

func looksLikeStubToolJSON(c string) bool {
	var m map[string]any
	if json.Unmarshal([]byte(c), &m) != nil {
		return false
	}
	resp, ok := m["response"].(string)
	if !ok {
		return false
	}
	lower := strings.ToLower(strings.TrimSpace(resp))
	return strings.Contains(lower, "success") ||
		strings.Contains(lower, "read the file") ||
		strings.Contains(lower, "file content")
}

func synthesizeAnswerFromToolResults(body Body, userQuery string) string {
	if msg, isErr := lastToolMessage(body, "write_file"); msg != "" {
		if isErr {
			return msg
		}
		if strings.HasPrefix(msg, "wrote ") {
			return "Created " + strings.TrimPrefix(msg, "wrote ") + " in the workspace."
		}
	}
	if searchOut, isErr := lastToolMessage(body, "search"); searchOut != "" {
		if isErr {
			return searchOut
		}
		if strings.TrimSpace(searchOut) == "" {
			return "No matches found in the workspace for that search."
		}
		return searchOut
	}
	if errMsg, isErr := lastToolMessage(body, "read_file"); isErr && errMsg != "" {
		return errMsg
	}
	fileContent := lastSuccessfulToolContent(body, "read_file")
	if fileContent == "" {
		fileContent = lastSuccessfulToolContent(body, "")
	}
	if strings.TrimSpace(fileContent) == "" {
		return ""
	}
	q := strings.ToLower(strings.TrimSpace(userQuery))
	if strings.Contains(q, "first paragraph") || strings.Contains(q, "first para") {
		return firstParagraph(fileContent)
	}
	if strings.Contains(q, "paragraph") {
		return firstParagraph(fileContent)
	}
	if strings.Contains(q, "readme") || strings.Contains(q, "read ") {
		return firstParagraph(fileContent)
	}
	if len(fileContent) > 4000 {
		return fileContent[:4000] + "…"
	}
	return fileContent
}

func lastToolMessage(body Body, preferName string) (string, bool) {
	raw, ok := body["messages"]
	if !ok {
		return "", false
	}
	var messages []json.RawMessage
	if json.Unmarshal(raw, &messages) != nil {
		return "", false
	}
	for i := len(messages) - 1; i >= 0; i-- {
		var m struct {
			Role    string `json:"role"`
			Name    string `json:"name"`
			Content string `json:"content"`
		}
		if json.Unmarshal(messages[i], &m) != nil || m.Role != "tool" {
			continue
		}
		if preferName != "" && strings.TrimSpace(m.Name) != preferName {
			continue
		}
		var payload struct {
			Content string `json:"content"`
			IsError bool   `json:"is_error"`
		}
		if json.Unmarshal([]byte(m.Content), &payload) != nil {
			continue
		}
		text := strings.TrimSpace(payload.Content)
		if text == "" {
			continue
		}
		return text, payload.IsError
	}
	return "", false
}

func lastSuccessfulToolContent(body Body, preferName string) string {
	raw, ok := body["messages"]
	if !ok {
		return ""
	}
	var messages []json.RawMessage
	if json.Unmarshal(raw, &messages) != nil {
		return ""
	}
	fallback := ""
	for i := len(messages) - 1; i >= 0; i-- {
		text, isErr := toolMessageAt(messages[i])
		if text == "" || isErr {
			continue
		}
		var m struct {
			Role string `json:"role"`
			Name string `json:"name"`
		}
		if json.Unmarshal(messages[i], &m) != nil || m.Role != "tool" {
			continue
		}
		if preferName != "" && strings.TrimSpace(m.Name) == preferName {
			return text
		}
		if fallback == "" {
			fallback = text
		}
	}
	return fallback
}

func toolMessageAt(raw json.RawMessage) (string, bool) {
	var m struct {
		Role    string `json:"role"`
		Content string `json:"content"`
	}
	if json.Unmarshal(raw, &m) != nil || m.Role != "tool" {
		return "", false
	}
	var payload struct {
		Content string `json:"content"`
		IsError bool   `json:"is_error"`
	}
	if json.Unmarshal([]byte(m.Content), &payload) != nil {
		return "", false
	}
	return strings.TrimSpace(payload.Content), payload.IsError
}

func firstParagraph(text string) string {
	text = strings.TrimSpace(text)
	if text == "" {
		return ""
	}
	parts := strings.Split(text, "\n\n")
	for _, p := range parts {
		p = strings.TrimSpace(p)
		if p != "" {
			return p
		}
	}
	return text
}

func patchCompletionContent(raw []byte, newContent string) ([]byte, bool) {
	var doc map[string]any
	if json.Unmarshal(raw, &doc) != nil {
		return raw, false
	}
	choices, ok := doc["choices"].([]any)
	if !ok || len(choices) == 0 {
		return raw, false
	}
	choice, ok := choices[0].(map[string]any)
	if !ok {
		return raw, false
	}
	msg, ok := choice["message"].(map[string]any)
	if !ok {
		return raw, false
	}
	msg["content"] = newContent
	delete(msg, "tool_calls")
	out, err := json.Marshal(doc)
	if err != nil {
		return raw, false
	}
	return out, true
}
