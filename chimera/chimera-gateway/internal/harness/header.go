package harness

import (
	"encoding/base64"
	"net/http"

	"github.com/lynn/porcelain/internal/naming"
)

// WriteSummaryHeader sets X-Chimera-Harness-Summary with base64-encoded redacted envelope JSON.
func WriteSummaryHeader(w http.ResponseWriter, env *TurnEnvelope) {
	if w == nil || env == nil {
		return
	}
	b, err := RedactedJSON(env)
	if err != nil || len(b) == 0 {
		return
	}
	w.Header().Set(naming.HeaderHarnessSummaryTarget, base64.StdEncoding.EncodeToString(b))
}
