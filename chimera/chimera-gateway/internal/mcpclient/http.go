package mcpclient

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"sync"
	"sync/atomic"
)

// HTTPConfig configures streamable HTTP JSON-RPC (one POST per request).
type HTTPConfig struct {
	URL string
}

type httpTransport struct {
	url    string
	client *http.Client

	writeMu sync.Mutex
	nextID  uint64
}

// NewHTTPClient dials an MCP server over HTTP POST JSON-RPC.
func NewHTTPClient(ctx context.Context, cfg HTTPConfig) (*Client, error) {
	url := strings.TrimSpace(cfg.URL)
	if url == "" {
		return nil, fmt.Errorf("mcpclient: http url required")
	}
	tr := &httpTransport{
		url:    url,
		client: &http.Client{Timeout: 0},
	}
	return &Client{tr: tr, closeTr: tr.close}, nil
}

func (h *httpTransport) call(ctx context.Context, req *Request) (*Response, error) {
	req.JSONRPC = jsonRPCVersion
	if err := req.validate(); err != nil {
		return nil, err
	}
	if strings.HasPrefix(req.Method, "notifications/") {
		req.ID = nil
	}
	if len(req.ID) == 0 && !strings.HasPrefix(req.Method, "notifications/") {
		next := atomic.AddUint64(&h.nextID, 1)
		req.ID = json.RawMessage(fmt.Sprintf("%d", next))
	}
	raw, err := json.Marshal(req)
	if err != nil {
		return nil, err
	}
	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, h.url, bytes.NewReader(raw))
	if err != nil {
		return nil, err
	}
	httpReq.Header.Set("Content-Type", "application/json")
	resp, err := h.client.Do(httpReq)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(io.LimitReader(resp.Body, 8<<20))
	if err != nil {
		return nil, err
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, fmt.Errorf("mcpclient: http %d", resp.StatusCode)
	}
	if strings.HasPrefix(req.Method, "notifications/") {
		return nil, nil
	}
	var out Response
	if err := json.Unmarshal(body, &out); err != nil {
		return nil, fmt.Errorf("mcpclient: decode http response: %w", err)
	}
	return &out, nil
}

func (h *httpTransport) close() error { return nil }
