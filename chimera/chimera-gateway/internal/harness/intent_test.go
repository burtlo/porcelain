package harness

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/lynn/porcelain/chimera/chimera-gateway/internal/assistant"
	"github.com/lynn/porcelain/chimera/chimera-gateway/internal/operatorstore"
)

func TestIntentStageClassifiesBeforeRetrieval(t *testing.T) {
	enabled := true
	env := newEnvelope(&TurnContext{
		ProjectID: "repo",
		Stack: VMStack{VM: &assistant.Resolved{
			HarnessModules: map[string]assistant.HarnessModule{
				operatorstore.HarnessModuleIntent: {Enabled: enabled},
			},
		}},
	})
	body := Body{
		"messages": json.RawMessage(`[{"role":"user","content":"Please fix this code bug and add tests."}]`),
		"tools":    json.RawMessage(`[{"type":"function"},{"type":"function"}]`),
	}
	tc := &TurnContext{Stack: VMStack{VM: &assistant.Resolved{
		HarnessModules: map[string]assistant.HarnessModule{
			operatorstore.HarnessModuleIntent: {Enabled: true},
		},
	}}}
	if err := (IntentStage{}).Run(context.Background(), tc, env, body); err != nil {
		t.Fatal(err)
	}
	if env.Intent.TaskType != "code" || env.Intent.Complexity != "medium" {
		t.Fatalf("intent = %#v", env.Intent)
	}
	if env.Intent.RequiresRAG == nil || !*env.Intent.RequiresRAG {
		t.Fatalf("requires_rag = %#v", env.Intent.RequiresRAG)
	}
	if !shouldSkipRetrieval([]string{"code"}, tc, env) {
		t.Fatalf("intent tag should drive retrieval skip")
	}
}

func TestIntentStageLeavesDefaultsWhenDisabled(t *testing.T) {
	env := newEnvelope(&TurnContext{})
	tc := &TurnContext{Stack: VMStack{VM: &assistant.Resolved{
		HarnessModules: map[string]assistant.HarnessModule{
			operatorstore.HarnessModuleIntent: {Enabled: false},
		},
	}}}
	if err := (IntentStage{}).Run(context.Background(), tc, env, Body{
		"messages": json.RawMessage(`[{"role":"user","content":"write code"}]`),
	}); err != nil {
		t.Fatal(err)
	}
	if env.Intent.TaskType != "unknown" {
		t.Fatalf("intent = %#v", env.Intent)
	}
}
