package operatorstore

import (
	"encoding/json"
	"strings"
)

const DefaultRetrievalMaxContextChars = 12000

// RetrievalConfig is the typed config persisted in the retrieval harness module.
// Zero-valued knobs inherit gateway RAG defaults supplied to ParseRetrievalConfig.
type RetrievalConfig struct {
	TopK             int      `json:"top_k"`
	ScoreFloor       float32  `json:"score_floor"`
	MaxContextChars  int      `json:"max_context_chars"`
	CompressStrategy string   `json:"compress_strategy"`
	SummarizeModelID string   `json:"summarize_model_id"`
	SkipIf           []string `json:"skip_if"`
}

// ParseRetrievalConfig parses a module config and applies safe defaults.
func ParseRetrievalConfig(raw string, defaultTopK int, defaultScoreFloor float32) RetrievalConfig {
	cfg := RetrievalConfig{
		TopK:             defaultTopK,
		ScoreFloor:       defaultScoreFloor,
		MaxContextChars:  DefaultRetrievalMaxContextChars,
		CompressStrategy: "truncate",
		SkipIf:           []string{},
	}
	if strings.TrimSpace(raw) != "" {
		_ = json.Unmarshal([]byte(raw), &cfg)
	}
	if cfg.TopK <= 0 {
		cfg.TopK = defaultTopK
	}
	if cfg.ScoreFloor <= 0 {
		cfg.ScoreFloor = defaultScoreFloor
	}
	if cfg.MaxContextChars <= 0 {
		cfg.MaxContextChars = DefaultRetrievalMaxContextChars
	}
	switch strings.ToLower(strings.TrimSpace(cfg.CompressStrategy)) {
	case "none", "truncate", "summarize":
		cfg.CompressStrategy = strings.ToLower(strings.TrimSpace(cfg.CompressStrategy))
	default:
		cfg.CompressStrategy = "truncate"
	}
	cfg.SummarizeModelID = strings.TrimSpace(cfg.SummarizeModelID)
	cfg.SkipIf = normalizedSkipIf(cfg.SkipIf)
	return cfg
}

func normalizedSkipIf(in []string) []string {
	out := make([]string, 0, len(in))
	for _, item := range in {
		item = strings.TrimSpace(strings.ToLower(item))
		if item != "" {
			out = append(out, item)
		}
	}
	return out
}
