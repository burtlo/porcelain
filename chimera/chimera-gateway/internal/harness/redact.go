package harness

import (
	"encoding/json"
)

// Redact returns a copy of env suitable for logs, SQLite, and response headers.
// Message bodies, tool arguments, and long text fields are stripped; counts and ids remain.
func Redact(env *TurnEnvelope) *TurnEnvelope {
	if env == nil {
		return nil
	}
	b, err := json.Marshal(env)
	if err != nil {
		return env
	}
	var cp TurnEnvelope
	if err := json.Unmarshal(b, &cp); err != nil {
		return env
	}
	cp.Execution.ToolCalls = redactToolCalls(cp.Execution.ToolCalls)
	cp.Evaluation = redactEvaluation(cp.Evaluation)
	cp.Response.Citations = redactCitations(cp.Response.Citations)
	cp.Response.Metadata = redactMetadata(cp.Response.Metadata)
	return &cp
}

func redactToolCalls(in []ToolCallRecord) []ToolCallRecord {
	if len(in) == 0 {
		return []ToolCallRecord{}
	}
	out := make([]ToolCallRecord, len(in))
	for i, tc := range in {
		out[i] = ToolCallRecord{Name: tc.Name, ID: tc.ID}
	}
	return out
}

func redactEvaluation(in Evaluation) Evaluation {
	out := in
	if len(in.Issues) > 0 {
		out.Issues = []string{}
	}
	return out
}

func redactCitations(in []Citation) []Citation {
	if len(in) == 0 {
		return []Citation{}
	}
	out := make([]Citation, len(in))
	for i, c := range in {
		out[i] = Citation{Source: c.Source, ID: c.ID}
	}
	return out
}

func redactMetadata(in map[string]any) map[string]any {
	if len(in) == 0 {
		return map[string]any{}
	}
	out := make(map[string]any, len(in))
	for k, v := range in {
		switch k {
		case "tools_before", "tools_after", "router_model", "tool_router_ran":
			out[k] = v
		default:
			// Drop unknown metadata that might hold prompt fragments.
		}
	}
	return out
}

// RedactedJSON marshals the redacted envelope for persistence.
func RedactedJSON(env *TurnEnvelope) ([]byte, error) {
	return json.Marshal(Redact(env))
}
