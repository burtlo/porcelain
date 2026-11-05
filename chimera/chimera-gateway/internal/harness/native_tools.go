package harness

import (
	"github.com/lynn/porcelain/chimera/chimera-gateway/internal/harness/tools"
	"github.com/lynn/porcelain/chimera/chimera-gateway/internal/rag"
	"github.com/lynn/porcelain/chimera/chimera-gateway/internal/vectorstore"
)

func nativeExpansionEnabled(tc *TurnContext) bool {
	if tc == nil || tc.RAG == nil || tc.OperatorStore == nil || tc.Resolved == nil {
		return false
	}
	if !tc.Resolved.RAG.Enabled || !tc.Resolved.RAG.ToolingEnabled {
		return false
	}
	return true
}

func buildNativeToolExecutor(tc *TurnContext, roots []string, policy string) tools.ToolExecutor {
	ws := tools.NewWorkspaceExecutor(roots, policy)
	if !nativeExpansionEnabled(tc) {
		return ws
	}
	exp := rag.NewExpansionService(rag.ExpansionOptions{
		RAG:           tc.RAG,
		OperatorStore: tc.OperatorStore,
		StaleStore:    tc.CorpusStale,
		Resolved:      tc.Resolved,
	})
	return &tools.NativeExecutor{
		Workspace: ws,
		Expansion: exp,
		Coords: vectorstore.Coords{
			TenantID:  tc.TenantID,
			ProjectID: tc.ProjectID,
			FlavorID:  tc.FlavorID,
		},
	}
}
