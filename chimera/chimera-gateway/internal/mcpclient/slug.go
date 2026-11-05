package mcpclient

import (
	"crypto/sha256"
	"encoding/hex"
	"strings"
	"unicode"
)

const maxOpenAINameLen = 64

// OpenAIName derives the model-visible function.name for an MCP tool.
// Algorithm: sanitize serverId and mcpToolName to [a-zA-Z0-9_], join with "__".
// If longer than 64 chars, replace the suffix with the first 16 hex chars of
// SHA-256(serverId + "\x00" + mcpToolName), trimming the left segment as needed.
func OpenAIName(serverID, mcpToolName string) string {
	left := sanitizeSlugSegment(serverID, "srv")
	right := sanitizeSlugSegment(mcpToolName, "tool")
	candidate := left + "__" + right
	if len(candidate) <= maxOpenAINameLen {
		return candidate
	}
	sum := sha256.Sum256([]byte(serverID + "\x00" + mcpToolName))
	digest := hex.EncodeToString(sum[:])[:16]
	sep := "__"
	maxLeft := maxOpenAINameLen - len(sep) - len(digest)
	if maxLeft < 1 {
		return digest[:maxOpenAINameLen]
	}
	if len(left) > maxLeft {
		left = left[:maxLeft]
	}
	return left + sep + digest
}

func sanitizeSlugSegment(s, fallback string) string {
	var b strings.Builder
	for _, r := range s {
		if unicode.IsLetter(r) || unicode.IsDigit(r) || r == '_' {
			b.WriteRune(r)
		} else {
			b.WriteByte('_')
		}
	}
	out := strings.Trim(b.String(), "_")
	if out == "" {
		return fallback
	}
	return out
}
