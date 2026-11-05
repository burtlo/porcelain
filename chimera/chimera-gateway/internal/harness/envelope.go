package harness

// TurnEnvelope is the per-turn artifact mutated by harness stages (schema_version: 1).
type TurnEnvelope struct {
	SchemaVersion  int        `json:"schema_version"`
	RequestID      string     `json:"request_id"`
	ConversationID string     `json:"conversation_id"`
	TurnIndex      int        `json:"turn_index"`
	AssistantID    string     `json:"assistant_id"`
	Scope          Scope      `json:"scope"`
	Intent         Intent     `json:"intent"`
	Plan           Plan       `json:"plan"`
	Retrieval      Retrieval  `json:"retrieval"`
	Execution      Execution  `json:"execution"`
	Evaluation     Evaluation `json:"evaluation"`
	Escalation     Escalation `json:"escalation"`
	Response       Response   `json:"response"`
}

// Scope holds tenant, RAG scope, and workspace policy fields (policy fields populated in later plans).
type Scope struct {
	TenantID            string `json:"tenant_id"`
	ProjectID           string `json:"project_id"`
	FlavorID            string `json:"flavor_id"`
	WorkspaceID         *int64 `json:"workspace_id"`
	WorkspacePermission string `json:"workspace_permission"`
	Sensitivity         string `json:"sensitivity"`
	AllowCloud          *bool  `json:"allow_cloud"`
}

// Intent holds classifier outputs (populated by intent plan).
type Intent struct {
	TaskType      string   `json:"task_type"`
	Domain        string   `json:"domain"`
	Complexity    string   `json:"complexity"`
	Ambiguity     string   `json:"ambiguity"`
	Sensitivity   string   `json:"sensitivity"`
	RequiresRAG   *bool    `json:"requires_rag"`
	RequiresTools *bool    `json:"requires_tools"`
	Tags          []string `json:"tags"`
}

// Plan holds harness stage plan metadata for the turn.
type Plan struct {
	Stages           []string `json:"stages"`
	PrimaryModelID   *string  `json:"primary_model_id"`
	EvaluatorMode    *string  `json:"evaluator_mode"`
	EscalationBudget int      `json:"escalation_budget"`
}

// Retrieval records vector retrieval for the turn.
type Retrieval struct {
	Ran              bool     `json:"ran"`
	TopK             *int     `json:"top_k"`
	HitsCount        int      `json:"hits_count"`
	EvidenceIDs      []string `json:"evidence_ids"`
	CompressStrategy *string  `json:"compress_strategy"`
}

// Execution records upstream and tool activity during the turn.
type Execution struct {
	ToolCalls        []ToolCallRecord  `json:"tool_calls"`
	UpstreamAttempts []UpstreamAttempt `json:"upstream_attempts"`
	ResolvedModelID  *string           `json:"resolved_model_id"`
}

// ToolCallRecord is a redacted tool invocation summary.
type ToolCallRecord struct {
	Name string `json:"name,omitempty"`
	ID   string `json:"id,omitempty"`
}

// UpstreamAttempt is one broker completion attempt during fallback routing.
type UpstreamAttempt struct {
	UpstreamModel string `json:"upstream_model"`
	Attempt       int    `json:"attempt"`
	Status        int    `json:"status,omitempty"`
}

// Evaluation holds evaluator module output (populated by evaluator plan).
type Evaluation struct {
	Ran                 bool     `json:"ran"`
	Confidence          *float64 `json:"confidence"`
	Issues              []string `json:"issues"`
	RecommendEscalation bool     `json:"recommend_escalation"`
}

// Escalation holds in-loop escalation state.
type Escalation struct {
	Rounds     int     `json:"rounds"`
	LastAction *string `json:"last_action"`
}

// Response holds citations and response metadata for the client turn.
type Response struct {
	Citations []Citation     `json:"citations"`
	Metadata  map[string]any `json:"metadata"`
}

// Citation is a source reference attached to the assistant reply.
type Citation struct {
	Source string `json:"source,omitempty"`
	ID     string `json:"id,omitempty"`
}

func newEnvelope(tc *TurnContext) *TurnEnvelope {
	env := &TurnEnvelope{
		SchemaVersion: 1,
		Intent:        defaultIntent(),
		Plan:          Plan{Stages: []string{}, EscalationBudget: 0},
		Retrieval:     Retrieval{EvidenceIDs: []string{}},
		Execution: Execution{
			ToolCalls:        []ToolCallRecord{},
			UpstreamAttempts: []UpstreamAttempt{},
		},
		Evaluation: Evaluation{Issues: []string{}},
		Escalation: Escalation{Rounds: 0},
		Response: Response{
			Citations: []Citation{},
			Metadata:  map[string]any{},
		},
	}
	if tc == nil {
		return env
	}
	env.RequestID = tc.RequestID
	env.ConversationID = tc.ConversationID
	env.TurnIndex = tc.TurnIndex
	env.AssistantID = tc.AssistantID()
	env.Scope = Scope{
		TenantID:            tc.TenantID,
		ProjectID:           tc.ProjectID,
		FlavorID:            tc.FlavorID,
		WorkspacePermission: "unknown",
		Sensitivity:         "unknown",
	}
	return env
}

func defaultIntent() Intent {
	return Intent{
		TaskType:    "unknown",
		Domain:      "unknown",
		Complexity:  "unknown",
		Ambiguity:   "unknown",
		Sensitivity: "unknown",
		Tags:        []string{},
	}
}
