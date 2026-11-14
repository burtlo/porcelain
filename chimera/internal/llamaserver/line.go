package llamaserver

import (
	"encoding/json"
	"io"
	"strings"

	wline "github.com/lynn/porcelain/chimera/internal/wrapper/line"
)

type normalized struct {
	Timestamp   string `json:"timestamp,omitempty"`
	Level       string `json:"level,omitempty"`
	Service     string `json:"service"`
	Msg         string `json:"msg"`
	Detail      string `json:"detail,omitempty"`
	ChimeraNorm int    `json:"_chimera_norm,omitempty"`
}

// LineNormalizer turns llama-server and wrapper output into Chimera structured log lines
// for one wrapper product (chimera-embed or chimera-llm).
type LineNormalizer struct {
	Service   string // wrapper product name, e.g. chimera-embed
	MsgPrefix string // wrapper slog messages with this prefix (e.g. "embed.") pass through as-is
}

func (n LineNormalizer) NormalizePayload(raw string) []byte {
	return wline.NormalizePerLine(raw, n.alreadyNormalized, n.normalizePlain, n.normalizeJSON)
}

// NewWriter wraps dst so each written line is normalized.
func (n LineNormalizer) NewWriter(dst io.Writer) io.Writer {
	return wline.NewWriter(dst, n.NormalizePayload)
}

func (n LineNormalizer) alreadyNormalized(raw []byte) ([]byte, bool) {
	if reordered, ok := wline.ReorderNormalizedJSON(raw); ok {
		return reordered, true
	}
	return wline.NormalizeSlogLine(raw, n.Service)
}

func (n LineNormalizer) normalizeJSON(raw string) []byte {
	var fields map[string]json.RawMessage
	if err := json.Unmarshal([]byte(raw), &fields); err != nil {
		return n.normalizePlain(raw)
	}
	msg := strings.TrimSpace(wline.JSONString(fields, "msg"))
	if msg == "" {
		msg = strings.TrimSpace(wline.JSONString(fields, "message"))
	}
	if !strings.HasPrefix(msg, n.MsgPrefix) {
		return n.normalizePlain(raw)
	}
	level := strings.ToUpper(strings.TrimSpace(wline.JSONString(fields, "level")))
	if level == "" {
		level = "INFO"
	}
	timestamp := wline.JSONString(fields, "time")
	if timestamp == "" {
		timestamp = wline.JSONString(fields, "timestamp")
	}
	rec := normalized{
		Timestamp:   wline.NormalizeTimestampUTC(timestamp),
		Level:       level,
		Service:     n.Service,
		Msg:         msg,
		Detail:      wline.TrimRunes(wline.UpstreamDetailFromFields(fields), 2048),
		ChimeraNorm: wline.ChimeraNormValue,
	}
	b, _ := json.Marshal(rec)
	return b
}

func (n LineNormalizer) normalizePlain(raw string) []byte {
	detail := strings.TrimSpace(raw)
	lower := strings.ToLower(detail)
	level := "INFO"
	switch {
	case strings.Contains(lower, "error"), strings.Contains(lower, "fatal"):
		level = "ERROR"
	case strings.Contains(lower, "warn"):
		level = "WARN"
	}
	msg := n.MsgPrefix + "upstream.line"
	if strings.Contains(lower, "listening") {
		msg = n.MsgPrefix + "llama_server.listening"
	}
	rec := normalized{
		Timestamp:   wline.UTCTimestampNow(),
		Level:       level,
		Service:     n.Service,
		Msg:         msg,
		Detail:      wline.TrimRunes(detail, 4096),
		ChimeraNorm: wline.ChimeraNormValue,
	}
	b, _ := json.Marshal(rec)
	return b
}
