package config

import (
	"fmt"
	"net"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"

	"github.com/lynn/porcelain/internal/naming"
)

// ServiceLocalModels is the supervisor.services entry that launches every enabled local_models entry.
const ServiceLocalModels = "local_models"

// LocalModel is one supervised llama-server chat model (one chimera-llm wrapper per model).
// The supervisor also registers it with the broker as a keyless provider named ProviderName().
type LocalModel struct {
	ID         string   // short id; provider becomes "local-<id>"
	Enabled    bool     // false keeps the entry configured but not launched
	Endpoint   string   // llama-server host:port
	Listen     string   // chimera-llm wrapper host:port (/healthz, /readyz, /status)
	ModelPath  string   // GGUF path (resolved against the chimera.yaml directory)
	MmprojPath string   // optional vision projector GGUF
	Alias      string   // model name served by llama-server and used after "local-<id>/"
	CtxSize    int      // -c
	NGPULayers int      // -ngl
	NCPUMoE    int      // --n-cpu-moe (MoE expert layers kept on CPU); 0 = all on GPU
	Parallel   int      // --parallel
	Thinking   bool     // false (default) disables Qwen-style thinking via template kwargs + reasoning budget 0
	ExtraArgs  []string // passed through to llama-server verbatim (sampling, cache types, batch sizes, ...)
	CacheDir   string   // llama-server working directory
	WaitReady  bool     // supervisor waits for /readyz before starting the broker
	StartupSec int      // readiness timeout in seconds
}

// LocalModels is the resolved local_models: block.
type LocalModels struct {
	LlamaServerBin string // optional override of the llama-server binary for chat models
	Models         []LocalModel
}

type localModelsDoc struct {
	LlamaServerBin string          `yaml:"llama_server_bin"`
	ModelsDir      string          `yaml:"models_dir"`
	Models         []localModelDoc `yaml:"models"`
}

type localModelDoc struct {
	ID         string   `yaml:"id"`
	Enabled    *bool    `yaml:"enabled"`
	Endpoint   string   `yaml:"endpoint"`
	Listen     string   `yaml:"listen"`
	Model      string   `yaml:"model"`
	Mmproj     string   `yaml:"mmproj"`
	Alias      string   `yaml:"alias"`
	CtxSize    int      `yaml:"ctx_size"`
	NGPULayers *int     `yaml:"n_gpu_layers"`
	NCPUMoE    int      `yaml:"n_cpu_moe"`
	Parallel   int      `yaml:"parallel"`
	Thinking   bool     `yaml:"thinking"`
	ExtraArgs  []string `yaml:"extra_args"`
	Wait       *bool    `yaml:"wait"`
	StartupSec int      `yaml:"startup_timeout_seconds"`
}

var localModelIDRe = regexp.MustCompile(`^[a-z0-9][a-z0-9-]{0,31}$`)

func (d localModelsDoc) effective(configDir string) LocalModels {
	out := LocalModels{LlamaServerBin: resolveOptionalPath(configDir, d.LlamaServerBin)}
	modelsDir := strings.TrimSpace(d.ModelsDir)
	if modelsDir != "" && !filepath.IsAbs(modelsDir) {
		modelsDir = filepath.Join(configDir, modelsDir)
	}
	for i, m := range d.Models {
		lm := LocalModel{
			ID:         strings.ToLower(strings.TrimSpace(m.ID)),
			Enabled:    m.Enabled == nil || *m.Enabled,
			Endpoint:   strings.TrimSpace(m.Endpoint),
			Listen:     strings.TrimSpace(m.Listen),
			Alias:      strings.TrimSpace(m.Alias),
			CtxSize:    m.CtxSize,
			NGPULayers: naming.DefaultLLMNGPULayers,
			NCPUMoE:    m.NCPUMoE,
			Parallel:   m.Parallel,
			Thinking:   m.Thinking,
			ExtraArgs:  append([]string(nil), m.ExtraArgs...),
			WaitReady:  m.Wait == nil || *m.Wait,
			StartupSec: m.StartupSec,
		}
		if m.NGPULayers != nil {
			lm.NGPULayers = *m.NGPULayers
		}
		lm.ModelPath = resolveModelPath(configDir, modelsDir, m.Model)
		lm.MmprojPath = resolveModelPath(configDir, modelsDir, m.Mmproj)
		if lm.Listen == "" {
			lm.Listen = net.JoinHostPort(naming.DefaultLLMListenHost, strconv.Itoa(naming.DefaultLLMListenBasePort+i))
		}
		if lm.Alias == "" {
			lm.Alias = lm.ID
		}
		if lm.CtxSize <= 0 {
			lm.CtxSize = naming.DefaultLLMCtxSize
		}
		if lm.Parallel <= 0 {
			lm.Parallel = naming.DefaultLLMParallel
		}
		if lm.StartupSec <= 0 {
			lm.StartupSec = 300
		}
		lm.CacheDir = filepath.Clean(filepath.Join(configDir, "..", naming.DefaultLLMDataPath, lm.ID))
		out.Models = append(out.Models, lm)
	}
	return out
}

func resolveModelPath(configDir, modelsDir, p string) string {
	p = strings.TrimSpace(p)
	if p == "" {
		return ""
	}
	if filepath.IsAbs(p) {
		return filepath.Clean(p)
	}
	if modelsDir != "" {
		return filepath.Clean(filepath.Join(modelsDir, p))
	}
	return filepath.Clean(filepath.Join(configDir, p))
}

func resolveOptionalPath(configDir, p string) string {
	p = strings.TrimSpace(p)
	if p == "" || filepath.IsAbs(p) || !strings.ContainsAny(p, `/\`) {
		return p // empty, absolute, or a bare command name resolved on PATH
	}
	return filepath.Clean(filepath.Join(configDir, p))
}

// ProviderName is the BiFrost provider key for this model ("local-<id>").
func (m LocalModel) ProviderName() string { return naming.LocalModelProviderPrefix + m.ID }

// ModelRef is the fully qualified id clients send ("local-<id>/<alias>").
func (m LocalModel) ModelRef() string { return m.ProviderName() + "/" + m.Alias }

// Enabled returns the models that should be launched.
func (lm LocalModels) Enabled() []LocalModel {
	out := make([]LocalModel, 0, len(lm.Models))
	for _, m := range lm.Models {
		if m.Enabled {
			out = append(out, m)
		}
	}
	return out
}

// Validate checks ids, addresses and uniqueness without requiring model files to exist yet.
func (lm LocalModels) Validate() error {
	ids, ports := map[string]bool{}, map[string]string{}
	for i, m := range lm.Models {
		where := fmt.Sprintf("local_models.models[%d]", i)
		if !localModelIDRe.MatchString(m.ID) {
			return fmt.Errorf("%s.id %q must be 1-32 chars of a-z, 0-9 or '-'", where, m.ID)
		}
		if ids[m.ID] {
			return fmt.Errorf("%s.id %q is duplicated", where, m.ID)
		}
		ids[m.ID] = true
		if strings.TrimSpace(m.ModelPath) == "" {
			return fmt.Errorf("%s (%s): model is required", where, m.ID)
		}
		for _, addr := range []struct{ name, v string }{{"endpoint", m.Endpoint}, {"listen", m.Listen}} {
			host, port, err := net.SplitHostPort(addr.v)
			if err != nil || host == "" || port == "" {
				return fmt.Errorf("%s (%s): %s must be host:port, got %q", where, m.ID, addr.name, addr.v)
			}
			if n, err := strconv.Atoi(port); err != nil || n <= 0 || n > 65535 {
				return fmt.Errorf("%s (%s): %s port invalid in %q", where, m.ID, addr.name, addr.v)
			}
			if !m.Enabled {
				continue
			}
			if prev, dup := ports[addr.v]; dup {
				return fmt.Errorf("%s (%s): %s %q already used by %s", where, m.ID, addr.name, addr.v, prev)
			}
			ports[addr.v] = m.ID + "." + addr.name
		}
		if m.NCPUMoE < 0 || m.NGPULayers < 0 {
			return fmt.Errorf("%s (%s): n_cpu_moe and n_gpu_layers must be >= 0", where, m.ID)
		}
	}
	return nil
}
