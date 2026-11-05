package harness

import (
	"context"
)

// MetaPolicyStage resolves the server-owned workspace scope before any
// LLM-assisted stage. A cloud route is identified conservatively as a
// provider/model id whose provider is not Ollama; bare ids stay eligible.
type MetaPolicyStage struct{}

func (MetaPolicyStage) Name() string   { return "meta_policy" }
func (MetaPolicyStage) Module() string { return "meta_policy" }

func (MetaPolicyStage) Run(ctx context.Context, tc *TurnContext, env *TurnEnvelope, _ Body) error {
	if tc == nil || env == nil || tc.OperatorStore == nil {
		return nil
	}
	applyWorkspaceScope(ctx, tc, env)
	return nil
}
