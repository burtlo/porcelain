package operatorstore

import (
	"encoding/json"
	"strings"
)

const (
	StreamPolicyImmediate           = "immediate"
	StreamPolicyGateOnEvaluator     = "gate_on_evaluator"
	StreamPolicyBufferUntilComplete = "buffer_until_complete"
)

// EvaluatorConfig controls the evaluator implementation for one assistant.
type EvaluatorConfig struct {
	ModelID              string  `json:"model_id"`
	Mode                 string  `json:"mode"`
	DraftCount           int     `json:"draft_count"`
	SynthesizeModelID    string  `json:"synthesize_model_id"`
	MinConfidence        float64 `json:"min_confidence"`
	HallucinationRiskMax float64 `json:"hallucination_risk_max"`
	StreamPolicy         string  `json:"stream_policy"`
}

// ParseEvaluatorConfig applies the safe new-VM defaults to persisted config.
func ParseEvaluatorConfig(raw string) EvaluatorConfig {
	cfg := EvaluatorConfig{Mode: "single_pass", DraftCount: 3, StreamPolicy: StreamPolicyImmediate}
	if json.Unmarshal([]byte(raw), &cfg) != nil {
		return cfg
	}
	cfg.ModelID = strings.TrimSpace(cfg.ModelID)
	cfg.Mode = strings.ToLower(strings.TrimSpace(cfg.Mode))
	if cfg.Mode != "single_pass" && cfg.Mode != "multi_draft" {
		cfg.Mode = "single_pass"
	}
	if cfg.DraftCount < 1 {
		cfg.DraftCount = 3
	}
	if cfg.DraftCount > 8 {
		cfg.DraftCount = 8
	}
	cfg.SynthesizeModelID = strings.TrimSpace(cfg.SynthesizeModelID)
	cfg.StreamPolicy = strings.ToLower(strings.TrimSpace(cfg.StreamPolicy))
	switch cfg.StreamPolicy {
	case StreamPolicyImmediate, StreamPolicyGateOnEvaluator, StreamPolicyBufferUntilComplete:
	default:
		cfg.StreamPolicy = StreamPolicyImmediate
	}
	if cfg.MinConfidence < 0 || cfg.MinConfidence > 1 {
		cfg.MinConfidence = 0
	}
	if cfg.HallucinationRiskMax < 0 || cfg.HallucinationRiskMax > 1 {
		cfg.HallucinationRiskMax = 0
	}
	return cfg
}

// EscalationConfig controls bounded evaluator remediation.
type EscalationConfig struct {
	MaxRounds          int            `json:"max_rounds"`
	OnFail             []string       `json:"on_fail"`
	HumanSurfaces      []HumanSurface `json:"human_surfaces"`
	PrivacyDisclosure  string         `json:"privacy_disclosure"`
	PasteBackDelimiter string         `json:"paste_back_delimiter"`
}

// HumanSurface is an operator-owned escalation destination presented to clients.
type HumanSurface struct {
	Name string `json:"name"`
	URL  string `json:"url"`
}

// ParseEscalationConfig accepts known actions and bounds the per-turn loop.
func ParseEscalationConfig(raw string) EscalationConfig {
	cfg := EscalationConfig{MaxRounds: 2, OnFail: []string{}, PasteBackDelimiter: "<<<CHIMERA_HUMAN_ANSWER>>>"}
	_ = json.Unmarshal([]byte(raw), &cfg)
	if cfg.MaxRounds < 0 {
		cfg.MaxRounds = 0
	}
	if cfg.MaxRounds > 2 {
		cfg.MaxRounds = 2
	}
	seen := map[string]bool{}
	out := make([]string, 0, len(cfg.OnFail))
	for _, action := range cfg.OnFail {
		action = strings.ToLower(strings.TrimSpace(action))
		switch action {
		case "re_retrieve", "fallback_chain", "ensemble", "human":
			if !seen[action] {
				seen[action] = true
				out = append(out, action)
			}
		}
	}
	cfg.OnFail = out
	cfg.PrivacyDisclosure = strings.TrimSpace(cfg.PrivacyDisclosure)
	cfg.PasteBackDelimiter = strings.TrimSpace(cfg.PasteBackDelimiter)
	if cfg.PasteBackDelimiter == "" {
		cfg.PasteBackDelimiter = "<<<CHIMERA_HUMAN_ANSWER>>>"
	}
	surfaces := make([]HumanSurface, 0, len(cfg.HumanSurfaces))
	for _, surface := range cfg.HumanSurfaces {
		surface.Name = strings.TrimSpace(surface.Name)
		surface.URL = strings.TrimSpace(surface.URL)
		if surface.Name != "" && surface.URL != "" {
			surfaces = append(surfaces, surface)
		}
	}
	cfg.HumanSurfaces = surfaces
	return cfg
}
