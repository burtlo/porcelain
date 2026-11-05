package harness

import (
	"encoding/base64"
	"encoding/json"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/lynn/porcelain/chimera/chimera-gateway/internal/assistant"
	"github.com/lynn/porcelain/internal/naming"
)

func TestRedact_stripsSensitiveMetadata(t *testing.T) {
	env := &TurnEnvelope{
		SchemaVersion: 1,
		AssistantID:   "Test-1.0",
		Execution: Execution{
			ToolCalls: []ToolCallRecord{{Name: "read_file", ID: "tc1"}},
		},
		Evaluation: Evaluation{Issues: []string{"possible hallucination in user secret password123"}},
		Response: Response{
			Citations: []Citation{{Source: "doc.md", ID: "pt-1"}},
			Metadata: map[string]any{
				"tools_before": 5,
				"tools_after":  2,
				"raw_prompt":   "should not appear",
			},
		},
	}
	out := Redact(env)
	if len(out.Response.Metadata) != 2 {
		t.Fatalf("metadata=%v", out.Response.Metadata)
	}
	if _, ok := out.Response.Metadata["raw_prompt"]; ok {
		t.Fatal("raw_prompt should be stripped")
	}
	b, err := RedactedJSON(env)
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(b), "password123") || strings.Contains(string(b), "raw_prompt") {
		t.Fatalf("redacted json leaked secrets: %s", string(b))
	}
}

func TestWriteSummaryHeader_base64JSON(t *testing.T) {
	rec := httptest.NewRecorder()
	tc := &TurnContext{
		RequestID:      "req-1",
		ConversationID: "conv-1",
		TurnIndex:      1,
		Stack:          VMStack{VM: &assistant.Resolved{ModelID: "VM-1.0"}},
	}
	WriteSummaryHeader(rec, newEnvelope(tc))
	raw := rec.Header().Get(naming.HeaderHarnessSummaryTarget)
	if raw == "" {
		t.Fatal("missing header")
	}
	decoded, err := base64.StdEncoding.DecodeString(raw)
	if err != nil {
		t.Fatal(err)
	}
	var parsed map[string]any
	if err := json.Unmarshal(decoded, &parsed); err != nil {
		t.Fatal(err)
	}
	if parsed["assistant_id"] != "VM-1.0" {
		t.Fatalf("parsed=%v", parsed)
	}
}

func TestNewEnvelope_defaults(t *testing.T) {
	env := newEnvelope(nil)
	if env.SchemaVersion != 1 {
		t.Fatalf("schema_version=%d", env.SchemaVersion)
	}
	if env.Intent.TaskType != "unknown" {
		t.Fatalf("intent=%v", env.Intent)
	}
	if env.Plan.Stages == nil {
		t.Fatal("plan.stages nil")
	}
}
