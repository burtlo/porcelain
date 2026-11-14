package adapter

import (
	"context"
	"fmt"
	"io"
	"log/slog"
	"os"
	"os/exec"
	"strings"

	"github.com/lynn/porcelain/chimera/chimera-llm/internal/config"
	"github.com/lynn/porcelain/chimera/internal/llamaserver"
	"github.com/lynn/porcelain/chimera/internal/wrapper/contract"
	"github.com/lynn/porcelain/internal/naming"
)

var lines = llamaserver.LineNormalizer{Service: naming.ProductLLMName, MsgPrefix: "llm."}

// LlamaServer adapts a llama-server chat model to the shared wrapper runtime.
type LlamaServer struct {
	Cfg  config.Config
	Host string
	Port int
}

func (a *LlamaServer) Start(ctx context.Context, capture io.Writer, log *slog.Logger) (*exec.Cmd, error) {
	return llamaserver.Start(ctx, llamaserver.Config{
		Mode:       llamaserver.ModeChat,
		Bin:        a.Cfg.Bin,
		ModelPath:  a.Cfg.ModelPath,
		MmprojPath: a.Cfg.MmprojPath,
		Alias:      a.Cfg.Alias,
		CacheDir:   a.Cfg.CacheDir,
		BindHost:   a.Host,
		Port:       a.Port,
		CtxSize:    a.Cfg.CtxSize,
		NGPULayers: a.Cfg.NGPULayers,
		NCPUMoE:    a.Cfg.NCPUMoE,
		Parallel:   a.Cfg.Parallel,
		Thinking:   a.Cfg.Thinking,
		ExtraArgs:  a.Cfg.ExtraArgs,
		Stdout:     lines.NewWriter(io.MultiWriter(capture, os.Stdout)),
		Stderr:     lines.NewWriter(io.MultiWriter(capture, os.Stderr)),
	}, log)
}

// ReadyURL is llama-server's /health: 503 while the model loads, 200 once it can serve.
func (a *LlamaServer) ReadyURL() string {
	return fmt.Sprintf("http://%s:%d/health", strings.TrimSpace(a.Host), a.Port)
}

func (a *LlamaServer) MetricsURL() string {
	return fmt.Sprintf("http://%s:%d%s", strings.TrimSpace(a.Host), a.Port, contract.MetricsPath)
}

func (a *LlamaServer) BackendName() string { return naming.ProductLlamaServerBinName }

// ChildLogWriter normalizes supervised chimera-llm output for operator logs.
func ChildLogWriter(dst io.Writer) io.Writer { return lines.NewWriter(dst) }

func WrapUpstreamLine(raw string) string { return string(lines.NormalizePayload(raw)) }
