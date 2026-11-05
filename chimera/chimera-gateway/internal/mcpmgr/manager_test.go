package mcpmgr

import (
	"context"
	"os"
	"path/filepath"
	"runtime"
	"syscall"
	"testing"
	"time"

	"github.com/lynn/porcelain/chimera/internal/config"
)

func TestManagerStdioFakeSubprocess(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("subprocess fake server test skipped on windows")
	}
	_, filename, _, _ := runtime.Caller(0)
	root := filepath.Clean(filepath.Join(filepath.Dir(filename), "..", "..", "..", ".."))
	fakeMain := filepath.Join(root, "chimera", "chimera-gateway", "internal", "mcpclient", "fake", "cmd", "main.go")

	cfg := config.MCP{
		Servers: []config.MCPServer{{
			ID:        "fake",
			Transport: "stdio",
			Command:   "go",
			Args:      []string{"run", fakeMain},
			EnvAllowlist: []string{
				"PATH", "HOME", "GOROOT", "GOPATH", "GOMOD", "GOWORK", "CGO_ENABLED",
			},
			AutoStart: true,
		}},
	}
	mgr := New(cfg, nil)
	mgr.Start(context.Background())
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()
	cli, err := mgr.Client(ctx, "fake")
	if err != nil {
		t.Fatal(err)
	}
	tools, err := cli.ListTools(ctx)
	if err != nil || len(tools) == 0 {
		t.Fatalf("list: %v tools=%v", err, tools)
	}
	mgr.Stop()
}

func TestManagerStopNoOrphans(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("subprocess fake server test skipped on windows")
	}
	_, filename, _, _ := runtime.Caller(0)
	root := filepath.Clean(filepath.Join(filepath.Dir(filename), "..", "..", "..", ".."))
	fakeMain := filepath.Join(root, "chimera", "chimera-gateway", "internal", "mcpclient", "fake", "cmd", "main.go")

	cfg := config.MCP{
		Servers: []config.MCPServer{{
			ID:        "fake",
			Transport: "stdio",
			Command:   "go",
			Args:      []string{"run", fakeMain},
			EnvAllowlist: []string{
				"PATH", "HOME", "GOROOT", "GOPATH", "GOMOD", "GOWORK", "CGO_ENABLED",
			},
		}},
	}
	mgr := New(cfg, nil)
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()
	if err := mgr.EnsureRunning(ctx, "fake"); err != nil {
		t.Fatal(err)
	}
	cli, err := mgr.Client(ctx, "fake")
	if err != nil {
		t.Fatal(err)
	}
	pid := cli.ChildPID()
	if pid <= 0 {
		t.Fatal("expected running child pid")
	}
	mgr.Stop()
	waitChildExit(t, pid, 10*time.Second)
}

func waitChildExit(t *testing.T, pid int, timeout time.Duration) {
	t.Helper()
	deadline := time.Now().Add(timeout)
	proc, err := os.FindProcess(pid)
	if err != nil {
		return
	}
	for time.Now().Before(deadline) {
		if err := proc.Signal(syscall.Signal(0)); err != nil {
			return
		}
		time.Sleep(50 * time.Millisecond)
	}
	t.Fatalf("stdio MCP child pid %d still running after manager Stop", pid)
}

func TestAllowlistedEnvDenyByDefault(t *testing.T) {
	const key = "CHIMERA_MCP_ENV_TEST_MARKER"
	t.Setenv(key, "must-not-leak")
	out := allowlistedEnv(nil)
	for _, kv := range out {
		if stringsBeforeEq(kv) == key {
			t.Fatalf("unexpected inherited env %q", kv)
		}
	}
}

func stringsBeforeEq(kv string) string {
	for i := 0; i < len(kv); i++ {
		if kv[i] == '=' {
			return kv[:i]
		}
	}
	return kv
}
