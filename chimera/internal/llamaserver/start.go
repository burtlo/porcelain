package llamaserver

import (
	"context"
	"fmt"
	"io"
	"log/slog"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
)

// Mode selects how llama-server is launched.
type Mode string

const (
	// ModeEmbed serves /v1/embeddings only (chimera-embed). Zero value.
	ModeEmbed Mode = ""
	// ModeChat serves /v1/chat/completions with the model's Jinja template (chimera-llm).
	ModeChat Mode = "chat"
)

type Config struct {
	Mode       Mode
	Bin        string
	ModelPath  string
	CacheDir   string
	BindHost   string
	Port       int
	CtxSize    int
	NGPULayers int
	Pooling    string // embed mode only
	Stdout     io.Writer
	Stderr     io.Writer

	// Chat mode only.
	MmprojPath string   // optional vision projector; empty → --no-mmproj
	Alias      string   // served model name
	NCPUMoE    int      // --n-cpu-moe; 0 = all experts on GPU
	Parallel   int      // --parallel
	Thinking   bool     // false → thinking disabled (template kwarg + reasoning budget 0)
	ExtraArgs  []string // appended verbatim (sampling, KV cache types, batch sizes, ...)
}

// Command resolves and validates the operator-supplied runtime and model.
func Command(cfg Config) (*exec.Cmd, error) {
	bin := strings.TrimSpace(cfg.Bin)
	if bin == "" {
		return nil, fmt.Errorf("llama-server: empty binary")
	}
	if strings.ContainsAny(bin, `/\`) && !filepath.IsAbs(bin) {
		var err error
		bin, err = filepath.Abs(bin)
		if err != nil {
			return nil, fmt.Errorf("resolve llama-server binary: %w", err)
		}
	}
	modelPath := strings.TrimSpace(cfg.ModelPath)
	if modelPath == "" {
		return nil, fmt.Errorf("llama-server: empty model path")
	}
	var err error
	modelPath, err = filepath.Abs(modelPath)
	if err != nil {
		return nil, fmt.Errorf("resolve model path: %w", err)
	}
	if info, statErr := os.Stat(modelPath); statErr != nil {
		return nil, fmt.Errorf("llama-server model missing at %s: %w", modelPath, statErr)
	} else if info.IsDir() {
		return nil, fmt.Errorf("llama-server model path is not a file: %s", modelPath)
	}

	ctxSize := cfg.CtxSize
	if ctxSize <= 0 {
		ctxSize = 2048
	}
	var args []string
	switch cfg.Mode {
	case ModeEmbed:
		pooling := strings.TrimSpace(cfg.Pooling)
		if pooling == "" {
			pooling = "mean"
		}
		args = []string{
			"-m", modelPath,
			"--embedding",
			"--host", strings.TrimSpace(cfg.BindHost),
			"--port", strconv.Itoa(cfg.Port),
			"-c", strconv.Itoa(ctxSize),
			"--n-gpu-layers", strconv.Itoa(cfg.NGPULayers),
			"--pooling", pooling,
			"--metrics",
		}
	case ModeChat:
		args, err = chatArgs(cfg, modelPath, ctxSize)
		if err != nil {
			return nil, err
		}
	default:
		return nil, fmt.Errorf("llama-server: unknown mode %q", cfg.Mode)
	}
	cmd := exec.Command(bin, args...)
	cacheDir := strings.TrimSpace(cfg.CacheDir)
	if cacheDir != "" {
		cacheDir, err = filepath.Abs(cacheDir)
		if err != nil {
			return nil, fmt.Errorf("resolve cache dir: %w", err)
		}
		if err := os.MkdirAll(cacheDir, 0o755); err != nil {
			return nil, fmt.Errorf("create llama-server cache dir: %w", err)
		}
		cmd.Dir = cacheDir
	}
	return cmd, nil
}

func Start(ctx context.Context, cfg Config, log *slog.Logger) (*exec.Cmd, error) {
	_ = ctx // shared wrapper runtime owns graceful termination.
	cmd, err := Command(cfg)
	if err != nil {
		return nil, err
	}
	cmd.Stdout = cfg.Stdout
	if cmd.Stdout == nil {
		cmd.Stdout = os.Stdout
	}
	cmd.Stderr = cfg.Stderr
	if cmd.Stderr == nil {
		cmd.Stderr = os.Stderr
	}
	applyNoConsoleWindow(cmd)
	if log != nil {
		msg, text := "embed.llama_server.starting", "starting llama-server embedding backend"
		if cfg.Mode == ModeChat {
			msg, text = "llm.llama_server.starting", "starting llama-server chat backend"
		}
		log.Info(text, "msg", msg, "bin", cmd.Path, "model", cfg.ModelPath, "alias", cfg.Alias, "host", cfg.BindHost, "port", cfg.Port)
	}
	if err := cmd.Start(); err != nil {
		return nil, fmt.Errorf("start llama-server: %w", err)
	}
	return cmd, nil
}

// chatArgs builds the llama-server command line for a chat model.
// Thinking is disabled unless cfg.Thinking is set: Qwen-style models otherwise
// spend most of their tokens reasoning (see docs/plans/local-inference-services.md).
func chatArgs(cfg Config, modelPath string, ctxSize int) ([]string, error) {
	args := []string{
		"-m", modelPath,
		"--host", strings.TrimSpace(cfg.BindHost),
		"--port", strconv.Itoa(cfg.Port),
		"-c", strconv.Itoa(ctxSize),
		"--n-gpu-layers", strconv.Itoa(cfg.NGPULayers),
		"--jinja",
		"--metrics",
	}
	if alias := strings.TrimSpace(cfg.Alias); alias != "" {
		args = append(args, "--alias", alias)
	}
	if cfg.NCPUMoE > 0 {
		args = append(args, "--n-cpu-moe", strconv.Itoa(cfg.NCPUMoE))
	}
	parallel := cfg.Parallel
	if parallel <= 0 {
		parallel = 1
	}
	args = append(args, "--parallel", strconv.Itoa(parallel))
	if mm := strings.TrimSpace(cfg.MmprojPath); mm != "" {
		abs, err := filepath.Abs(mm)
		if err != nil {
			return nil, fmt.Errorf("resolve mmproj path: %w", err)
		}
		if _, err := os.Stat(abs); err != nil {
			return nil, fmt.Errorf("llama-server mmproj missing at %s: %w", abs, err)
		}
		args = append(args, "--mmproj", abs)
	} else {
		// Vision-capable GGUFs otherwise try to auto-load a projector and can hang at startup.
		args = append(args, "--no-mmproj")
	}
	if cfg.Thinking {
		args = append(args, "--reasoning-budget", "-1", "--chat-template-kwargs", `{"enable_thinking":true}`)
	} else {
		args = append(args, "--reasoning-budget", "0", "--chat-template-kwargs", `{"enable_thinking":false}`)
	}
	for _, a := range cfg.ExtraArgs {
		if a = strings.TrimSpace(a); a != "" {
			args = append(args, a)
		}
	}
	return args, nil
}
