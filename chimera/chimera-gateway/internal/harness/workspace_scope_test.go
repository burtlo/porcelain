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

func TestApplyWorkspaceScope_byWorkspaceRowID(t *testing.T) {
	store, err := operatorstore.Open(filepath.Join(t.TempDir(), "operator.sqlite"), testsupport.GatewayOperatorMigrationsDir(t), nil)
	if err != nil {
		t.Fatal(err)
	}
	defer store.Close()
	root := t.TempDir()
	ws, err := store.CreateWorkspace(context.Background(), "default", "porcelain", "", []string{root}, operatorstore.WorkspacePolicy{
		FileActionPolicy: "read",
	})
	if err != nil {
		t.Fatal(err)
	}
	tc := &harness.TurnContext{
		OperatorStore:  store,
		TenantID:       "default",
		ProjectID:      "default",
		WorkspaceRowID: ws.ID,
		Stack: harness.VMStack{
			VM: &assistant.Resolved{
				ModelID: "Eval-Porcelain-DevMCP-1.0",
				HarnessModules: map[string]assistant.HarnessModule{
					operatorstore.HarnessModuleToolExecutor: {Enabled: true, ConfigJSON: `{}`},
				},
			},
		},
	}
	env := &harness.TurnEnvelope{}
	if err := (harness.MetaPolicyStage{}).Run(context.Background(), tc, env, harness.Body{}); err != nil {
		t.Fatal(err)
	}
	if tc.ToolExecutor == nil || len(tc.WorkspaceRoots) != 1 {
		t.Fatalf("executor=%v roots=%v", tc.ToolExecutor, tc.WorkspaceRoots)
	}
}
