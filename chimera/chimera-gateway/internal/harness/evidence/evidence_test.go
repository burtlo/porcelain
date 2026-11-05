package evidence

import (
	"context"
	"strings"
	"testing"

	"github.com/lynn/porcelain/chimera/chimera-gateway/internal/vectorstore"
)

func TestTruncateDropsLowestScoresUntilWithinBudget(t *testing.T) {
	hits := []vectorstore.Hit{
		{ID: "high", Score: 0.9, Payload: vectorstore.Payload{Source: "high.md", Text: strings.Repeat("a", 200)}},
		{ID: "low", Score: 0.1, Payload: vectorstore.Payload{Source: "low.md", Text: strings.Repeat("b", 200)}},
	}
	full, _, err := New("none").Compress(context.Background(), hits, 0, Config{})
	if err != nil {
		t.Fatal(err)
	}
	block, strategy, err := New("truncate").Compress(context.Background(), hits, len(full.Text)-100, Config{})
	if err != nil {
		t.Fatal(err)
	}
	if strategy != "truncate" || len(block.Hits) != 1 || block.Hits[0].ID != "high" {
		t.Fatalf("expected only highest hit, got strategy=%q hits=%+v", strategy, block.Hits)
	}
}
