package harness

import "testing"

func TestCompletionTextSupportsJSONAndSSE(t *testing.T) {
	jsonBody := []byte(`{"choices":[{"message":{"content":"final answer"}}]}`)
	if got := completionText(jsonBody); got != "final answer" {
		t.Fatalf("JSON completionText = %q", got)
	}
	sseBody := []byte("data: {\"choices\":[{\"delta\":{\"content\":\"final \"}}]}\n\ndata: {\"choices\":[{\"delta\":{\"content\":\"answer\"}}]}\n\ndata: [DONE]\n")
	if got := completionText(sseBody); got != "final answer" {
		t.Fatalf("SSE completionText = %q", got)
	}
	reasoningJSON := []byte(`{"choices":[{"message":{"content":"","reasoning":"reasoned answer"}}]}`)
	if got := completionText(reasoningJSON); got != "reasoned answer" {
		t.Fatalf("reasoning JSON completionText = %q", got)
	}
}
