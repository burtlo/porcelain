package harness

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"

	"github.com/lynn/porcelain/chimera/chimera-gateway/internal/assistant"
	"github.com/lynn/porcelain/chimera/chimera-gateway/internal/chat"
	"github.com/lynn/porcelain/chimera/chimera-gateway/internal/harness/evidence"
	"github.com/lynn/porcelain/chimera/chimera-gateway/internal/operatorstore"
	"github.com/lynn/porcelain/chimera/chimera-gateway/internal/rag"
	"github.com/lynn/porcelain/chimera/chimera-gateway/internal/transform"
	"github.com/lynn/porcelain/chimera/chimera-gateway/internal/vectorstore"
	"github.com/lynn/porcelain/internal/naming"
)

// StackResolveStage validates the resolved assistant stack is present.
type StackResolveStage struct{}

func (StackResolveStage) Name() string   { return "stack_resolve" }
func (StackResolveStage) Module() string { return "primary" }

func (StackResolveStage) Run(_ context.Context, tc *TurnContext, env *TurnEnvelope, _ Body) error {
	if tc == nil || tc.Stack.VM == nil {
		return &AbortError{
			Status: http.StatusServiceUnavailable,
			Body: map[string]any{
				"error": map[string]any{
					"message": "Assistant stack unavailable",
					"type":    "gateway_config",
				},
			},
		}
	}
	env.AssistantID = tc.Stack.VM.ModelID
	return nil
}

// ToolRouterStage slims client tool declarations when enabled (fail-open).
type ToolRouterStage struct{}

func (ToolRouterStage) Name() string   { return "tool_router" }
func (ToolRouterStage) Module() string { return "tool_router" }

func (ToolRouterStage) Run(ctx context.Context, tc *TurnContext, env *TurnEnvelope, body Body) error {
	if tc == nil || tc.Resolved == nil {
		return nil
	}
	th := tc.Stack.ToolThresh
	if tc.HeaderToolThresh > 0 {
		th = tc.HeaderToolThresh
	}
	out, sum := transform.ApplyToolRouter(ctx, body, transform.Config{
		Enabled:      tc.Stack.ToolEnabled && !tc.SkipToolRouter,
		RouterModels: tc.Stack.RouterModels,
		Threshold:    th,
		BaseURL:      tc.Resolved.UpstreamBaseURL,
		APIKey:       tc.APIKey,
		HTTPTimeout:  tc.Timeout,
		Log:          tc.RouteLog,
		OnAttempt:    tc.OnToolRouterAttempt,
	})
	replaceBody(body, out)
	tc.ToolRouter = sum
	ApplyToolRouterToEnvelope(env, sum)
	if tc.RouteLog != nil && sum.Ran {
		errStr := ""
		if sum.Err != nil {
			errStr = sum.Err.Error()
			if len(errStr) > 300 {
				errStr = errStr[:300] + "…"
			}
		}
		tc.RouteLog.Debug("conversation tool router", "msg", naming.MsgConversationToolRouter,
			"tools_before", sum.ToolsBefore, "tools_after", sum.ToolsAfter,
			"router_model", sum.RouterModel, "assistant_id", tc.AssistantID(),
			"err", errStr, "timeline_kind", naming.TimelineKindBroker)
	}
	return nil
}

// RetrievalStage retrieves workspace context and injects a system message (fail-open).
type RetrievalStage struct{}

func (RetrievalStage) Name() string   { return "retrieval" }
func (RetrievalStage) Module() string { return "retrieval" }

func (RetrievalStage) Run(ctx context.Context, tc *TurnContext, env *TurnEnvelope, body Body) error {
	if tc == nil {
		return nil
	}
	if tc.BodyBeforeRetrieval == nil {
		tc.BodyBeforeRetrieval = cloneBody(body)
	}
	assistantID := tc.AssistantID()
	coords := vectorstore.Coords{TenantID: tc.TenantID, ProjectID: tc.ProjectID, FlavorID: tc.FlavorID}
	collection := vectorstore.CollectionName(coords)
	retrievalOn := true
	if tc.Stack.VM != nil {
		retrievalOn = tc.Stack.VM.HarnessEnabled(operatorstore.HarnessModuleRetrieval)
	}
	if tc.Resolved == nil || !tc.Resolved.RAG.Enabled || tc.RAG == nil || !retrievalOn {
		if tc.RouteLog != nil {
			reason := "disabled"
			if tc.Resolved != nil && tc.Resolved.RAG.Enabled && tc.RAG != nil && !retrievalOn {
				reason = "vm_harness_off"
			}
			tc.RouteLog.Debug("conversation RAG skipped", "msg", naming.MsgConversationRagSkipped,
				"reason", reason, "assistant_id", assistantID, "timeline_kind", naming.TimelineKindVectorstore)
		}
		return nil
	}
	q := rag.LastUserText(body["messages"])
	if strings.TrimSpace(q) == "" {
		if tc.RouteLog != nil {
			tc.RouteLog.Debug("conversation RAG skipped", "msg", naming.MsgConversationRagSkipped,
				"reason", "empty_query", "assistant_id", assistantID, "timeline_kind", naming.TimelineKindVectorstore)
		}
		return nil
	}
	defaultTopK := tc.RAG.TopK()
	defaultScoreFloor := tc.RAG.ScoreThreshold()
	retrievalCfg := operatorstore.ParseRetrievalConfig("", defaultTopK, defaultScoreFloor)
	if tc.Stack.VM != nil {
		if module, ok := tc.Stack.VM.HarnessModules[operatorstore.HarnessModuleRetrieval]; ok {
			retrievalCfg = operatorstore.ParseRetrievalConfig(module.ConfigJSON, defaultTopK, defaultScoreFloor)
		}
	}
	if tc.RetrievalTopKOverride > retrievalCfg.TopK {
		retrievalCfg.TopK = tc.RetrievalTopKOverride
	}
	if shouldSkipRetrieval(retrievalCfg.SkipIf, tc, env) {
		if tc.RouteLog != nil {
			tc.RouteLog.Debug("conversation RAG skipped", "msg", naming.MsgConversationRagSkipped,
				"reason", "module_skip_if", "assistant_id", assistantID, "timeline_kind", naming.TimelineKindVectorstore)
		}
		return nil
	}
	hits, rerr := tc.RAG.Retrieve(ctx, rag.RetrieveRequest{
		Coords: coords, Query: q, RequestID: tc.RequestID, ConversationID: tc.ConversationID,
		TurnIndex: tc.TurnIndex, LifecycleLog: tc.RouteLog, TopK: retrievalCfg.TopK,
		ScoreThreshold: retrievalCfg.ScoreFloor,
	})
	if rerr != nil {
		if tc.RouteLog != nil {
			tc.RouteLog.Warn("rag retrieve failed; proceeding without context", "msg", naming.MsgRagRetrieveError,
				"err", rerr, "assistant_id", assistantID, "timeline_kind", naming.TimelineKindVectorstore)
		}
		return nil
	}
	if len(hits) > 0 {
		compressCfg := evidence.Config{
			Strategy: retrievalCfg.CompressStrategy, SummarizeModelID: retrievalCfg.SummarizeModelID,
			UpstreamBaseURL: tc.Resolved.UpstreamBaseURL, APIKey: tc.APIKey, HTTPTimeout: tc.Timeout,
		}
		block, strategy, cerr := evidence.New(retrievalCfg.CompressStrategy).Compress(ctx, hits, retrievalCfg.MaxContextChars, compressCfg)
		if cerr != nil && retrievalCfg.CompressStrategy == "summarize" {
			if tc.RouteLog != nil {
				tc.RouteLog.Warn("retrieval summarize failed; using truncated evidence",
					"msg", naming.MsgHarnessRetrievalCompressFallback, "assistant_id", assistantID,
					"turn_index", tc.TurnIndex, "stage", "retrieval", "module", operatorstore.HarnessModuleRetrieval,
					"err", cerr, "timeline_kind", naming.TimelineKindVectorstore)
			}
			block, strategy, cerr = evidence.New("truncate").Compress(ctx, hits, retrievalCfg.MaxContextChars, compressCfg)
		}
		if cerr != nil {
			if tc.RouteLog != nil {
				tc.RouteLog.Warn("retrieval evidence compression failed; proceeding without context",
					"msg", naming.MsgRagRetrieveError, "err", cerr, "assistant_id", assistantID,
					"timeline_kind", naming.TimelineKindVectorstore)
			}
			return nil
		}
		if strings.TrimSpace(block.Text) == "" {
			return nil
		}
		tc.RAGHits = block.Hits
		rag.InjectSystemMessage(body, block.Text)
		ApplyRetrievalToEnvelope(env, block.Hits, retrievalCfg.TopK, strategy)
		if tc.RouteLog != nil {
			tc.RouteLog.Info("conversation RAG attached", "msg", naming.MsgConversationRagAttached,
				"assistant_id", assistantID, "tenant", coords.TenantID, "project", coords.ProjectID,
				"flavor", coords.FlavorID, "hits", len(block.Hits), "collection", collection,
				"timeline_kind", naming.TimelineKindVectorstore)
		}
	}
	return nil
}

func shouldSkipRetrieval(skipIf []string, tc *TurnContext, env *TurnEnvelope) bool {
	for _, rule := range skipIf {
		switch strings.ToLower(strings.TrimSpace(rule)) {
		case "no_workspace":
			if tc == nil || (strings.TrimSpace(tc.ProjectID) == "" && strings.TrimSpace(tc.FlavorID) == "") {
				return true
			}
			if env != nil && env.Scope.WorkspaceID != nil && *env.Scope.WorkspaceID == 0 {
				return true
			}
		case "empty_query":
			// The query is checked before retrieval; retain this value as a declarative config option.
		default:
			if env != nil {
				for _, tag := range env.Intent.Tags {
					if strings.EqualFold(strings.TrimSpace(rule), strings.TrimSpace(tag)) {
						return true
					}
				}
			}
		}
	}
	return false
}

// RequestWitnessStage logs the normalized request payload snapshot.
type RequestWitnessStage struct{}

func (RequestWitnessStage) Name() string   { return "request_witness" }
func (RequestWitnessStage) Module() string { return "telemetry" }

func (RequestWitnessStage) Run(_ context.Context, tc *TurnContext, _ *TurnEnvelope, body Body) error {
	if tc != nil && tc.EmitRequestWitness != nil {
		tc.EmitRequestWitness(tc.RouteLog, tc.Resolved, body)
	}
	return nil
}

// InitialPickStage selects the first upstream model from routing policy and chain.
type InitialPickStage struct{}

func (InitialPickStage) Name() string   { return "initial_pick" }
func (InitialPickStage) Module() string { return "primary" }

func (InitialPickStage) Run(_ context.Context, tc *TurnContext, env *TurnEnvelope, body Body) error {
	if tc == nil || tc.Stack.VM == nil {
		return nil
	}
	assistantID := tc.AssistantID()
	modelAvailable := tc.ModelAvailable
	if modelAvailable == nil {
		modelAvailable = func(string) bool { return true }
	}
	stackVM := *tc.Stack.VM
	stackVM.FallbackChain = tc.Stack.Fallback
	initial, _ := assistant.PickInitialModelWithAvailability(&stackVM, body, tc.RouteLog, modelAvailable)
	if initial == "" {
		if tc.RouteLog != nil {
			tc.RouteLog.Warn("conversation errored", "msg", naming.MsgConversationErrored,
				"statusCode", http.StatusServiceUnavailable, "errorType", "gateway_config",
				"assistant_id", assistantID, "timeline_kind", naming.TimelineKindBroker)
		}
		errBody := map[string]any{
			"error": map[string]any{
				"message": "Could not resolve an initial upstream model for the assistant (check routing policy and fallback chain).",
				"type":    "gateway_config",
			},
		}
		if tc.HistRec != nil {
			tc.HistRec.SetRAGHits(tc.RAGHits)
			if env != nil {
				if b, err := RedactedJSON(env); err == nil {
					tc.HistRec.SetHarnessSummary(b)
				}
			}
			tc.HistRec.PersistGatewayError(http.StatusServiceUnavailable, errBody)
		}
		return &AbortError{Status: http.StatusServiceUnavailable, Body: errBody}
	}
	tc.InitialModel = initial
	env.Plan.PrimaryModelID = &initial
	if tc.RouteLog != nil {
		tc.RouteLog.Info("chat routing resolved", "msg", naming.MsgChatRoutingResolved,
			"assistant_id", assistantID, "clientModel", assistantID, "upstreamModel", initial,
			"timeline_kind", naming.TimelineKindBroker)
	}
	return nil
}

// FallbackProxyStage proxies to upstream with assistant fallback retry semantics.
type FallbackProxyStage struct{}

func (FallbackProxyStage) Name() string   { return "fallback_proxy" }
func (FallbackProxyStage) Module() string { return "primary" }

func (FallbackProxyStage) Run(ctx context.Context, tc *TurnContext, env *TurnEnvelope, body Body) error {
	if tc == nil || tc.Resolved == nil || tc.InitialModel == "" {
		return ErrTurnComplete
	}
	rag.WriteResponseHeaders(tc.W, tc.InitialModel, tc.RAGHits)
	WriteSummaryHeader(tc.W, env)
	if tc.HistRec != nil {
		tc.HistRec.SetRAGHits(tc.RAGHits)
	}
	opts := tc.ChatOpts
	if opts == nil {
		opts = &chat.ProxyOpts{}
	} else {
		cp := *opts
		opts = &cp
	}
	if tc.ModelAvailable != nil {
		opts.ModelAvailable = tc.ModelAvailable
	}
	opts.AssistantID = tc.AssistantID()
	evaluator, evaluatorEnabled := evaluatorConfig(tc)
	if evaluatorEnabled {
		logStreamPolicy(tc, evaluator.StreamPolicy)
	}
	if escalation, escalationEnabled := escalationConfig(tc); escalationEnabled {
		mergeHumanPasteBack(body, escalation, tc)
	}
	if evaluatorEnabled && evaluator.Mode == "multi_draft" {
		// Multi-draft always buffers: the synthesized response is the only
		// client-visible answer. `immediate` remains telemetry-only for the
		// single-pass evaluator and is not meaningful for this mode.
		buffer := httptest.NewRecorder()
		answer, err := runMultiDraft(ctx, tc, env, evaluator, lastUserText(body))
		if err != nil {
			logEvaluatorFailure(tc, err)
			chat.WithAssistantFallback(ctx, buffer, tc.InitialModel, tc.Stack.Fallback, tc.Resolved.UpstreamBaseURL,
				tc.APIKey, false, body, tc.Timeout, tc.RouteLog, tc.Metrics, tc.LimitsGuard, opts)
		} else {
			writeCompletionResponse(buffer, answer, evaluator.SynthesizeModelID)
			SetResolvedModel(env, evaluator.SynthesizeModelID)
			if env != nil && env.Evaluation.RecommendEscalation {
				applyEscalation(ctx, tc, env, body, evaluator.StreamPolicy, buffer)
			}
		}
		LogTurnCompleted(tc, env, buffer.Code)
		if tc.HistRec != nil {
			if b, err := RedactedJSON(env); err == nil {
				tc.HistRec.SetHarnessSummary(b)
			}
		}
		writeBufferedResponse(tc.W, buffer, tc.Stream)
		return ErrTurnComplete
	}
	if env != nil {
		opts.OnFallbackAttempt = func(upstreamModel string, attempt int) {
			RecordUpstreamAttempt(env, upstreamModel, attempt)
		}
	}
	prevCaptured := opts.OnResponseCaptured
	opts.OnResponseCaptured = func(statusCode int, upstreamModel string, stream bool, body []byte) {
		if env != nil {
			SetResolvedModel(env, upstreamModel)
			if len(env.Execution.UpstreamAttempts) > 0 {
				last := len(env.Execution.UpstreamAttempts) - 1
				env.Execution.UpstreamAttempts[last].Status = statusCode
			}
			if evaluatorEnabled {
				if err := runEvaluator(ctx, tc, env, evaluator, completionText(body)); err != nil {
					logEvaluatorFailure(tc, err)
				}
			}
			LogTurnCompleted(tc, env, statusCode)
			if tc.HistRec != nil {
				if b, err := RedactedJSON(env); err == nil {
					tc.HistRec.SetHarnessSummary(b)
				}
			}
		}
		if prevCaptured != nil {
			prevCaptured(statusCode, upstreamModel, stream, body)
		}
	}
	if toolExecutorModuleEnabled(tc) {
		if tc.ToolExecutor == nil {
			applyWorkspaceScope(ctx, tc, env)
		}
		if tc.ToolExecutor != nil {
			runWorkspaceToolLoop(ctx, tc, env, body, opts)
			return ErrTurnComplete
		}
		logToolExecutorUnavailable(tc, env)
	}
	if !evaluatorEnabled || evaluator.StreamPolicy == operatorstore.StreamPolicyImmediate {
		chat.WithAssistantFallback(ctx, tc.W, tc.InitialModel, tc.Stack.Fallback, tc.Resolved.UpstreamBaseURL,
			tc.APIKey, tc.Stream, body, tc.Timeout, tc.RouteLog, tc.Metrics, tc.LimitsGuard, opts)
		return ErrTurnComplete
	}

	// Gate policies deliberately execute the existing fallback helper against a
	// recorder, preserving its admission/retry semantics while withholding bytes
	// until the evaluator has decided whether escalation is needed.
	buffer := httptest.NewRecorder()
	chat.WithAssistantFallback(ctx, buffer, tc.InitialModel, tc.Stack.Fallback, tc.Resolved.UpstreamBaseURL,
		tc.APIKey, false, body, tc.Timeout, tc.RouteLog, tc.Metrics, tc.LimitsGuard, opts)
	if env != nil && env.Evaluation.RecommendEscalation {
		applyEscalation(ctx, tc, env, body, evaluator.StreamPolicy, buffer)
	}
	writeBufferedResponse(tc.W, buffer, tc.Stream)
	return ErrTurnComplete
}

func logStreamPolicy(tc *TurnContext, policy string) {
	if tc == nil || tc.RouteLog == nil {
		return
	}
	tc.RouteLog.Info("harness evaluator stream policy applied", "msg", naming.MsgHarnessStreamPolicyApplied,
		"assistant_id", tc.AssistantID(), "turn_index", tc.TurnIndex, "stage", "fallback_proxy",
		"module", operatorstore.HarnessModuleEvaluator, "stream_policy", policy,
		"timeline_kind", naming.TimelineKindBroker)
}

func logEvaluatorFailure(tc *TurnContext, err error) {
	if tc == nil || tc.RouteLog == nil {
		return
	}
	tc.RouteLog.Warn("harness evaluator failed; delivering primary response", "msg", naming.MsgHarnessEvaluatorFailed,
		"assistant_id", tc.AssistantID(), "turn_index", tc.TurnIndex, "stage", "evaluator",
		"module", operatorstore.HarnessModuleEvaluator, "err", err, "timeline_kind", naming.TimelineKindBroker)
}

func applyEscalation(ctx context.Context, tc *TurnContext, env *TurnEnvelope, body Body, streamPolicy string, buffer *httptest.ResponseRecorder) {
	cfg, enabled := escalationConfig(tc)
	if !enabled || cfg.MaxRounds == 0 || len(cfg.OnFail) == 0 || env.Escalation.Rounds >= cfg.MaxRounds {
		if env.Response.Metadata == nil {
			env.Response.Metadata = map[string]any{}
		}
		env.Response.Metadata["escalation_exhausted"] = true
		return
	}
	for _, action := range cfg.OnFail {
		if action == "human" {
			// External escalation is emitted only after the internal action
			// loop below has consumed its configured opportunity.
			continue
		}
		env.Escalation.Rounds++
		env.Escalation.LastAction = &action
		logEscalation(tc, naming.MsgHarnessEscalationStarted, action, "started")
		switch action {
		case "fallback_chain":
			resolved := ""
			if env.Execution.ResolvedModelID != nil {
				resolved = *env.Execution.ResolvedModelID
			}
			next := nextFallback(resolved, tc.Stack.Fallback)
			if len(next) > 0 {
				nextBuffer := httptest.NewRecorder()
				chat.WithAssistantFallback(ctx, nextBuffer, next[0], next, tc.Resolved.UpstreamBaseURL,
					tc.APIKey, false, body, tc.Timeout, tc.RouteLog, tc.Metrics, tc.LimitsGuard, nil)
				if nextBuffer.Code >= 200 && nextBuffer.Code < 300 {
					*buffer = *nextBuffer
					logEscalation(tc, naming.MsgHarnessEscalationCompleted, action, "completed")
					return
				}
			}
		case "re_retrieve":
			if tc.BodyBeforeRetrieval == nil {
				logEscalation(tc, naming.MsgHarnessEscalationAction, action, "unavailable")
				break
			}
			topK := 4
			if env.Retrieval.TopK != nil && *env.Retrieval.TopK > 0 {
				topK = *env.Retrieval.TopK * 2
			}
			tc.RetrievalTopKOverride = topK
			replaceBody(body, cloneBody(tc.BodyBeforeRetrieval))
			_ = (RetrievalStage{}).Run(ctx, tc, env, body) // retrieval remains fail-open
			nextBuffer := httptest.NewRecorder()
			chat.WithAssistantFallback(ctx, nextBuffer, tc.InitialModel, tc.Stack.Fallback, tc.Resolved.UpstreamBaseURL,
				tc.APIKey, false, body, tc.Timeout, tc.RouteLog, tc.Metrics, tc.LimitsGuard, nil)
			if nextBuffer.Code >= 200 && nextBuffer.Code < 300 {
				*buffer = *nextBuffer
				logEscalation(tc, naming.MsgHarnessEscalationAction, action, "re_retrieved")
				return
			}
		case "ensemble":
			logEscalation(tc, naming.MsgHarnessEscalationAction, action, "placeholder")
		}
		logEscalation(tc, naming.MsgHarnessEscalationCompleted, action, "best_effort")
		if env.Escalation.Rounds >= cfg.MaxRounds {
			break
		}
	}
	if containsAction(cfg.OnFail, "human") {
		action := "human"
		env.Escalation.LastAction = &action
		writeHumanEscalationResponse(buffer, cfg, tc, body)
		logEscalation(tc, naming.MsgHarnessEscalationCompleted, action, "requested")
		return
	}
	if env.Response.Metadata == nil {
		env.Response.Metadata = map[string]any{}
	}
	env.Response.Metadata["escalation_exhausted"] = true
}

func containsAction(actions []string, want string) bool {
	for _, action := range actions {
		if action == want {
			return true
		}
	}
	return false
}

func writeCompletionResponse(w *httptest.ResponseRecorder, content, model string) {
	if w == nil {
		return
	}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	_ = json.NewEncoder(w).Encode(map[string]any{
		"object":  "chat.completion",
		"model":   model,
		"choices": []map[string]any{{"index": 0, "message": map[string]string{"role": "assistant", "content": content}, "finish_reason": "stop"}},
	})
}

func writeHumanEscalationResponse(w *httptest.ResponseRecorder, cfg operatorstore.EscalationConfig, tc *TurnContext, body Body) {
	var prompt strings.Builder
	prompt.WriteString("This request needs an external review after the configured internal attempts were exhausted.\n\n")
	if cfg.PrivacyDisclosure != "" {
		prompt.WriteString("Privacy: ")
		prompt.WriteString(cfg.PrivacyDisclosure)
		prompt.WriteString("\n\n")
	}
	if len(cfg.HumanSurfaces) > 0 {
		prompt.WriteString("Escalation surfaces:\n")
		for _, surface := range cfg.HumanSurfaces {
			fmt.Fprintf(&prompt, "- %s: %s\n", surface.Name, surface.URL)
		}
		prompt.WriteString("\n")
	}
	prompt.WriteString("Copy the request and the external answer back in your next message using this delimiter:\n")
	prompt.WriteString(cfg.PasteBackDelimiter)
	prompt.WriteString("\n")
	writeCompletionResponse(w, prompt.String(), tc.AssistantID())
}

func mergeHumanPasteBack(body Body, cfg operatorstore.EscalationConfig, tc *TurnContext) {
	raw, ok := body["messages"]
	if !ok || cfg.PasteBackDelimiter == "" {
		return
	}
	var messages []map[string]any
	if json.Unmarshal(raw, &messages) != nil || len(messages) == 0 {
		return
	}
	last := messages[len(messages)-1]
	content, _ := last["content"].(string)
	parts := strings.SplitN(content, cfg.PasteBackDelimiter, 2)
	if len(parts) != 2 || strings.TrimSpace(parts[1]) == "" {
		return
	}
	messages = append(messages, map[string]any{
		"role":    "system",
		"content": "External human answer supplied by the user. Treat it as additional conversation context:\n" + strings.TrimSpace(parts[1]),
	})
	if encoded, err := json.Marshal(messages); err == nil {
		body["messages"] = encoded
		if tc != nil && tc.RouteLog != nil {
			tc.RouteLog.Info("harness human answer merged", "msg", naming.MsgHarnessHumanPasteBackMerged,
				"assistant_id", tc.AssistantID(), "turn_index", tc.TurnIndex, "stage", "escalation",
				"module", operatorstore.HarnessModuleEscalation, "timeline_kind", naming.TimelineKindBroker)
		}
	}
}

func lastUserText(body Body) string {
	raw := body["messages"]
	var messages []map[string]any
	if json.Unmarshal(raw, &messages) != nil {
		return ""
	}
	for i := len(messages) - 1; i >= 0; i-- {
		if messages[i]["role"] == "user" {
			if content, ok := messages[i]["content"].(string); ok {
				return content
			}
		}
	}
	return ""
}

func nextFallback(resolved string, chain []string) []string {
	for i, model := range chain {
		if model == resolved && i+1 < len(chain) {
			return append([]string(nil), chain[i+1:]...)
		}
	}
	return nil
}

func logEscalation(tc *TurnContext, msg, action, outcome string) {
	if tc == nil || tc.RouteLog == nil {
		return
	}
	tc.RouteLog.Info("harness escalation", "msg", msg, "assistant_id", tc.AssistantID(),
		"turn_index", tc.TurnIndex, "stage", "escalation", "module", operatorstore.HarnessModuleEscalation,
		"action", action, "outcome", outcome, "timeline_kind", naming.TimelineKindBroker)
}

func replaceBody(dst, src Body) {
	if dst == nil || src == nil {
		return
	}
	for k := range dst {
		if _, ok := src[k]; !ok {
			delete(dst, k)
		}
	}
	for k, v := range src {
		dst[k] = v
	}
}

func cloneBody(body Body) Body {
	if body == nil {
		return nil
	}
	out := make(Body, len(body))
	for key, value := range body {
		out[key] = append(json.RawMessage(nil), value...)
	}
	return out
}

// HandleAbort writes an AbortError response to the client.
func HandleAbort(w http.ResponseWriter, err *AbortError) {
	if w == nil || err == nil {
		return
	}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(err.Status)
	_ = json.NewEncoder(w).Encode(err.Body)
}
