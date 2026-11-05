package harness

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestWriteBufferedResponse_streamEncodesJSONAsSSE(t *testing.T) {
	rec := httptest.NewRecorder()
	rec.Header().Set("Content-Type", "application/json")
	rec.WriteHeader(http.StatusOK)
	_, _ = rec.Write([]byte(`{"id":"cmpl-1","model":"qwen3:8b","choices":[{"index":0,"message":{"role":"assistant","content":"hello"},"finish_reason":"stop"}]}`))

	out := httptest.NewRecorder()
	writeBufferedResponse(out, rec, true)

	body := out.Body.String()
	if !strings.Contains(out.Header().Get("Content-Type"), "text/event-stream") {
		t.Fatalf("content-type=%q", out.Header().Get("Content-Type"))
	}
	if !strings.Contains(body, "data:") || !strings.Contains(body, "[DONE]") {
		t.Fatalf("expected SSE body, got %q", body)
	}
	if !strings.Contains(body, `"content":"hello"`) {
		t.Fatalf("missing content delta: %q", body)
	}
}

func TestWriteBufferedResponse_nonStreamPassthroughJSON(t *testing.T) {
	raw := `{"choices":[{"message":{"content":"x"}}]}`
	rec := httptest.NewRecorder()
	rec.WriteHeader(http.StatusOK)
	_, _ = rec.Write([]byte(raw))

	out := httptest.NewRecorder()
	writeBufferedResponse(out, rec, false)

	if out.Body.String() != raw {
		t.Fatalf("got %q", out.Body.String())
	}
}
