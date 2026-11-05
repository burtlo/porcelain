package mcpmgr

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"strings"
	"sync"

	"github.com/lynn/porcelain/chimera/chimera-gateway/internal/mcpclient"
	"github.com/lynn/porcelain/chimera/internal/config"
)

// State is the lifecycle state of one MCP server client.
type State string

const (
	StateStopped  State = "stopped"
	StateStarting State = "starting"
	StateRunning  State = "running"
	StateError    State = "error"
)

// Manager owns MCP server transports for the gateway process.
type Manager struct {
	log *slog.Logger
	cfg config.MCP

	mu      sync.Mutex
	servers map[string]*managed
}

type managed struct {
	cfg   config.MCPServer
	state State
	err   error
	cli   *mcpclient.Client
}

// New constructs a manager from resolved MCP config.
func New(cfg config.MCP, log *slog.Logger) *Manager {
	return &Manager{
		log:     log,
		cfg:     cfg,
		servers: make(map[string]*managed),
	}
}

// Start launches auto_start servers that are not disabled.
func (m *Manager) Start(ctx context.Context) {
	if m == nil {
		return
	}
	for _, s := range m.cfg.Servers {
		if s.Disabled || !s.AutoStart {
			continue
		}
		_ = m.EnsureRunning(ctx, s.ID)
	}
}

// Stop tears down all running MCP clients (gateway shutdown).
func (m *Manager) Stop() {
	if m == nil {
		return
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	for id, ms := range m.servers {
		if ms.cli != nil {
			_ = ms.cli.Close()
		}
		ms.cli = nil
		ms.state = StateStopped
		ms.err = nil
		m.servers[id] = ms
	}
}

// State returns the current state for serverID.
func (m *Manager) State(serverID string) State {
	if m == nil {
		return StateStopped
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	ms := m.servers[strings.TrimSpace(serverID)]
	if ms == nil {
		return StateStopped
	}
	return ms.state
}

// Client returns a running client for serverID after ensuring the transport is up.
func (m *Manager) Client(ctx context.Context, serverID string) (*mcpclient.Client, error) {
	serverID = strings.TrimSpace(serverID)
	m.mu.Lock()
	ms := m.servers[serverID]
	if ms != nil && ms.cli != nil && ms.state == StateRunning {
		cli := ms.cli
		m.mu.Unlock()
		return cli, nil
	}
	m.mu.Unlock()
	if err := m.EnsureRunning(ctx, serverID); err != nil {
		return nil, err
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	ms = m.servers[serverID]
	if ms == nil || ms.cli == nil {
		return nil, ErrNotRunning
	}
	return ms.cli, nil
}

// EnsureRunning starts serverID when configured and not disabled.
func (m *Manager) EnsureRunning(ctx context.Context, serverID string) error {
	if m == nil {
		return fmt.Errorf("mcpmgr: nil manager")
	}
	serverID = strings.TrimSpace(serverID)
	rec := m.cfg.ServerByID(serverID)
	if rec == nil {
		return fmt.Errorf("mcpmgr: unknown server %q", serverID)
	}
	if rec.Disabled {
		return fmt.Errorf("mcpmgr: server %q disabled", serverID)
	}

	m.mu.Lock()
	ms := m.servers[serverID]
	if ms == nil {
		ms = &managed{cfg: *rec, state: StateStopped}
		m.servers[serverID] = ms
	}
	if ms.state == StateRunning && ms.cli != nil {
		m.mu.Unlock()
		return nil
	}
	ms.state = StateStarting
	ms.err = nil
	m.mu.Unlock()

	cli, err := m.startTransport(ctx, *rec)
	m.mu.Lock()
	defer m.mu.Unlock()
	ms = m.servers[serverID]
	if err != nil {
		ms.state = StateError
		ms.err = err
		ms.cli = nil
		return err
	}
	ms.cli = cli
	ms.state = StateRunning
	return nil
}

func (m *Manager) startTransport(ctx context.Context, rec config.MCPServer) (*mcpclient.Client, error) {
	switch strings.ToLower(rec.Transport) {
	case "stdio":
		env := allowlistedEnv(rec.EnvAllowlist)
		cli, err := mcpclient.NewStdioClient(ctx, mcpclient.StdioConfig{
			Command: rec.Command,
			Args:    rec.Args,
			Env:     env,
		})
		if err != nil {
			return nil, err
		}
		if err := cli.Handshake(ctx); err != nil {
			_ = cli.Close()
			return nil, err
		}
		return cli, nil
	case "http":
		cli, err := mcpclient.NewHTTPClient(ctx, mcpclient.HTTPConfig{URL: rec.URL})
		if err != nil {
			return nil, err
		}
		if err := cli.Handshake(ctx); err != nil {
			_ = cli.Close()
			return nil, err
		}
		return cli, nil
	default:
		return nil, fmt.Errorf("mcpmgr: unsupported transport %q", rec.Transport)
	}
}

func allowlistedEnv(allow []string) []string {
	if len(allow) == 0 {
		// Non-nil empty env: child must not inherit gateway/supervisor secrets.
		return []string{}
	}
	allowed := make(map[string]bool, len(allow))
	for _, k := range allow {
		k = strings.TrimSpace(k)
		if k != "" {
			allowed[k] = true
		}
	}
	var out []string
	for _, kv := range os.Environ() {
		key, _, _ := strings.Cut(kv, "=")
		if allowed[key] {
			out = append(out, kv)
		}
	}
	return out
}
