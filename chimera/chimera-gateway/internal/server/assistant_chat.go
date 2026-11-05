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
	"github.com/lynn/porcelain/chimera/chimera-gateway/internal/harness"
	"github.com/lynn/porcelain/chimera/chimera-gateway/internal/operatorstore"
	"github.com/lynn/porcelain/chimera/internal/config"
)

func assistantsForCatalog(rt *Runtime, principalID string) []*assistant.Resolved {
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
	vms := assistantsForCatalog(rt, principalID)
	if len(vms) == 0 {
		return data
	}
	out := make([]any, 0, len(vms)+len(data))
	for _, vm := range vms {
		out = append(out, openAIModelEntry(vm.ModelID, vm.Description))
	}
	return append(out, data...)
}

type assistantChatContext struct {
	vm           *assistant.Resolved
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
	vm, err := reg.Resolve(clientModel, principalID)
	if err == nil {
		return &assistantChatContext{
			vm:           vm,
			fallback:     vm.FallbackChain,
			toolEnabled:  vm.ToolRouterEnabled,
			routerModels: vm.RouterModels,
			toolThresh:   vm.ToolRouterConfidence,
		}, 0, nil
	}
	if errors.Is(err, assistant.ErrForbidden) {
		return nil, http.StatusForbidden, map[string]any{
			"error": map[string]any{"message": "Assistant not accessible", "type": "invalid_request"},
		}
	}
	if store := rt.OperatorStore(); store != nil && errors.Is(err, assistant.ErrNotFound) {
		row, dbErr := store.GetAssistantByModelID(context.Background(), clientModel)
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

func routeLogWithAssistant(routeLog *slog.Logger, assistantID string) *slog.Logger {
	if routeLog == nil || assistantID == "" {
		return routeLog
	}
	return routeLog.With("assistant_id", assistantID)
}

func handleAssistantChat(
	ctx context.Context,
	w http.ResponseWriter,
	rt *Runtime,
	res *config.Resolved,
	vmCtx *assistantChatContext,
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
	workspaceRowID int64,
	apiKey string,
	rtDur time.Duration,
	chatOpts *chat.ProxyOpts,
	histRec *conversationhistory.Recorder,
) bool {
	vm := vmCtx.vm
	if vm == nil {
		return false
	}
	if merged := rt.ChatResolved(ctx); merged != nil {
		res = merged
	}
	assistantID := vm.ModelID
	routeLog = routeLogWithAssistant(routeLog, assistantID)

	tenantSnap := rt.ProviderModelAvailability(sessTenant)
	modelAvailable := func(id string) bool { return tenantSnap.IsAvailable(id) }

	tc := &harness.TurnContext{
		W:        w,
		Resolved: res,
		Stack: harness.VMStack{
			VM:           vm,
			Fallback:     vmCtx.fallback,
			ToolEnabled:  vmCtx.toolEnabled,
			RouterModels: vmCtx.routerModels,
			ToolThresh:   vmCtx.toolThresh,
		},
		Stream:           stream,
		SkipToolRouter:   skipToolRouter,
		HeaderToolThresh: headerThresh,
		RouteLog:         routeLog,
		ConversationID:   cid,
		TurnIndex:        turnIdx,
		RequestID:        rid,
		TenantID:         sessTenant,
		ProjectID:        proj,
		FlavorID:         flav,
		WorkspaceRowID:   workspaceRowID,
		APIKey:           apiKey,
		Timeout:          rtDur,
		ChatOpts:         chatOpts,
		HistRec:          histRec,
		RAG:              rt.RAG(),
		CorpusStale:      rt.CorpusStaleStore(),
		OperatorStore:    rt.OperatorStore(),
		Metrics:          rt.Metrics(),
		LimitsGuard:      rt.LimitsGuard(),
		ModelAvailable:   modelAvailable,
		OnToolRouterAttempt: func(model string, err error) {
			rt.NoteToolRouterAttempt(model, err)
		},
		EmitRequestWitness: emitConversationRequestWitness,
		MCPManager:         rt.MCPManager(),
	}

	err := harness.DefaultRunner().Run(ctx, tc, harness.Body(raw))
	if err != nil {
		var abort *harness.AbortError
		if errors.As(err, &abort) {
			if tc.Envelope != nil {
				harness.LogTurnCompleted(tc, tc.Envelope, abort.Status)
			}
			harness.HandleAbort(w, abort)
			return true
		}
		return false
	}
	return true
}
