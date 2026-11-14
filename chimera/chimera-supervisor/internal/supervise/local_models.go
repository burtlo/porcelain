package supervise

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"os/exec"
	"strconv"
	"strings"
	"time"

	llmadapter "github.com/lynn/porcelain/chimera/chimera-llm/adapter"
	svconfig "github.com/lynn/porcelain/chimera/chimera-supervisor/internal/config"
	"github.com/lynn/porcelain/chimera/chimera-supervisor/internal/control"
	"github.com/lynn/porcelain/chimera/chimera-supervisor/internal/proc"
	gwconfig "github.com/lynn/porcelain/chimera/internal/config"
	"github.com/lynn/porcelain/chimera/internal/servicelogs"
)

// localModelChild is one supervised chimera-llm wrapper (one local chat model).
type localModelChild struct {
	model   gwconfig.LocalModel
	cmd     *exec.Cmd
	wait    chan error
	readyz  string
	started bool
}

// llamaServerBinForChat picks the llama-server binary for chat models:
// -llm-backend-bin, then local_models.llama_server_bin, then the embed backend default.
func llamaServerBinForChat(cfg svconfig.Config, res *gwconfig.Resolved) string {
	if b := strings.TrimSpace(cfg.LLMBackendBin); b != "" {
		return b
	}
	if res != nil {
		if b := strings.TrimSpace(res.LocalModels.LlamaServerBin); b != "" {
			return b
		}
	}
	return strings.TrimSpace(cfg.EmbedBackendBin)
}

// localModelArgs builds the chimera-llm command line for one model.
func localModelArgs(m gwconfig.LocalModel, llamaBin string) ([]string, error) {
	args := []string{
		"-listen", m.Listen,
		"-bin", llamaBin,
		"-endpoint", m.Endpoint,
		"-model-id", m.ID,
		"-model-path", m.ModelPath,
		"-alias", m.Alias,
		"-cache-dir", m.CacheDir,
		"-ctx-size", strconv.Itoa(m.CtxSize),
		"-n-gpu-layers", strconv.Itoa(m.NGPULayers),
		"-n-cpu-moe", strconv.Itoa(m.NCPUMoE),
		"-parallel", strconv.Itoa(m.Parallel),
		"-startup-timeout", (time.Duration(m.StartupSec) * time.Second).String(),
		"-thinking=" + strconv.FormatBool(m.Thinking),
	}
	if m.MmprojPath != "" {
		args = append(args, "-mmproj-path", m.MmprojPath)
	}
	if len(m.ExtraArgs) > 0 {
		b, err := json.Marshal(m.ExtraArgs)
		if err != nil {
			return nil, err
		}
		args = append(args, "-extra-args", string(b))
	}
	return WrapperArgs(args), nil
}

// startLocalModelChildren starts one chimera-llm per enabled local model, then waits (in parallel)
// for those that request it. Local models are optional: a model that fails to start or load is
// logged and reported on /status, but never aborts the supervised stack.
func startLocalModelChildren(cfg svconfig.Config, res *gwconfig.Resolved, controlBaseURL string, logStore *servicelogs.Store, logLevel slog.Level, log *slog.Logger, controlState *control.State) []*localModelChild {
	models := res.LocalModels.Enabled()
	if len(models) == 0 {
		return nil
	}
	llamaBin := llamaServerBinForChat(cfg, res)
	status := make([]control.LocalModelStatus, 0, len(models))
	children := make([]*localModelChild, 0, len(models))
	for _, m := range models {
		status = append(status, control.LocalModelStatus{ID: m.ID, Provider: m.ProviderName(), ModelRef: m.ModelRef(), Endpoint: m.Endpoint, Listen: m.Listen})
		children = append(children, &localModelChild{model: m, readyz: fmt.Sprintf("http://%s/readyz", m.Listen)})
	}
	controlState.SetLocalModels(status)

	for _, c := range children {
		args, err := localModelArgs(c.model, llamaBin)
		if err != nil {
			log.Error("local model args", "msg", "chimera-supervisor.local_model.config_error", "model_id", c.model.ID, "err", err)
			continue
		}
		cmd := exec.Command(strings.TrimSpace(cfg.LLMBin), args...)
		cmd.Env = mergeEnv(ChildEnv(controlBaseURL))
		proc.ApplyNoConsoleWindow(cmd)
		sink := LogSink(logStore.Writer(servicelogs.SourceChimeraLLM), llmadapter.ChildLogWriter, logLevel)
		cmd.Stdout = sink
		cmd.Stderr = sink
		if err := cmd.Start(); err != nil {
			controlState.SetLastError(err.Error())
			log.Error("start chimera-llm failed", "msg", "chimera-supervisor.local_model.start_failed", "model_id", c.model.ID, "bin", cfg.LLMBin, "err", err)
			continue
		}
		c.cmd, c.started = cmd, true
		c.wait = make(chan error, 1)
		go func(ch chan error) { ch <- cmd.Wait() }(c.wait)
		log.Info("chimera-llm started", "msg", "chimera-supervisor.local_model.started", "model_id", c.model.ID, "model", c.model.ModelRef(), "endpoint", c.model.Endpoint, "listen", c.model.Listen)
	}

	// Models load concurrently (they are separate processes); wait for the ones that asked for it.
	done := make(chan struct{}, len(children))
	waiting := 0
	for _, c := range children {
		if !c.started || !c.model.WaitReady || cfg.NoWaitLLM {
			continue
		}
		waiting++
		go func(c *localModelChild) {
			defer func() { done <- struct{}{} }()
			timeout := time.Duration(c.model.StartupSec)*time.Second + 15*time.Second
			ctx, cancel := context.WithTimeout(context.Background(), timeout)
			defer cancel()
			if err := waitHealthy(ctx, c.readyz, timeout, log, "chimera-llm"); err != nil {
				log.Error("local model not ready; continuing without it", "msg", "chimera-supervisor.local_model.not_ready", "model_id", c.model.ID, "err", err)
				return
			}
			controlState.SetLocalModelReady(c.model.ID, true)
		}(c)
	}
	for i := 0; i < waiting; i++ {
		<-done
	}
	return children
}

// monitorLocalModels keeps /status readiness current for every started local model.
func monitorLocalModels(ctx context.Context, children []*localModelChild, controlState *control.State, interval time.Duration) {
	client := &http.Client{Timeout: 3 * time.Second}
	for _, c := range children {
		if !c.started {
			continue
		}
		go func(c *localModelChild) {
			t := time.NewTicker(interval)
			defer t.Stop()
			for {
				ok := false
				if req, err := http.NewRequestWithContext(ctx, http.MethodGet, c.readyz, nil); err == nil {
					if res, err := client.Do(req); err == nil {
						res.Body.Close()
						ok = res.StatusCode >= 200 && res.StatusCode < 300
					}
				}
				controlState.SetLocalModelReady(c.model.ID, ok)
				select {
				case <-ctx.Done():
					return
				case <-t.C:
				}
			}
		}(c)
	}
}

// localModelShutdownChildren adapts started local models to the shared shutdown helpers.
func localModelShutdownChildren(children []*localModelChild) ([]Child, []*exec.Cmd) {
	var out []Child
	var cmds []*exec.Cmd
	for _, c := range children {
		if c == nil || !c.started {
			continue
		}
		out = append(out, Child{Name: "llm:" + c.model.ID, Cmd: c.cmd, WaitCh: c.wait})
		cmds = append(cmds, c.cmd)
	}
	return out, cmds
}
