package embedline

import (
	"io"

	"github.com/lynn/porcelain/chimera/internal/llamaserver"
	"github.com/lynn/porcelain/internal/naming"
)

var normalizer = llamaserver.LineNormalizer{Service: naming.ProductEmbedName, MsgPrefix: "embed."}

func NormalizePayload(raw string) []byte { return normalizer.NormalizePayload(raw) }

func NewWriter(dst io.Writer) io.Writer { return normalizer.NewWriter(dst) }
