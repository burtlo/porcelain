package config

import (
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/lynn/porcelain/chimera/internal/wrapper/contract"
	wruntime "github.com/lynn/porcelain/chimera/internal/wrapper/runtime"
	"github.com/lynn/porcelain/internal/naming"
)

type BuildInfo struct {
	Version string
	Commit  string
	Date    string
}

// Config is one chimera-llm wrapper instance (one local chat model).
type Config struct {
	Listen                 string
	Bin                    string
	Backend                string
	Endpoint               string
	ModelID                string
	ModelPath              string
	MmprojPath             string
	Alias                  string
	CacheDir               string
	CtxSize                int
	NGPULayers             int
	NCPUMoE                int
	Parallel               int
	Thinking               bool
	ExtraArgs              []string
	extraArgsJSON          string
	StartupTimeout         time.Duration
	ShutdownTimeout        time.Duration
	TerminateWait          time.Duration
	BackoffInitial         time.Duration
	BackoffMultiplier      float64
	BackoffMax             time.Duration
	BackoffResetAfter      time.Duration
	DebugEnableUpstream    bool
	DebugAllowRemote       bool
	ForwardUpstreamInDebug bool
	UpstreamVersion        string
}

func PrintHelp() {
	fmt.Printf(`Chimera local chat model runtime (one llama-server per instance)

Usage:
  %s [flags]
  %s -version

Flags:
`, naming.ProductLLMName, naming.ProductLLMName)
	fs := flag.NewFlagSet(naming.ProductLLMName, flag.ContinueOnError)
	fs.SetOutput(os.Stdout)
	bindFlags(fs, &Config{})
	fs.PrintDefaults()
}

func Parse(args []string, build BuildInfo) (Config, error) {
	fs := flag.NewFlagSet(naming.ProductLLMName, flag.ContinueOnError)
	fs.SetOutput(io.Discard)
	cfg := Config{}
	var showVersion bool
	bindFlags(fs, &cfg)
	fs.BoolVar(&showVersion, "version", false, "print version")
	fs.BoolVar(&showVersion, "v", false, "print version")
	if err := fs.Parse(args); err != nil {
		return cfg, err
	}
	if showVersion {
		fmt.Printf("%s %s\ncommit %s\nbuild date %s\n", naming.ProductLLMName, build.Version, build.Commit, build.Date)
		return cfg, io.EOF
	}
	if strings.TrimSpace(strings.ToLower(cfg.Backend)) != naming.ProductLlamaServerBinName {
		return cfg, fmt.Errorf("%s must be %s for binary mode", naming.EnvLLMBackend, naming.ProductLlamaServerBinName)
	}
	if strings.TrimSpace(cfg.ModelPath) == "" {
		return cfg, fmt.Errorf("model path is required (-model-path or %s)", naming.EnvLLMModelPath)
	}
	if raw := strings.TrimSpace(cfg.extraArgsJSON); raw != "" {
		if err := json.Unmarshal([]byte(raw), &cfg.ExtraArgs); err != nil {
			return cfg, fmt.Errorf("-extra-args must be a JSON array of strings: %w", err)
		}
	}
	if cfg.NCPUMoE < 0 || cfg.NGPULayers < 0 {
		return cfg, fmt.Errorf("-n-cpu-moe and -n-gpu-layers must be >= 0")
	}
	return cfg, nil
}

func bindFlags(fs *flag.FlagSet, cfg *Config) {
	fs.StringVar(&cfg.Listen, "listen", envOrDefault(naming.EnvLLMListen, fmt.Sprintf("%s:%d", naming.DefaultLLMListenHost, naming.DefaultLLMListenBasePort)), "wrapper listen addr (host:port)")
	fs.StringVar(&cfg.Bin, "bin", envOrDefault(naming.EnvLLMBin, naming.ProductLlamaServerBinName), "llama-server binary path")
	fs.StringVar(&cfg.Backend, "backend", envOrDefault(naming.EnvLLMBackend, naming.ProductLlamaServerBinName), "backend name")
	fs.StringVar(&cfg.Endpoint, "endpoint", envOrDefault(naming.EnvLLMEndpoint, ""), "llama-server endpoint host:port")
	fs.StringVar(&cfg.ModelID, "model-id", envOrDefault(naming.EnvLLMModelID, ""), "local_models id (for logs and status)")
	fs.StringVar(&cfg.ModelPath, "model-path", envOrDefault(naming.EnvLLMModelPath, ""), "GGUF chat model path")
	fs.StringVar(&cfg.MmprojPath, "mmproj-path", envOrDefault(naming.EnvLLMMmprojPath, ""), "optional vision projector GGUF")
	fs.StringVar(&cfg.Alias, "alias", envOrDefault(naming.EnvLLMAlias, ""), "model name served by llama-server")
	fs.StringVar(&cfg.CacheDir, "cache-dir", envOrDefault(naming.EnvLLMCacheDir, ""), "llama-server working directory")
	fs.IntVar(&cfg.CtxSize, "ctx-size", envInt(naming.EnvLLMCtxSize, naming.DefaultLLMCtxSize), "context size")
	fs.IntVar(&cfg.NGPULayers, "n-gpu-layers", envInt(naming.EnvLLMNGPULayers, naming.DefaultLLMNGPULayers), "GPU layers")
	fs.IntVar(&cfg.NCPUMoE, "n-cpu-moe", envInt(naming.EnvLLMNCPUMoE, 0), "MoE expert layers kept on CPU (0 = all on GPU)")
	fs.IntVar(&cfg.Parallel, "parallel", envInt(naming.EnvLLMParallel, naming.DefaultLLMParallel), "parallel request slots")
	fs.BoolVar(&cfg.Thinking, "thinking", wruntime.EnvBool(naming.EnvLLMThinking), "enable model thinking (default off)")
	fs.StringVar(&cfg.extraArgsJSON, "extra-args", envOrDefault(naming.EnvLLMExtraArgs, ""), "JSON array of extra llama-server arguments")
	fs.DurationVar(&cfg.StartupTimeout, "startup-timeout", envDuration(naming.EnvLLMTimeoutsStartup, 300*time.Second), "startup readiness timeout (large models load slowly)")
	fs.DurationVar(&cfg.ShutdownTimeout, "shutdown-timeout", envDuration(naming.EnvLLMTimeoutsShutdown, contract.DefaultShutdownTimeout), "wrapper graceful shutdown timeout")
	fs.DurationVar(&cfg.TerminateWait, "terminate-wait", contract.DefaultTerminateWait, "wait before force-killing llama-server")
	fs.DurationVar(&cfg.BackoffInitial, "backoff-initial", contract.DefaultBackoffInitial, "restart backoff initial delay")
	fs.Float64Var(&cfg.BackoffMultiplier, "backoff-multiplier", contract.DefaultBackoffMultiplier, "restart backoff multiplier")
	fs.DurationVar(&cfg.BackoffMax, "backoff-max", contract.DefaultBackoffMax, "restart backoff max delay")
	fs.DurationVar(&cfg.BackoffResetAfter, "backoff-reset-after", contract.DefaultBackoffResetAfter, "healthy runtime to reset backoff")
	fs.BoolVar(&cfg.DebugEnableUpstream, "debug-enable-llm-logs", wruntime.EnvBool(contract.DebugEnableEnvKey(contract.ComponentLLM)), "enable "+contract.DebugLLMLogsPath)
	fs.BoolVar(&cfg.DebugAllowRemote, "debug-allow-remote", wruntime.EnvBool(contract.DebugAllowRemoteEnv), "allow /debug/* on non-loopback bind")
	fs.BoolVar(&cfg.ForwardUpstreamInDebug, "debug-forward-upstream", false, "forward upstream lines to stderr in debug mode")
	fs.StringVar(&cfg.UpstreamVersion, "upstream-version", "", "optional llama-server version for status")
}

func ParseEndpoint(endpoint string) (string, int, error) {
	host, portText, ok := strings.Cut(strings.TrimSpace(endpoint), ":")
	if !ok || strings.TrimSpace(host) == "" || strings.TrimSpace(portText) == "" {
		return "", 0, fmt.Errorf("invalid endpoint %q, expected host:port", endpoint)
	}
	port, err := strconv.Atoi(strings.TrimSpace(portText))
	if err != nil || port <= 0 || port > 65535 {
		return "", 0, fmt.Errorf("invalid endpoint port in %q", endpoint)
	}
	return strings.TrimSpace(host), port, nil
}

func envOrDefault(key, def string) string {
	if v := strings.TrimSpace(os.Getenv(key)); v != "" {
		return v
	}
	return def
}

func envInt(key string, def int) int {
	if v := strings.TrimSpace(os.Getenv(key)); v != "" {
		if n, err := strconv.Atoi(v); err == nil {
			return n
		}
	}
	return def
}

func envDuration(key string, def time.Duration) time.Duration {
	if v := strings.TrimSpace(os.Getenv(key)); v != "" {
		if d, err := time.ParseDuration(v); err == nil {
			return d
		}
	}
	return def
}
