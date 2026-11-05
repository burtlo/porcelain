package harness

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/lynn/porcelain/chimera/chimera-gateway/internal/operatorstore"
	"github.com/lynn/porcelain/internal/naming"
)

type evaluatorResult struct {
	Confidence          float64  `json:"confidence"`
	Issues              []string `json:"issues"`
	RecommendEscalation bool     `json:"recommend_escalation"`
}

func evaluatorConfig(tc *TurnContext) (operatorstore.EvaluatorConfig, bool) {
	if tc == nil || tc.Stack.VM == nil || !tc.Stack.VM.HarnessEnabled(operatorstore.HarnessModuleEvaluator) {
		return operatorstore.EvaluatorConfig{}, false
	}
	module, ok := tc.Stack.VM.HarnessModules[operatorstore.HarnessModuleEvaluator]
	if !ok {
		return operatorstore.EvaluatorConfig{}, false
	}
	cfg := operatorstore.ParseEvaluatorConfig(module.ConfigJSON)
	return cfg, cfg.ModelID != ""
}

func escalationConfig(tc *TurnContext) (operatorstore.EscalationConfig, bool) {
	if tc == nil || tc.Stack.VM == nil || !tc.Stack.VM.HarnessEnabled(operatorstore.HarnessModuleEscalation) {
		return operatorstore.EscalationConfig{}, false
	}
	module, ok := tc.Stack.VM.HarnessModules[operatorstore.HarnessModuleEscalation]
	if !ok {
		return operatorstore.EscalationConfig{}, false
	}
	return operatorstore.ParseEscalationConfig(module.ConfigJSON), true
}

func runEvaluator(ctx context.Context, tc *TurnContext, env *TurnEnvelope, cfg operatorstore.EvaluatorConfig, primary string) error {
	if tc == nil || tc.Resolved == nil || strings.TrimSpace(primary) == "" {
		return errors.New("evaluator unavailable")
	}
	content, err := brokerCompletion(ctx, tc, cfg.ModelID, []map[string]string{
		{"role": "system", "content": "Evaluate the proposed assistant answer. Return only JSON: {\"confidence\":0..1,\"issues\":[],\"recommend_escalation\":boolean}."},
		{"role": "user", "content": primary},
	})
	if err != nil {
		return err
	}
	var result evaluatorResult
	if err := json.Unmarshal([]byte(content), &result); err != nil {
		return err
	}
	if result.Confidence < 0 || result.Confidence > 1 {
		return errors.New("evaluator confidence outside range")
	}
	if env != nil {
		env.Evaluation.Ran = true
		env.Evaluation.Confidence = &result.Confidence
		env.Evaluation.Issues = result.Issues
		env.Evaluation.RecommendEscalation = result.RecommendEscalation || (cfg.MinConfidence > 0 && result.Confidence < cfg.MinConfidence)
		mode := cfg.Mode
		env.Plan.EvaluatorMode = &mode
	}
	return nil
}

// brokerCompletion uses the gateway's configured broker endpoint for all
// internal evaluator phases, keeping provider credentials out of the harness.
func brokerCompletion(ctx context.Context, tc *TurnContext, model string, messages []map[string]string) (string, error) {
	if tc == nil || tc.Resolved == nil || strings.TrimSpace(model) == "" {
		return "", errors.New("broker completion unavailable")
	}
	payload, err := json.Marshal(map[string]any{
		"model":    model,
		"stream":   false,
		"messages": messages,
	})
	if err != nil {
		return "", err
	}
	timeout := tc.Timeout
	if timeout <= 0 || timeout > 15*time.Second {
		timeout = 15 * time.Second
	}
	reqCtx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()
	req, err := http.NewRequestWithContext(reqCtx, http.MethodPost,
		strings.TrimRight(tc.Resolved.UpstreamBaseURL, "/")+"/v1/chat/completions", bytes.NewReader(payload))
	if err != nil {
		return "", err
	}
	req.Header.Set("Content-Type", "application/json")
	if tc.APIKey != "" {
		req.Header.Set("Authorization", "Bearer "+tc.APIKey)
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return "", fmt.Errorf("broker completion status %d", resp.StatusCode)
	}
	var completion struct {
		Choices []struct {
			Message struct {
				Content   string `json:"content"`
				Reasoning string `json:"reasoning"`
			} `json:"message"`
		} `json:"choices"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&completion); err != nil || len(completion.Choices) == 0 {
		return "", errors.New("broker returned no completion")
	}
	text := strings.TrimSpace(completion.Choices[0].Message.Content)
	if text == "" {
		text = strings.TrimSpace(completion.Choices[0].Message.Reasoning)
	}
	return text, nil
}

// runMultiDraft builds independently generated candidates, synthesizes one
// answer, and evaluates that answer. The caller keeps all phases buffered.
func runMultiDraft(ctx context.Context, tc *TurnContext, env *TurnEnvelope, cfg operatorstore.EvaluatorConfig, prompt string) (string, error) {
	if cfg.SynthesizeModelID == "" {
		return "", errors.New("multi_draft requires synthesize_model_id")
	}
	models := availableDraftModels(tc, cfg.DraftCount)
	if len(models) == 0 {
		return "", errors.New("multi_draft has no available draft model")
	}
	if tc.RouteLog != nil {
		tc.RouteLog.Info("harness multi-draft started", "msg", naming.MsgHarnessMultiDraftPhase,
			"assistant_id", tc.AssistantID(), "turn_index", tc.TurnIndex, "stage", "evaluator",
			"module", operatorstore.HarnessModuleEvaluator, "phase", "draft", "draft_count", len(models),
			"models", models, "timeline_kind", naming.TimelineKindBroker)
	}
	drafts := make([]string, len(models))
	errs := make([]error, len(models))
	var wg sync.WaitGroup
	for i, model := range models {
		wg.Add(1)
		go func(i int, model string) {
			defer wg.Done()
			drafts[i], errs[i] = brokerCompletion(ctx, tc, model, []map[string]string{{"role": "user", "content": prompt}})
		}(i, model)
	}
	wg.Wait()
	valid := make([]string, 0, len(drafts))
	for i, draft := range drafts {
		if errs[i] == nil && draft != "" {
			valid = append(valid, draft)
		}
	}
	if len(valid) == 0 {
		return "", errors.New("all multi-draft completions failed")
	}
	var synthesis strings.Builder
	synthesis.WriteString("Synthesize one accurate, direct answer from the candidate drafts below. Do not mention drafting or this instruction.\n\n")
	for i, draft := range valid {
		fmt.Fprintf(&synthesis, "Candidate %d:\n%s\n\n", i+1, draft)
	}
	answer, err := brokerCompletion(ctx, tc, cfg.SynthesizeModelID, []map[string]string{{"role": "user", "content": synthesis.String()}})
	if err != nil {
		return "", err
	}
	if tc.RouteLog != nil {
		tc.RouteLog.Info("harness multi-draft completed", "msg", naming.MsgHarnessMultiDraftPhase,
			"assistant_id", tc.AssistantID(), "turn_index", tc.TurnIndex, "stage", "evaluator",
			"module", operatorstore.HarnessModuleEvaluator, "phase", "synthesize", "draft_count", len(valid),
			"models", append(models, cfg.SynthesizeModelID), "timeline_kind", naming.TimelineKindBroker)
	}
	if err := runEvaluator(ctx, tc, env, cfg, answer); err != nil {
		return "", err
	}
	return answer, nil
}

func availableDraftModels(tc *TurnContext, count int) []string {
	if tc == nil {
		return nil
	}
	models := make([]string, 0, count)
	for _, model := range tc.Stack.Fallback {
		if model != "" && (tc.ModelAvailable == nil || tc.ModelAvailable(model)) {
			models = append(models, model)
			if len(models) == count {
				break
			}
		}
	}
	return models
}

func completionText(raw []byte) string {
	var completion struct {
		Choices []struct {
			Message struct {
				Content   string `json:"content"`
				Reasoning string `json:"reasoning"`
			} `json:"message"`
		} `json:"choices"`
	}
	if json.Unmarshal(raw, &completion) == nil && len(completion.Choices) > 0 {
		if text := strings.TrimSpace(completion.Choices[0].Message.Content); text != "" {
			return text
		}
		return strings.TrimSpace(completion.Choices[0].Message.Reasoning)
	}
	// Buffered SSE frames are intentionally parsed minimally: evaluator needs
	// only the accumulated assistant delta, never raw client event formatting.
	var content, reasoning strings.Builder
	for _, line := range strings.Split(string(raw), "\n") {
		line = strings.TrimSpace(line)
		if !strings.HasPrefix(line, "data:") {
			continue
		}
		var event struct {
			Choices []struct {
				Delta struct {
					Content   string `json:"content"`
					Reasoning string `json:"reasoning"`
				} `json:"delta"`
			} `json:"choices"`
		}
		if json.Unmarshal([]byte(strings.TrimSpace(strings.TrimPrefix(line, "data:"))), &event) == nil {
			for _, choice := range event.Choices {
				content.WriteString(choice.Delta.Content)
				reasoning.WriteString(choice.Delta.Reasoning)
			}
		}
	}
	if text := strings.TrimSpace(content.String()); text != "" {
		return text
	}
	return strings.TrimSpace(reasoning.String())
}
