package harness

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"time"
)

// writeBufferedResponse copies a recorder response to the client. When the client
// requested streaming, JSON completions are re-encoded as SSE so stream clients
// (eval runners, chat UI) receive parseable events.
func writeBufferedResponse(w http.ResponseWriter, recorder *httptest.ResponseRecorder, clientStream bool) {
	if w == nil || recorder == nil {
		return
	}
	raw := recorder.Body.Bytes()
	if clientStream && recorder.Code >= 200 && recorder.Code < 300 && json.Valid(raw) {
		copyResponseHeaders(w, recorder.Result().Header)
		w.Header().Set("Content-Type", "text/event-stream; charset=utf-8")
		w.Header().Set("Cache-Control", "no-cache")
		w.Header().Set("Connection", "keep-alive")
		w.WriteHeader(recorder.Code)
		_, _ = w.Write(jsonCompletionToSSE(raw))
		if f, ok := w.(http.Flusher); ok {
			f.Flush()
		}
		return
	}
	copyResponseHeaders(w, recorder.Result().Header)
	w.WriteHeader(recorder.Code)
	_, _ = w.Write(raw)
}

func copyResponseHeaders(w http.ResponseWriter, hdr http.Header) {
	for key, values := range hdr {
		w.Header().Del(key)
		for _, value := range values {
			w.Header().Add(key, value)
		}
	}
}

func jsonCompletionToSSE(jsonBody []byte) []byte {
	var comp struct {
		ID      string `json:"id"`
		Created int64  `json:"created"`
		Model   string `json:"model"`
		Choices []struct {
			Index        int    `json:"index"`
			FinishReason string `json:"finish_reason"`
			Message      struct {
				Role      string `json:"role"`
				Content   string `json:"content"`
				Reasoning string `json:"reasoning"`
			} `json:"message"`
		} `json:"choices"`
	}
	if json.Unmarshal(jsonBody, &comp) != nil || len(comp.Choices) == 0 {
		return []byte("data: [DONE]\n\n")
	}
	choice := comp.Choices[0]
	content := strings.TrimSpace(choice.Message.Content)
	if content == "" {
		content = strings.TrimSpace(choice.Message.Reasoning)
	}
	id := comp.ID
	if id == "" {
		id = "chatcmpl-harness"
	}
	model := comp.Model
	if model == "" {
		model = "harness"
	}
	created := comp.Created
	if created == 0 {
		created = time.Now().Unix()
	}
	index := choice.Index
	var buf bytes.Buffer
	writeSSEDelta(&buf, id, model, created, index, map[string]string{"role": "assistant"})
	if content != "" {
		writeSSEDelta(&buf, id, model, created, index, map[string]string{"content": content})
	}
	finish := strings.TrimSpace(choice.FinishReason)
	if finish == "" {
		finish = "stop"
	}
	writeSSEFinish(&buf, id, model, created, index, finish)
	buf.WriteString("data: [DONE]\n\n")
	return buf.Bytes()
}

func writeSSEDelta(buf *bytes.Buffer, id, model string, created int64, index int, delta map[string]string) {
	payload := map[string]any{
		"id":      id,
		"object":  "chat.completion.chunk",
		"created": created,
		"model":   model,
		"choices": []map[string]any{{
			"index": index,
			"delta": delta,
		}},
	}
	b, _ := json.Marshal(payload)
	fmt.Fprintf(buf, "data: %s\n\n", b)
}

func writeSSEFinish(buf *bytes.Buffer, id, model string, created int64, index int, finishReason string) {
	payload := map[string]any{
		"id":      id,
		"object":  "chat.completion.chunk",
		"created": created,
		"model":   model,
		"choices": []map[string]any{{
			"index":         index,
			"delta":         map[string]string{},
			"finish_reason": finishReason,
		}},
	}
	b, _ := json.Marshal(payload)
	fmt.Fprintf(buf, "data: %s\n\n", b)
}
