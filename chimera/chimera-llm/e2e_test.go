package main

import (
	"encoding/json"
	"fmt"
	"net"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"slices"
	"strings"
	"testing"
	"time"
)

func TestE2ELLMWrapperServesSupervisedChatModel(t *testing.T) {
	moduleRoot, err := findModuleRoot()
	if err != nil {
		t.Fatal(err)
	}
	buildDir := t.TempDir()
	ext := ""
	if runtime.GOOS == "windows" {
		ext = ".exe"
	}
	llmBin := filepath.Join(buildDir, "chimera-llm"+ext)
	fakeBin := filepath.Join(buildDir, "llama-server"+ext)
	buildForE2E(t, moduleRoot, llmBin, "./chimera/chimera-llm")
	buildForE2E(t, moduleRoot, fakeBin, "./chimera/chimera-embed/testdata/fakellamaserver")
	modelPath := filepath.Join(buildDir, "qwen.gguf")
	if err := os.WriteFile(modelPath, []byte("fake"), 0o644); err != nil {
		t.Fatal(err)
	}
	cacheDir := filepath.Join(buildDir, "cache")
	wrapperAddr := allocateAddr(t)
	backendAddr := allocateAddr(t)
	cmd := exec.Command(llmBin,
		"-listen", wrapperAddr,
		"-bin", fakeBin,
		"-endpoint", backendAddr,
		"-model-id", "qwen35b",
		"-model-path", modelPath,
		"-alias", "qwen3.6-35b",
		"-cache-dir", cacheDir,
		"-ctx-size", "65536",
		"-n-cpu-moe", "2",
		"-extra-args", `["-fa","on","-ctk","q8_0"]`,
		"-startup-timeout", "8s",
		"-terminate-wait", "1s",
	)
	cmd.Env = append(os.Environ(), "CHIMERA_LOG_JSON=1")
	if err := cmd.Start(); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { stopE2EProcess(cmd) })
	waitForStatus(t, "http://"+wrapperAddr+"/readyz", http.StatusOK, 10*time.Second)

	resp, err := http.Post("http://"+backendAddr+"/v1/chat/completions", "application/json", strings.NewReader(`{"messages":[{"role":"user","content":"ping"}]}`))
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	var body struct {
		Model   string `json:"model"`
		Choices []struct {
			Message struct {
				Content string `json:"content"`
			} `json:"message"`
		} `json:"choices"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&body); err != nil {
		t.Fatal(err)
	}
	if body.Model != "qwen3.6-35b" || len(body.Choices) != 1 || body.Choices[0].Message.Content != "pong" {
		t.Fatalf("unexpected chat response: %+v", body)
	}

	raw, err := os.ReadFile(filepath.Join(cacheDir, "fake-llama-argv.json"))
	if err != nil {
		t.Fatal(err)
	}
	var argv []string
	if err := json.Unmarshal(raw, &argv); err != nil {
		t.Fatal(err)
	}
	for _, want := range []string{"--jinja", "--alias", "qwen3.6-35b", "--n-cpu-moe", "2", "65536", "--no-mmproj", "--reasoning-budget", "0", `{"enable_thinking":false}`, "-fa", "on", "-ctk", "q8_0"} {
		if !slices.Contains(argv, want) {
			t.Fatalf("llama-server argv missing %q: %v", want, argv)
		}
	}
	if slices.Contains(argv, "--embedding") {
		t.Fatalf("chat model must not run in embedding mode: %v", argv)
	}
}

func findModuleRoot() (string, error) {
	wd, err := os.Getwd()
	if err != nil {
		return "", err
	}
	for current, i := wd, 0; i < 8; i++ {
		if info, statErr := os.Stat(filepath.Join(current, "go.mod")); statErr == nil && !info.IsDir() {
			return current, nil
		}
		next := filepath.Dir(current)
		if next == current {
			break
		}
		current = next
	}
	return "", fmt.Errorf("go.mod not found from %s", wd)
}

func buildForE2E(t *testing.T, root, output, pkg string) {
	t.Helper()
	cmd := exec.Command("go", "build", "-o", output, pkg)
	cmd.Dir = root
	if combined, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("build %s: %v: %s", pkg, err, combined)
	}
}

func allocateAddr(t *testing.T) string {
	t.Helper()
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	defer listener.Close()
	return listener.Addr().String()
}

func waitForStatus(t *testing.T, url string, want int, timeout time.Duration) {
	t.Helper()
	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		response, err := http.Get(url)
		if err == nil {
			_ = response.Body.Close()
			if response.StatusCode == want {
				return
			}
		}
		time.Sleep(100 * time.Millisecond)
	}
	t.Fatalf("timed out waiting for %s", url)
}

func stopE2EProcess(cmd *exec.Cmd) {
	if cmd == nil || cmd.Process == nil || cmd.ProcessState != nil {
		return
	}
	if runtime.GOOS == "windows" {
		_ = exec.Command("taskkill", "/T", "/F", "/PID", fmt.Sprintf("%d", cmd.Process.Pid)).Run()
	} else {
		_ = cmd.Process.Signal(os.Interrupt)
	}
	done := make(chan error, 1)
	go func() { done <- cmd.Wait() }()
	select {
	case <-done:
	case <-time.After(5 * time.Second):
		_ = cmd.Process.Kill()
	}
}
