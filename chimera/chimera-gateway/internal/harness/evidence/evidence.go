// Package evidence compresses retrieval hits before they are injected upstream.
package evidence

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"sort"
	"strings"
	"time"

	"github.com/lynn/porcelain/chimera/chimera-gateway/internal/rag"
	"github.com/lynn/porcelain/chimera/chimera-gateway/internal/vectorstore"
)

type Config struct {
	Strategy         string
	SummarizeModelID string
	UpstreamBaseURL  string
	APIKey           string
	HTTPTimeout      time.Duration
}

type EvidenceBlock struct {
	Text string
	Hits []vectorstore.Hit
}

// Compressor turns retrieved hits into bounded upstream context.
type Compressor interface {
	Compress(context.Context, []vectorstore.Hit, int, Config) (EvidenceBlock, string, error)
}

func New(strategy string) Compressor {
	switch strings.ToLower(strings.TrimSpace(strategy)) {
	case "none":
		return noneCompressor{}
	case "summarize":
		return summarizeCompressor{}
	default:
		return truncateCompressor{}
	}
}

type noneCompressor struct{}

func (noneCompressor) Compress(_ context.Context, hits []vectorstore.Hit, _ int, _ Config) (EvidenceBlock, string, error) {
	return EvidenceBlock{Text: rag.FormatRetrievedContext(hits), Hits: append([]vectorstore.Hit(nil), hits...)}, "none", nil
}

type truncateCompressor struct{}

func (truncateCompressor) Compress(_ context.Context, hits []vectorstore.Hit, budget int, _ Config) (EvidenceBlock, string, error) {
	kept := append([]vectorstore.Hit(nil), hits...)
	sort.SliceStable(kept, func(i, j int) bool { return kept[i].Score > kept[j].Score })
	for len(kept) > 1 && len(rag.FormatRetrievedContext(kept)) > budget {
		kept = kept[:len(kept)-1]
	}
	text := rag.FormatRetrievedContext(kept)
	if budget > 0 && len(text) > budget {
		text = text[:budget]
	}
	return EvidenceBlock{Text: text, Hits: kept}, "truncate", nil
}

type summarizeCompressor struct{}

func (summarizeCompressor) Compress(ctx context.Context, hits []vectorstore.Hit, budget int, cfg Config) (EvidenceBlock, string, error) {
	if strings.TrimSpace(cfg.UpstreamBaseURL) == "" || strings.TrimSpace(cfg.SummarizeModelID) == "" {
		return EvidenceBlock{}, "summarize", fmt.Errorf("summarize upstream or model is not configured")
	}
	prompt := "Condense the retrieved context below. Preserve source names and only factual details useful to answer the user. Stay within the requested context budget.\n\n" + rag.FormatRetrievedContext(hits)
	payload := map[string]any{
		"model": cfg.SummarizeModelID,
		"messages": []map[string]string{
			{"role": "system", "content": "You compress retrieved evidence faithfully. Do not invent facts."},
			{"role": "user", "content": prompt},
		},
		"temperature": 0,
	}
	body, err := json.Marshal(payload)
	if err != nil {
		return EvidenceBlock{}, "summarize", err
	}
	timeout := cfg.HTTPTimeout
	if timeout <= 0 {
		timeout = 20 * time.Second
	}
	reqCtx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()
	req, err := http.NewRequestWithContext(reqCtx, http.MethodPost, strings.TrimRight(cfg.UpstreamBaseURL, "/")+"/v1/chat/completions", bytes.NewReader(body))
	if err != nil {
		return EvidenceBlock{}, "summarize", err
	}
	req.Header.Set("Content-Type", "application/json")
	if cfg.APIKey != "" {
		req.Header.Set("Authorization", "Bearer "+cfg.APIKey)
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return EvidenceBlock{}, "summarize", err
	}
	defer resp.Body.Close()
	raw, err := io.ReadAll(io.LimitReader(resp.Body, 2<<20))
	if err != nil {
		return EvidenceBlock{}, "summarize", err
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return EvidenceBlock{}, "summarize", fmt.Errorf("summarize completion status %d", resp.StatusCode)
	}
	var out struct {
		Choices []struct {
			Message struct {
				Content string `json:"content"`
			} `json:"message"`
		} `json:"choices"`
	}
	if err := json.Unmarshal(raw, &out); err != nil || len(out.Choices) == 0 {
		return EvidenceBlock{}, "summarize", fmt.Errorf("decode summarize completion: %w", err)
	}
	text := strings.TrimSpace(out.Choices[0].Message.Content)
	if text == "" {
		return EvidenceBlock{}, "summarize", fmt.Errorf("summarize completion was empty")
	}
	if budget > 0 && len(text) > budget {
		text = text[:budget]
	}
	return EvidenceBlock{Text: text, Hits: append([]vectorstore.Hit(nil), hits...)}, "summarize", nil
}
