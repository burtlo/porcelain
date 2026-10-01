package harness

import (
	"context"
	"strings"

	"github.com/lynn/porcelain/chimera/chimera-gateway/internal/operatorstore"
	"github.com/lynn/porcelain/internal/naming"
)

// applyWorkspaceScope resolves workspace policy and wires the tool executor when a
// matching workspace exists. Returns true when tc.ToolExecutor was wired.
func applyWorkspaceScope(ctx context.Context, tc *TurnContext, env *TurnEnvelope) bool {
	if tc == nil || env == nil || tc.OperatorStore == nil || tc.ToolExecutor != nil {
		return tc != nil && tc.ToolExecutor != nil
	}
	workspace, matchedIDs, err := tc.OperatorStore.ResolveWorkspaceForHarness(ctx, tc.TenantID, tc.ProjectID, tc.FlavorID, tc.WorkspaceRowID)
	if err != nil {
		if tc.RouteLog != nil {
			tc.RouteLog.Warn("workspace scope resolution failed; proceeding without workspace policy",
				"err", err, "assistant_id", tc.AssistantID(), "turn_index", tc.TurnIndex,
				"workspace_row_id", tc.WorkspaceRowID,
				"timeline_kind", naming.TimelineKindBroker)
		}
		return false
	}
	if workspace == nil {
		return false
	}
	tc.WorkspaceScope = workspace
	tc.WorkspaceRoots = tc.WorkspaceRoots[:0]
	for _, path := range workspace.Paths {
		tc.WorkspaceRoots = append(tc.WorkspaceRoots, path.Path)
	}
	native := buildNativeToolExecutor(tc, tc.WorkspaceRoots, workspace.FileActionPolicy)
	wireToolExecutor(ctx, tc, native)
	env.Scope.WorkspaceID = &workspace.ID
	env.Scope.Sensitivity = workspace.Sensitivity
	env.Scope.AllowCloud = boolPtr(workspace.AllowCloud)
	env.Scope.WorkspacePermission = workspace.FileActionPolicy

	if len(matchedIDs) > 1 && tc.RouteLog != nil {
		tc.RouteLog.Info("multiple workspaces matched policy scope; selected lowest id",
			"msg", naming.MsgHarnessScopeWorkspaceAmbiguous,
			"matched_ids", matchedIDs, "chosen_id", workspace.ID,
			"assistant_id", tc.AssistantID(), "turn_index", tc.TurnIndex,
			"timeline_kind", naming.TimelineKindBroker)
	}
	if !workspace.AllowCloud {
		filterCloudFallback(tc)
	}
	return tc.ToolExecutor != nil
}

func filterCloudFallback(tc *TurnContext) {
	if tc == nil || len(tc.Stack.Fallback) == 0 {
		return
	}
	original := append([]string(nil), tc.Stack.Fallback...)
	filtered := make([]string, 0, len(original))
	for _, modelID := range original {
		if !looksCloudOnly(modelID) {
			filtered = append(filtered, modelID)
		}
	}
	if len(filtered) == 0 {
		if tc.RouteLog != nil {
			tc.RouteLog.Warn("workspace cloud policy removed every fallback candidate; restoring chain fail-open",
				"assistant_id", tc.AssistantID(), "turn_index", tc.TurnIndex,
				"timeline_kind", naming.TimelineKindBroker)
		}
		return
	}
	tc.Stack.Fallback = filtered
}

func looksCloudOnly(modelID string) bool {
	parts := strings.SplitN(strings.TrimSpace(modelID), "/", 2)
	return len(parts) == 2 && strings.TrimSpace(parts[0]) != "" && !strings.EqualFold(parts[0], "ollama")
}

func boolPtr(v bool) *bool { return &v }

func toolExecutorModuleEnabled(tc *TurnContext) bool {
	return tc != nil && tc.Stack.VM != nil &&
		tc.Stack.VM.HarnessEnabled(operatorstore.HarnessModuleToolExecutor)
}
