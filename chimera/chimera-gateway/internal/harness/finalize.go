package harness

import (
	"strings"

	"github.com/lynn/porcelain/chimera/chimera-gateway/internal/transform"
	"github.com/lynn/porcelain/chimera/chimera-gateway/internal/vectorstore"
	"github.com/lynn/porcelain/internal/naming"
)

// RecordStageCompletion appends a completed stage name to the envelope plan.
func RecordStageCompletion(env *TurnEnvelope, stageName string) {
	if env == nil || strings.TrimSpace(stageName) == "" {
		return
	}
	env.Plan.Stages = append(env.Plan.Stages, stageName)
}

// ApplyRetrievalToEnvelope updates retrieval fields from delivered RAG evidence.
func ApplyRetrievalToEnvelope(env *TurnEnvelope, hits []vectorstore.Hit, topK int, strategy string) {
	if env == nil {
		return
	}
	if len(hits) == 0 {
		return
	}
	env.Retrieval.Ran = true
	if topK > 0 {
		env.Retrieval.TopK = &topK
	}
	if strategy = strings.TrimSpace(strategy); strategy != "" {
		env.Retrieval.CompressStrategy = &strategy
	}
	env.Retrieval.HitsCount = len(hits)
	ids := make([]string, 0, len(hits))
	for _, h := range hits {
		id := strings.TrimSpace(h.ID)
		if id != "" {
			ids = append(ids, id)
		}
	}
	env.Retrieval.EvidenceIDs = ids
}

// ApplyToolRouterToEnvelope records tool-router summary in response metadata.
func ApplyToolRouterToEnvelope(env *TurnEnvelope, sum transform.ToolRouterSummary) {
	if env == nil || !sum.Ran {
		return
	}
	if env.Response.Metadata == nil {
		env.Response.Metadata = map[string]any{}
	}
	env.Response.Metadata["tool_router_ran"] = true
	env.Response.Metadata["tools_before"] = sum.ToolsBefore
	env.Response.Metadata["tools_after"] = sum.ToolsAfter
	if sum.RouterModel != "" {
		env.Response.Metadata["router_model"] = sum.RouterModel
	}
}

// RecordUpstreamAttempt appends a fallback attempt before proxying upstream.
func RecordUpstreamAttempt(env *TurnEnvelope, upstreamModel string, attempt int) {
	if env == nil || strings.TrimSpace(upstreamModel) == "" {
		return
	}
	env.Execution.UpstreamAttempts = append(env.Execution.UpstreamAttempts, UpstreamAttempt{
		UpstreamModel: upstreamModel,
		Attempt:       attempt,
	})
}

// SetResolvedModel records the upstream model that delivered the client-visible answer.
func SetResolvedModel(env *TurnEnvelope, upstreamModel string) {
	if env == nil {
		return
	}
	m := strings.TrimSpace(upstreamModel)
	if m == "" {
		return
	}
	env.Execution.ResolvedModelID = &m
}

// LogTurnCompleted emits a debug log with the redacted envelope snapshot.
func LogTurnCompleted(tc *TurnContext, env *TurnEnvelope, statusCode int) {
	if tc == nil || tc.RouteLog == nil || env == nil {
		return
	}
	redacted, err := RedactedJSON(env)
	if err != nil {
		return
	}
	tc.RouteLog.Debug("harness turn completed",
		"msg", naming.MsgHarnessTurnCompleted,
		"assistant_id", env.AssistantID,
		"turn_index", env.TurnIndex,
		"status_code", statusCode,
		"harness_summary", string(redacted),
		"timeline_kind", naming.TimelineKindBroker,
	)
}
