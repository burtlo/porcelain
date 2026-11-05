package harness

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"strings"
	"time"

	"github.com/lynn/porcelain/chimera/chimera-gateway/internal/operatorstore"
	"github.com/lynn/porcelain/chimera/chimera-gateway/internal/rag"
)

// IntentConfig controls the per-assistant intent classifier. LLM mode is deliberately
// fail-open: unsupported or failed assistance retains the deterministic result.
type IntentConfig struct {
	Mode    string `json:"mode"`
	ModelID string `json:"model_id"`
}

// ParseIntentConfig decodes operator configuration with safe heuristic defaults.
func ParseIntentConfig(raw string) IntentConfig {
	cfg := IntentConfig{Mode: "heuristic"}
	if json.Unmarshal([]byte(raw), &cfg) != nil {
		return IntentConfig{Mode: "heuristic"}
	}
	cfg.Mode = strings.ToLower(strings.TrimSpace(cfg.Mode))
	if cfg.Mode != "llm" {
		cfg.Mode = "heuristic"
	}
	cfg.ModelID = strings.TrimSpace(cfg.ModelID)
	return cfg
}

// IntentStage derives deterministic intent signals before retrieval so retrieval
// skip_if rules can use intent tags. LLM assistance replaces those signals only
// when its constrained JSON response is valid; failures retain heuristics.
type IntentStage struct{}

func (IntentStage) Name() string   { return "intent" }
func (IntentStage) Module() string { return operatorstore.HarnessModuleIntent }

func (IntentStage) Run(ctx context.Context, tc *TurnContext, env *TurnEnvelope, body Body) error {
	if tc == nil || env == nil || tc.Stack.VM == nil {
		return nil
	}
	if !tc.Stack.VM.HarnessEnabled(operatorstore.HarnessModuleIntent) {
		return nil
	}
	text := strings.TrimSpace(rag.LastUserText(body["messages"]))
	if text == "" {
		return nil
	}
	intent := heuristicIntent(text, body, env)
	if module, ok := tc.Stack.VM.HarnessModules[operatorstore.HarnessModuleIntent]; ok {
		cfg := ParseIntentConfig(module.ConfigJSON)
		if cfg.Mode == "llm" && cfg.ModelID != "" {
			if classified, err := classifyIntentWithLLM(ctx, tc, cfg, text, intent); err == nil {
				intent = classified
			}
		}
	}
	env.Intent = intent
	if !containsStage(env.Plan.Stages, "intent") {
		env.Plan.Stages = append(env.Plan.Stages, "intent")
	}
	return nil
}

func classifyIntentWithLLM(ctx context.Context, tc *TurnContext, cfg IntentConfig, text string, fallback Intent) (Intent, error) {
	if tc == nil || tc.Resolved == nil || strings.TrimSpace(tc.Resolved.UpstreamBaseURL) == "" {
		return fallback, errIntentClassifierUnavailable
	}
	payload, err := json.Marshal(map[string]any{
		"model": cfg.ModelID,
		"messages": []map[string]string{
			{"role": "system", "content": "Classify the user request. Return only JSON with task_type, domain, complexity, ambiguity, sensitivity, requires_rag, requires_tools, tags."},
			{"role": "user", "content": text},
		},
		"stream": false,
	})
	if err != nil {
		return fallback, err
	}
	reqCtx, cancel := context.WithTimeout(ctx, classifierTimeout(tc.Timeout))
	defer cancel()
	req, err := http.NewRequestWithContext(reqCtx, http.MethodPost,
		strings.TrimRight(tc.Resolved.UpstreamBaseURL, "/")+"/v1/chat/completions", bytes.NewReader(payload))
	if err != nil {
		return fallback, err
	}
	req.Header.Set("Content-Type", "application/json")
	if tc.APIKey != "" {
		req.Header.Set("Authorization", "Bearer "+tc.APIKey)
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return fallback, err
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return fallback, errIntentClassifierUnavailable
	}
	var result struct {
		Choices []struct {
			Message struct {
				Content string `json:"content"`
			} `json:"message"`
		} `json:"choices"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil || len(result.Choices) == 0 {
		return fallback, errIntentClassifierUnavailable
	}
	var classified Intent
	if err := json.Unmarshal([]byte(result.Choices[0].Message.Content), &classified); err != nil {
		return fallback, err
	}
	return normalizeLLMIntent(classified, fallback), nil
}

var errIntentClassifierUnavailable = &intentClassifierError{}

type intentClassifierError struct{}

func (*intentClassifierError) Error() string { return "intent classifier unavailable" }

func classifierTimeout(timeout time.Duration) time.Duration {
	if timeout > 0 && timeout < 15*time.Second {
		return timeout
	}
	return 15 * time.Second
}

func normalizeLLMIntent(got, fallback Intent) Intent {
	if got.TaskType != "chat" && got.TaskType != "code" && got.TaskType != "unknown" {
		got.TaskType = fallback.TaskType
	}
	if got.Complexity != "low" && got.Complexity != "medium" && got.Complexity != "high" {
		got.Complexity = fallback.Complexity
	}
	if strings.TrimSpace(got.Domain) == "" {
		got.Domain = fallback.Domain
	}
	if strings.TrimSpace(got.Ambiguity) == "" {
		got.Ambiguity = fallback.Ambiguity
	}
	if strings.TrimSpace(got.Sensitivity) == "" {
		got.Sensitivity = fallback.Sensitivity
	}
	if got.RequiresRAG == nil {
		got.RequiresRAG = fallback.RequiresRAG
	}
	if got.RequiresTools == nil {
		got.RequiresTools = fallback.RequiresTools
	}
	if got.Tags == nil {
		got.Tags = fallback.Tags
	}
	return got
}

func heuristicIntent(text string, body Body, env *TurnEnvelope) Intent {
	length := len([]rune(text))
	toolCount := declaredToolCount(body["tools"])
	taskType, domain := "chat", "general"
	lower := strings.ToLower(text)
	if toolCount > 0 || strings.Contains(lower, "```") ||
		strings.Contains(lower, "code") || strings.Contains(lower, "function") ||
		strings.Contains(lower, "bug") || strings.Contains(lower, "test") {
		taskType, domain = "code", "code"
	}
	complexity := "low"
	if length > 1200 || toolCount >= 6 {
		complexity = "high"
	} else if length > 240 || toolCount > 0 {
		complexity = "medium"
	}
	requiresRAG := strings.TrimSpace(env.Scope.ProjectID) != "" || strings.TrimSpace(env.Scope.FlavorID) != ""
	requiresTools := toolCount > 0
	tags := []string{taskType, "complexity_" + complexity}
	if requiresRAG {
		tags = append(tags, "workspace")
	}
	if env.Scope.Sensitivity != "" && env.Scope.Sensitivity != "unknown" {
		tags = append(tags, "sensitivity_"+env.Scope.Sensitivity)
	}
	return Intent{
		TaskType: taskType, Domain: domain, Complexity: complexity, Ambiguity: "low",
		Sensitivity: env.Scope.Sensitivity, RequiresRAG: boolPtr(requiresRAG),
		RequiresTools: boolPtr(requiresTools), Tags: tags,
	}
}

func declaredToolCount(raw json.RawMessage) int {
	var tools []json.RawMessage
	if json.Unmarshal(raw, &tools) != nil {
		return 0
	}
	return len(tools)
}

func containsStage(stages []string, want string) bool {
	for _, stage := range stages {
		if stage == want {
			return true
		}
	}
	return false
}
