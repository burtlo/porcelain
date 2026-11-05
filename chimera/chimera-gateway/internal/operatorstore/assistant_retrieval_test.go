package operatorstore

import "testing"

func TestParseRetrievalConfigAppliesDefaultsAndNormalizes(t *testing.T) {
	cfg := ParseRetrievalConfig(`{"top_k":2,"compress_strategy":"summarize","skip_if":[" no_workspace "]}`, 8, 0.72)
	if cfg.TopK != 2 || cfg.ScoreFloor != 0.72 || cfg.MaxContextChars != DefaultRetrievalMaxContextChars {
		t.Fatalf("unexpected config: %#v", cfg)
	}
	if cfg.CompressStrategy != "summarize" || len(cfg.SkipIf) != 1 || cfg.SkipIf[0] != "no_workspace" {
		t.Fatalf("unexpected normalized config: %#v", cfg)
	}
}
