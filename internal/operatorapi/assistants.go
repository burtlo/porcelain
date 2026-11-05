package operatorapi

import "encoding/json"

// AssistantSummary is a list entry for GET /api/ui/state and assistants list.
type AssistantSummary struct {
	ID                   int64    `json:"id"`
	ModelID              string   `json:"model_id"`
	Name                 string   `json:"name"`
	Version              string   `json:"version"`
	Description          string   `json:"description,omitempty"`
	Enabled              bool     `json:"enabled"`
	Visibility           string   `json:"visibility"`
	FallbackDepth        int      `json:"fallback_depth"`
	RoutingPolicyEnabled bool     `json:"routing_policy_enabled"`
	ToolRouterEnabled    bool     `json:"tool_router_enabled"`
	RouterModels         []string `json:"router_models,omitempty"`
}

// AssistantDetail is GET /api/ui/assistants/{id}.
type AssistantDetail struct {
	AssistantSummary
	RoutingPolicyYAML    string                   `json:"routing_policy_yaml,omitempty"`
	FallbackChain        []string                 `json:"fallback_chain"`
	FallbackUnavailable  []string                 `json:"fallback_unavailable,omitempty"`
	ToolRouterConfidence float64                  `json:"tool_router_confidence_threshold"`
	HarnessModules       []AssistantHarnessModule `json:"harness_modules,omitempty"`
	CreatedByPrincipalID string                   `json:"created_by_principal_id,omitempty"`
	CreatedAt            string                   `json:"created_at"`
	UpdatedAt            string                   `json:"updated_at"`
}

// AssistantCreateRequest is POST /api/ui/assistants body.
type AssistantCreateRequest struct {
	Name        string `json:"name"`
	Version     string `json:"version"`
	Description string `json:"description,omitempty"`
	Visibility  string `json:"visibility,omitempty"`
	ModelID     string `json:"model_id,omitempty"`
}

// AssistantUpdateRequest is PUT /api/ui/assistants/{id} body.
type AssistantUpdateRequest struct {
	Name        *string `json:"name,omitempty"`
	Version     *string `json:"version,omitempty"`
	Description *string `json:"description,omitempty"`
	Enabled     *bool   `json:"enabled,omitempty"`
	Visibility  *string `json:"visibility,omitempty"`
}

// AssistantFallbackSaveRequest is PUT /api/ui/assistants/{id}/fallback.
type AssistantFallbackSaveRequest struct {
	FallbackChain []string `json:"fallback_chain"`
}

// AssistantRoutingPolicySaveRequest is PUT /api/ui/assistants/{id}/routing-policy.
type AssistantRoutingPolicySaveRequest struct {
	Enabled           bool   `json:"enabled"`
	RoutingPolicyYAML string `json:"routing_policy_yaml"`
}

// AssistantToolRouterSaveRequest is PUT /api/ui/assistants/{id}/tool-router.
type AssistantToolRouterSaveRequest struct {
	Enabled             bool     `json:"tool_router_enabled"`
	RouterModels        []string `json:"router_models"`
	ConfidenceThreshold float64  `json:"confidence_threshold"`
}

// AssistantHarnessModule is one harness module on a assistant.
type AssistantHarnessModule struct {
	ModuleID   string          `json:"module_id"`
	Enabled    bool            `json:"enabled"`
	ConfigJSON json.RawMessage `json:"config_json,omitempty"`
	// UI hints (read-only on GET)
	Configurable   bool   `json:"configurable,omitempty"`
	DisabledReason string `json:"disabled_reason,omitempty"`
}

// AssistantHarnessResponse is GET /api/ui/assistants/{id}/harness.
type AssistantHarnessResponse struct {
	Modules []AssistantHarnessModule `json:"modules"`
}

// AssistantHarnessSaveRequest is PUT /api/ui/assistants/{id}/harness.
type AssistantHarnessSaveRequest struct {
	Modules []AssistantHarnessModule `json:"modules"`
}

// AssistantHarnessEvaluateRequest is a dry-run pre-primary harness input.
type AssistantHarnessEvaluateRequest struct {
	Message string `json:"message"`
	Project string `json:"project,omitempty"`
	Flavor  string `json:"flavor,omitempty"`
}

// AssistantHarnessEvaluateResponse is a redacted dry-run envelope.
type AssistantHarnessEvaluateResponse struct {
	OK       bool            `json:"ok"`
	Envelope json.RawMessage `json:"envelope"`
}

// AssistantListResponse is GET /api/ui/assistants.
type AssistantListResponse struct {
	Assistants []AssistantSummary `json:"assistants"`
}

// AssistantGenerateRequest is POST /api/ui/assistants/{id}/routing/generate.
type AssistantGenerateRequest struct {
	ProviderPrefix  string   `json:"provider_prefix,omitempty"`
	PreferredModels []string `json:"preferred_models,omitempty"`
	LocalFirst      *bool    `json:"local_first,omitempty"`
	Save            bool     `json:"save"`
}

// RoutingRuleDefinitionSummary is one catalog entry.
type RoutingRuleDefinitionSummary struct {
	ID          int64  `json:"id"`
	Name        string `json:"name"`
	Slug        string `json:"slug"`
	Description string `json:"description,omitempty"`
}
