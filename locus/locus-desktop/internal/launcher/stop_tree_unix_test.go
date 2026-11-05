//go:build !windows

package launcher

import (
	"fmt"
	"os/exec"
	"testing"
	"time"
)

func TestForceKillProcessTree_KillsDescendants(t *testing.T) {
	parent := exec.Command("sh", "-c", "sleep 120 & wait")
	if err := parent.Start(); err != nil {
		t.Fatalf("start parent: %v", err)
	}
	parentPID := parent.Process.Pid

	deadline := time.Now().Add(2 * time.Second)
	var kids []int
	for time.Now().Before(deadline) {
		var err error
		kids, err = descendantPIDs(parentPID)
		if err == nil && len(kids) >= 1 {
			break
		}
		time.Sleep(20 * time.Millisecond)
	}
	if len(kids) < 1 {
		_ = parent.Process.Kill()
		_ = parent.Wait()
		t.Fatal("expected at least one descendant before tree kill")
	}

	if err := forceKillProcessTree(parentPID); err != nil {
		t.Fatalf("forceKillProcessTree: %v", err)
	}
	done := make(chan error, 1)
	go func() { done <- parent.Wait() }()
	select {
	case <-done:
	case <-time.After(2 * time.Second):
		t.Fatal("parent still running after tree kill")
	}

	time.Sleep(50 * time.Millisecond)
	for _, pid := range kids {
		if exec.Command("kill", "-0", fmt.Sprintf("%d", pid)).Run() == nil {
			t.Fatalf("descendant pid %d still alive after tree kill", pid)
		}
	}
}
