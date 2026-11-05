package operatorstore

import (
	"context"
	"strings"
	"testing"
)

func TestAppendTurn_persistsHarnessSummary(t *testing.T) {
	s := openTestStore(t)
	ctx := context.Background()
	principal := "p1"
	cid := "conv-harness-1"
	if err := s.EnsureConversation(ctx, principal, cid, "hello", ConversationWorkspaceSnapshot{}); err != nil {
		t.Fatal(err)
	}
	summary := `{"schema_version":1,"assistant_id":"VM-1.0"}`
	turnID, err := s.AppendTurn(ctx, principal, cid, AppendTurnInput{
		Role:               "assistant",
		Content:            "ok",
		SelectedModel:      "VM-1.0",
		ResolvedModel:      "groq/fast",
		HarnessSummaryJSON: summary,
	})
	if err != nil {
		t.Fatal(err)
	}
	tr, err := s.GetConversationTranscript(ctx, principal, cid)
	if err != nil {
		t.Fatal(err)
	}
	for _, turn := range tr.Turns {
		if turn.TurnID == turnID {
			if strings.TrimSpace(turn.HarnessSummaryJSON) != summary {
				t.Fatalf("harness_summary=%q", turn.HarnessSummaryJSON)
			}
			return
		}
	}
	t.Fatal("assistant turn not found")
}
