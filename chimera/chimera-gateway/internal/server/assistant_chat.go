package server

import (
	"context"
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"
	"strings"
	"time"

	"github.com/lynn/porcelain/chimera/chimera-gateway/internal/assistant"
	"github.com/lynn/porcelain/chimera/chimera-gateway/internal/chat"
	"github.com/lynn/porcelain/chimera/chimera-gateway/internal/conversationhistory"
	"github.com/lynn/porcelain/chimera/chimera-gateway/internal/operatorstore"
	"github.com/lynn/porcelain/chimera/chimera-gateway/internal/rag"
	"github.com/lynn/porcelain/chimera/chimera-gateway/internal/transform"
	"github.com/lynn/porcelain/chimera/chimera-gateway/internal/vectorstore"
	"github.com/lynn/porcelain/chimera/internal/config"
	"github.com/lynn/porcelain/internal/naming"
)

func assistantsForCatalog(rt *Runtime, principalID string) []*assistant.Assistant {
	reg := rt.Assistants()
	if reg != nil {
		return reg.ListCatalog(principalID)
	}
	return nil
}

func openAIModelEntry(id, description string) map[string]any {
	entry := map[string]any{
		"id":       id,
		"object":   "model",
		"created":  time.Now().Unix(),
		"owned_by": "chimera",
	}
	if strings.TrimSpace(description) != "" {
		entry["description"] = description
	}
	return entry
}

func prependAssistantsToCatalog(data []any, rt *Runtime, principalID string) []any {
	assistants := assistantsForCatalog(rt, principalID)
	if len(assistants) == 0 {
		return data
	}
	out := make([]any, 0, len(assistants)+len(data))
	for _, a := range assistants {
		out = append(out, openAIModelEntry(a.ModelID, a.Description))
	}
	return append(out, data...)
}

type assistantChatContext struct {
	assistant    *assistant.Assistant
	fallback     []string
	toolEnabled  bool
	routerModels []string
	toolThresh   float64
}

func resolveAssistantChat(rt *Runtime, clientModel, principalID string) (*assistantChatContext, int, map[string]any) {
	reg := rt.Assistants()
	if reg == nil {
		return nil, 0, nil
	}
	a, err := reg.Resolve(clientModel, principalID)
	if err == nil {
		return &assistantChatContext{
			assistant:    a,
			fallback:     a.FallbackChain,
			toolEnabled:  a.ToolRouterEnabled,
			routerModels: a.RouterModels,
			toolThresh:   a.ToolRouterConfidence,
		}, 0, nil
	}
	if errors.Is(err, assistant.ErrForbidden) {
		return nil, http.StatusForbidden, map[string]any{
			"error": map[string]any{"message": "Assistant not accessible", "type": "invalid_request"},
		}
	}
	if store := rt.OperatorStore(); store != nil && errors.Is(err, assistant.ErrNotFound) {
		row, dbErr := store.GetVirtualModelByModelID(context.Background(), clientModel)
		if dbErr == nil && row != nil {
			if !row.Enabled {
				return nil, http.StatusNotFound, map[string]any{
					"error": map[string]any{"message": "Assistant is disabled", "type": "invalid_request"},
				}
			}
			if row.Visibility == operatorstore.VisibilityPrivate &&
				row.CreatedByPrincipalID != "" && row.CreatedByPrincipalID != principalID {
				return nil, http.StatusForbidden, map[string]any{
					"error": map[string]any{"message": "Assistant not accessible", "type": "invalid_request"},
				}
			}
		}
	}
	return nil, 0, nil
}

func routeLogWithAssistant(routeLog *slog.Logger, assistantModelID string) *slog.Logger {
	if routeLog == nil || assistantModelID == "" {
		return routeLog
	}
	return routeLog.With("assistant_id", assistantModelID)
}

func handleAssistantChat(
	ctx context.Context,
	w http.ResponseWriter,
	rt *Runtime,
	res *config.Resolved,
	chatCtx *assistantChatContext,
	raw map[string]json.RawMessage,
	stream bool,
	skipToolRouter bool,
	headerThresh float64,
	routeLog *slog.Logger,
	cid string,
	turnIdx int,
	rid string,
	sessTenant string,
	proj string,
	flav string,
	apiKey string,
	rtDur time.Duration,
	chatOpts *chat.ProxyOpts,
	histRec *conversationhistory.Recorder,
) bool {
	a := chatCtx.assistant
	if a == nil {
		return false
	}
	assistantID := a.ModelID
	routeLog = routeLogWithAssistant(routeLog, assistantID)

	th := chatCtx.toolThresh
	if headerThresh > 0 {
		th = headerThresh
	}
	raw, trSum := transform.ApplyToolRouter(ctx, raw, transform.Config{
		Enabled:      chatCtx.toolEnabled && !skipToolRouter,
		RouterModels: chatCtx.routerModels,
		Threshold:    th,
		BaseURL:      res.UpstreamBaseURL,
		APIKey:       apiKey,
		HTTPTimeout:  rtDur,
		Log:          routeLog,
		OnAttempt: func(model string, err error) {
			rt.NoteToolRouterAttempt(model, err)
		},
	})
	if routeLog != nil && trSum.Ran {
		errStr := ""
		if trSum.Err != nil {
			errStr = trSum.Err.Error()
			if len(errStr) > 300 {
				errStr = errStr[:300] + "…"
			}
		}
		routeLog.Debug("conversation tool router", "msg", naming.MsgConversationToolRouter,
			"tools_before", trSum.ToolsBefore, "tools_after", trSum.ToolsAfter,
			"router_model", trSum.RouterModel, "assistant_id", assistantID,
			"err", errStr, "timeline_kind", naming.TimelineKindBroker)
	}

	coords := vectorstore.Coords{TenantID: sessTenant, ProjectID: proj, FlavorID: flav}
	collection := vectorstore.CollectionName(coords)
	var ragHits []vectorstore.Hit
	if !res.RAG.Enabled || rt.RAG() == nil {
		if routeLog != nil {
			routeLog.Debug("conversation RAG skipped", "msg", naming.MsgConversationRagSkipped,
				"reason", "disabled", "assistant_id", assistantID, "timeline_kind", naming.TimelineKindVectorstore)
		}
	} else if q := rag.LastUserText(raw["messages"]); strings.TrimSpace(q) == "" {
		if routeLog != nil {
			routeLog.Debug("conversation RAG skipped", "msg", naming.MsgConversationRagSkipped,
				"reason", "empty_query", "assistant_id", assistantID, "timeline_kind", naming.TimelineKindVectorstore)
		}
	} else {
		hits, rerr := rt.RAG().Retrieve(ctx, rag.RetrieveRequest{
			Coords: coords, Query: q, RequestID: rid, ConversationID: cid, TurnIndex: turnIdx, LifecycleLog: routeLog,
		})
		if rerr != nil {
			if routeLog != nil {
				routeLog.Warn("rag retrieve failed; proceeding without context", "msg", "rag.retrieve.error", "err", rerr,
					"assistant_id", assistantID, "timeline_kind", naming.TimelineKindVectorstore)
			}
		} else if ctxBlock := rag.FormatRetrievedContext(hits); ctxBlock != "" {
			ragHits = hits
			rag.InjectSystemMessage(raw, ctxBlock)
			if routeLog != nil {
				routeLog.Info("conversation RAG attached", "msg", naming.MsgConversationRagAttached,
					"assistant_id", assistantID, "tenant", coords.TenantID, "project", coords.ProjectID,
					"flavor", coords.FlavorID, "hits", len(hits), "collection", collection,
					"timeline_kind", naming.TimelineKindVectorstore)
			}
		}
	}

	emitConversationRequestWitness(routeLog, res, raw)

	tenantSnap := rt.ProviderModelAvailability(sessTenant)
	modelAvailable := func(id string) bool { return tenantSnap.IsAvailable(id) }

	initial, _ := assistant.PickInitialModelWithAvailability(a, raw, routeLog, modelAvailable)
	if initial == "" {
		if routeLog != nil {
			routeLog.Warn("conversation errored", "msg", naming.MsgConversationErrored,
				"statusCode", http.StatusServiceUnavailable, "errorType", "gateway_config",
				"assistant_id", assistantID, "timeline_kind", naming.TimelineKindBroker)
		}
		errBody := map[string]any{
			"error": map[string]any{
				"message": "Could not resolve an initial upstream model for the assistant (check routing policy and fallback chain).",
				"type":    "gateway_config",
			},
		}
		if histRec != nil {
			histRec.SetRAGHits(ragHits)
			histRec.PersistGatewayError(http.StatusServiceUnavailable, errBody)
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusServiceUnavailable)
		_ = json.NewEncoder(w).Encode(errBody)
		return true
	}
	if routeLog != nil {
		routeLog.Info("chat routing resolved", "msg", "chat.routing.resolved",
			"assistant_id", assistantID, "clientModel", assistantID, "upstreamModel", initial,
			"timeline_kind", naming.TimelineKindBroker)
	}
	rag.WriteResponseHeaders(w, initial, ragHits)
	if histRec != nil {
		histRec.SetRAGHits(ragHits)
	}
	if chatOpts == nil {
		chatOpts = &chat.ProxyOpts{}
	} else {
		cp := *chatOpts
		chatOpts = &cp
	}
	chatOpts.ModelAvailable = modelAvailable
	chatOpts.AssistantID = assistantID
	chat.WithVirtualModelFallback(ctx, w, initial, chatCtx.fallback, res.UpstreamBaseURL, apiKey, stream, raw,
		chatTimeout(res), routeLog, rt.Metrics(), rt.LimitsGuard(), chatOpts)
	return true
}
