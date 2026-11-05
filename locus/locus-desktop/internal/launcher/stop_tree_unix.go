//go:build !windows

package launcher

import (
	"bytes"
	"fmt"
	"os"
	"os/exec"
	"strconv"
	"strings"
)

// forceKillProcessTree terminates pid and all of its descendants on Unix.
// Used when an owned supervisor does not exit after POST /shutdown within the
// stop timeout — without a tree kill, wrapper children (especially broker) can
// be orphaned under launchd (ppid 1).
func forceKillProcessTree(pid int) error {
	if pid <= 0 {
		return fmt.Errorf("invalid pid %d", pid)
	}
	descendants, err := descendantPIDs(pid)
	if err != nil {
		return killPID(pid)
	}
	for i := len(descendants) - 1; i >= 0; i-- {
		_ = killPID(descendants[i])
	}
	return killPID(pid)
}

func killPID(pid int) error {
	if pid <= 0 || pid == os.Getpid() {
		return nil
	}
	p, err := os.FindProcess(pid)
	if err != nil {
		return nil
	}
	err = p.Kill()
	if err == nil || err == os.ErrProcessDone || strings.Contains(err.Error(), "process already finished") {
		return nil
	}
	return err
}

func descendantPIDs(root int) ([]int, error) {
	children, err := processChildrenByParent()
	if err != nil {
		return nil, err
	}
	var out []int
	queue := []int{root}
	seen := map[int]struct{}{root: {}}
	for len(queue) > 0 {
		parent := queue[0]
		queue = queue[1:]
		for _, child := range children[parent] {
			if _, ok := seen[child]; ok {
				continue
			}
			seen[child] = struct{}{}
			out = append(out, child)
			queue = append(queue, child)
		}
	}
	return out, nil
}

func processChildrenByParent() (map[int][]int, error) {
	cmd := exec.Command("ps", "-axo", "pid=,ppid=")
	out, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("ps: %w", err)
	}
	children := make(map[int][]int)
	for _, line := range bytes.Split(out, []byte("\n")) {
		line = bytes.TrimSpace(line)
		if len(line) == 0 {
			continue
		}
		fields := strings.Fields(string(line))
		if len(fields) < 2 {
			continue
		}
		pid, err1 := strconv.Atoi(fields[0])
		ppid, err2 := strconv.Atoi(fields[1])
		if err1 != nil || err2 != nil || pid <= 0 {
			continue
		}
		children[ppid] = append(children[ppid], pid)
	}
	return children, nil
}
