package harness

import (
	"encoding/json"
	"log/slog"
	"net/http"
	"time"

	"github.com/lynn/porcelain/chimera/chimera-gateway/internal/assistant"
	"github.com/lynn/porcelain/chimera/chimera-gateway/internal/chat"
	"github.com/lynn/porcelain/chimera/chimera-gateway/internal/conversationhistory"
	"github.com/lynn/porcelain/chimera/chimera-gateway/internal/gatewaymetrics"
	"github.com/lynn/porcelain/chimera/chimera-gateway/internal/harness/tools"
	"github.com/lynn/porcelain/chimera/chimera-gateway/internal/mcpmgr"
	"github.com/lynn/porcelain/chimera/chimera-gateway/internal/operatorstore"
	"github.com/lynn/porcelain/chimera/chimera-gateway/internal/rag"
	"github.com/lynn/porcelain/chimera/chimera-gateway/internal/transform"
	"github.com/lynn/porcelain/chimera/chimera-gateway/internal/vectorstore"
	"github.com/lynn/porcelain/chimera/internal/config"
	"github.com/lynn/porcelain/chimera/internal/providerlimits"
)

// VMStack is the resolved assistant routing configuration for one chat turn.
type VMStack struct {
	VM           *assistant.Resolved
	Fallback     []string
	ToolEnabled  bool
	RouterModels []string
	ToolThresh   float64
}

// TurnContext carries immutable request dependencies and mutable stage outputs for
// one assistant chat turn.
type TurnContext struct {
	W        http.ResponseWriter
	Resolved *config.Resolved
	Stack    VMStack

	Stream           bool
	SkipToolRouter   bool
	HeaderToolThresh float64

	RouteLog       *slog.Logger
	ConversationID string
	TurnIndex      int
	RequestID      string
	TenantID       string
	ProjectID      string
	FlavorID       string
	WorkspaceRowID int64
	APIKey         string
	Timeout        time.Duration

	ChatOpts *chat.ProxyOpts
	HistRec  *conversationhistory.Recorder

	RAG                 *rag.Service
	CorpusStale         rag.StaleSourceStore
	OperatorStore       *operatorstore.Store
	Metrics             gatewaymetrics.Recorder
	LimitsGuard         *providerlimits.Guard
	ModelAvailable      func(id string) bool
	OnToolRouterAttempt func(model string, err error)
	EmitRequestWitness  func(log *slog.Logger, res *config.Resolved, body map[string]json.RawMessage)

	// Mutable outputs populated by stages.
	RAGHits          []vectorstore.Hit
	InitialModel     string
	ToolRouter       transform.ToolRouterSummary
	Envelope         *TurnEnvelope
	WorkspaceScope   *operatorstore.Workspace
	WorkspaceRoots   []string
	ToolExecutor     tools.ToolExecutor
	ToolDeclarations []map[string]any
	MCPManager       *mcpmgr.Manager
	// BodyBeforeRetrieval preserves the request after transforms but before RAG
	// injection so escalation can re-retrieve without duplicating evidence.
	BodyBeforeRetrieval   Body
	RetrievalTopKOverride int
}

// AssistantID returns the client-facing assistant id when stack is loaded.
func (tc *TurnContext) AssistantID() string {
	if tc == nil || tc.Stack.VM == nil {
		return ""
	}
	return tc.Stack.VM.ModelID
}
