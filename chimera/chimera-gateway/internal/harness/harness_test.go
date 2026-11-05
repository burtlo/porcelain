package harness_test

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/lynn/porcelain/chimera/chimera-gateway/internal/assistant"
	"github.com/lynn/porcelain/chimera/chimera-gateway/internal/harness"
	"github.com/lynn/porcelain/internal/naming"
)

type recordingStage struct {
	name   string
	module string
	fn     func(*harness.TurnContext) error
}

func (s recordingStage) Name() string   { return s.name }
func (s recordingStage) Module() string { return s.module }
func (s recordingStage) Run(ctx context.Context, tc *harness.TurnContext, _ *harness.TurnEnvelope, _ harness.Body) error {
	if s.fn != nil {
		return s.fn(tc)
	}
	return nil
}

func TestRunner_executesStagesInOrderAndLogs(t *testing.T) {
	var order []string
	var buf bytes.Buffer
	log := slog.New(slog.NewJSONHandler(&buf, &slog.HandlerOptions{Level: slog.LevelDebug}))

	tc := &harness.TurnContext{
		RouteLog:       log,
		ConversationID: "conv-1",
		TurnIndex:      2,
		RequestID:      "req-1",
		Stack: harness.VMStack{
			VM: &assistant.Resolved{ModelID: "Test-1.0"},
		},
	}

	runner := harness.NewRunner(
		recordingStage{name: "first", module: "mod_a", fn: func(*harness.TurnContext) error {
			order = append(order, "first")
			return nil
		}},
		recordingStage{name: "second", module: "mod_b", fn: func(*harness.TurnContext) error {
			order = append(order, "second")
			return harness.ErrTurnComplete
		}},
	)

	body := harness.Body{"model": json.RawMessage(`"Test-1.0"`)}
	if err := runner.Run(context.Background(), tc, body); err != nil {
		t.Fatalf("Run: %v", err)
	}
	if len(order) != 2 || order[0] != "first" || order[1] != "second" {
		t.Fatalf("order=%v", order)
	}
	logs := buf.String()
	if !strings.Contains(logs, naming.MsgHarnessStageStarted) || !strings.Contains(logs, naming.MsgHarnessStageCompleted) {
		t.Fatalf("missing harness stage slugs in logs: %s", logs)
	}
	if !strings.Contains(logs, `"stage":"first"`) || !strings.Contains(logs, `"stage":"second"`) {
		t.Fatalf("missing stage names in logs: %s", logs)
	}
}

func TestRunner_abortStopsPipeline(t *testing.T) {
	var ranSecond bool
	runner := harness.NewRunner(
		recordingStage{name: "abort", module: "primary", fn: func(*harness.TurnContext) error {
			return &harness.AbortError{Status: http.StatusServiceUnavailable, Body: map[string]any{"error": "x"}}
		}},
		recordingStage{name: "second", module: "primary", fn: func(*harness.TurnContext) error {
			ranSecond = true
			return nil
		}},
	)
	tc := &harness.TurnContext{
		Stack: harness.VMStack{VM: &assistant.Resolved{ModelID: "T-1"}},
	}
	err := runner.Run(context.Background(), tc, harness.Body{})
	var abort *harness.AbortError
	if !errors.As(err, &abort) {
		t.Fatalf("want AbortError, got %v", err)
	}
	if ranSecond {
		t.Fatal("second stage should not run after abort")
	}
}

func TestHandleAbort_writesJSON(t *testing.T) {
	rec := httptest.NewRecorder()
	harness.HandleAbort(rec, &harness.AbortError{
		Status: http.StatusServiceUnavailable,
		Body: map[string]any{
			"error": map[string]any{"message": "nope", "type": "gateway_config"},
		},
	})
	if rec.Code != http.StatusServiceUnavailable {
		t.Fatalf("status=%d", rec.Code)
	}
	var body map[string]any
	if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
		t.Fatal(err)
	}
	errObj, _ := body["error"].(map[string]any)
	if errObj["type"] != "gateway_config" {
		t.Fatalf("body=%v", body)
	}
}

func TestReplaceBody_sameMapPreservesKeys(t *testing.T) {
	body := harness.Body{
		"model":    json.RawMessage(`"Chimera-1.0"`),
		"messages": json.RawMessage(`[{"role":"user","content":"hi"}]`),
	}
	// Same map returned from a noop transform must not wipe the body.
	harness_testReplaceBody(body, body)
	if _, ok := body["model"]; !ok {
		t.Fatal("model key missing after in-place merge")
	}
}

// harness_testReplaceBody mirrors replaceBody (unexported) for regression tests.
func harness_testReplaceBody(dst, src harness.Body) {
	for k := range dst {
		if _, ok := src[k]; !ok {
			delete(dst, k)
		}
	}
	for k, v := range src {
		dst[k] = v
	}
}

func TestStackResolveStage_setsAssistantID(t *testing.T) {
	tc := &harness.TurnContext{
		Stack: harness.VMStack{VM: &assistant.Resolved{ModelID: "X-1", FallbackChain: []string{"groq/x"}}},
	}
	env := &harness.TurnEnvelope{SchemaVersion: 1}
	if err := (harness.StackResolveStage{}).Run(context.Background(), tc, env, harness.Body{}); err != nil {
		t.Fatal(err)
	}
	if env.AssistantID != "X-1" {
		t.Fatalf("assistant_id=%q", env.AssistantID)
	}
}
