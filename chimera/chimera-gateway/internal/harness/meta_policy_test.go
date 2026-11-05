package harness_test

import (
	"context"
	"path/filepath"
	"testing"

	"github.com/lynn/porcelain/chimera/chimera-gateway/internal/assistant"
	"github.com/lynn/porcelain/chimera/chimera-gateway/internal/harness"
	"github.com/lynn/porcelain/chimera/chimera-gateway/internal/operatorstore"
	"github.com/lynn/porcelain/chimera/chimera-gateway/internal/testsupport"
)

func TestMetaPolicyStageAppliesWorkspaceScopeAndCloudVeto(t *testing.T) {
	store, err := operatorstore.Open(filepath.Join(t.TempDir(), "operator.sqlite"), testsupport.GatewayOperatorMigrationsDir(t), nil)
	if err != nil {
		t.Fatal(err)
	}
	defer store.Close()
	workspace, err := store.CreateWorkspace(context.Background(), "tenant-a", "project-a", "flavor-a", []string{"/tmp/a"}, operatorstore.WorkspacePolicy{
		Sensitivity: "private", AllowCloud: false, AllowCloudSummaryOnly: true, FileActionPolicy: "read",
	})
	if err != nil {
		t.Fatal(err)
	}
	tc := &harness.TurnContext{
		OperatorStore: store,
		TenantID:      "tenant-a",
		ProjectID:     "project-a",
		FlavorID:      "flavor-a",
		Stack: harness.VMStack{
			VM:       &assistant.Resolved{ModelID: "Policy-1.0"},
			Fallback: []string{"groq/cloud", "ollama/local"},
		},
	}
	env := &harness.TurnEnvelope{}
	if err := (harness.MetaPolicyStage{}).Run(context.Background(), tc, env, harness.Body{}); err != nil {
		t.Fatal(err)
	}
	if env.Scope.WorkspaceID == nil || *env.Scope.WorkspaceID != workspace.ID || env.Scope.Sensitivity != "private" || env.Scope.AllowCloud == nil || *env.Scope.AllowCloud {
		t.Fatalf("scope=%+v", env.Scope)
	}
	if env.Scope.WorkspacePermission != "read" || tc.WorkspaceScope == nil {
		t.Fatalf("permission=%q workspace=%+v", env.Scope.WorkspacePermission, tc.WorkspaceScope)
	}
	if len(tc.Stack.Fallback) != 1 || tc.Stack.Fallback[0] != "ollama/local" {
		t.Fatalf("fallback=%v", tc.Stack.Fallback)
	}
}
