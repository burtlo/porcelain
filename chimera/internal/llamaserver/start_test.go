package llamaserver

import (
	"os"
	"path/filepath"
	"slices"
	"testing"
)

func TestCommandBuildsEmbeddingServerArgs(t *testing.T) {
	dir := t.TempDir()
	model := filepath.Join(dir, "model.gguf")
	if err := os.WriteFile(model, []byte("fake"), 0o644); err != nil {
		t.Fatal(err)
	}
	cmd, err := Command(Config{Bin: "llama-server", ModelPath: model, CacheDir: filepath.Join(dir, "cache"), BindHost: "127.0.0.1", Port: 18090, CtxSize: 512, NGPULayers: 4, Pooling: "mean"})
	if err != nil {
		t.Fatal(err)
	}
	for _, want := range []string{"--embedding", "--metrics", "--pooling", "mean", "--port", "18090"} {
		if !slices.Contains(cmd.Args, want) {
			t.Fatalf("args missing %q: %v", want, cmd.Args)
		}
	}
}

func TestCommandRequiresModelFile(t *testing.T) {
	_, err := Command(Config{Bin: "llama-server", ModelPath: filepath.Join(t.TempDir(), "missing.gguf"), BindHost: "127.0.0.1", Port: 18090})
	if err == nil {
		t.Fatal("expected missing model error")
	}
}

func TestCommandBuildsChatServerArgs(t *testing.T) {
	dir := t.TempDir()
	model := filepath.Join(dir, "model.gguf")
	if err := os.WriteFile(model, []byte("fake"), 0o644); err != nil {
		t.Fatal(err)
	}
	cmd, err := Command(Config{Mode: ModeChat, Bin: "llama-server", ModelPath: model, BindHost: "127.0.0.1", Port: 18181,
		CtxSize: 65536, NGPULayers: 99, NCPUMoE: 2, Alias: "qwen3.6-35b", ExtraArgs: []string{"-fa", "on", " "}})
	if err != nil {
		t.Fatal(err)
	}
	for _, want := range []string{"--jinja", "--alias", "qwen3.6-35b", "--n-cpu-moe", "2", "--no-mmproj", "--reasoning-budget", "0", `{"enable_thinking":false}`, "-fa", "on", "65536", "--parallel", "1"} {
		if !slices.Contains(cmd.Args, want) {
			t.Fatalf("args missing %q: %v", want, cmd.Args)
		}
	}
	for _, unwanted := range []string{"--embedding", "--pooling", " ", ""} {
		if slices.Contains(cmd.Args[1:], unwanted) {
			t.Fatalf("chat args must not contain %q: %v", unwanted, cmd.Args)
		}
	}
}

func TestCommandChatThinkingAndMmproj(t *testing.T) {
	dir := t.TempDir()
	model := filepath.Join(dir, "model.gguf")
	mm := filepath.Join(dir, "mmproj.gguf")
	for _, p := range []string{model, mm} {
		if err := os.WriteFile(p, []byte("fake"), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	cmd, err := Command(Config{Mode: ModeChat, Bin: "llama-server", ModelPath: model, MmprojPath: mm, Thinking: true, BindHost: "127.0.0.1", Port: 18181})
	if err != nil {
		t.Fatal(err)
	}
	for _, want := range []string{"--mmproj", `{"enable_thinking":true}`, "-1"} {
		if !slices.Contains(cmd.Args, want) {
			t.Fatalf("args missing %q: %v", want, cmd.Args)
		}
	}
	if slices.Contains(cmd.Args, "--no-mmproj") || slices.Contains(cmd.Args, "--n-cpu-moe") {
		t.Fatalf("unexpected args: %v", cmd.Args)
	}
	if _, err := Command(Config{Mode: ModeChat, Bin: "llama-server", ModelPath: model, MmprojPath: filepath.Join(dir, "missing.gguf"), BindHost: "127.0.0.1", Port: 1}); err == nil {
		t.Fatal("expected missing mmproj error")
	}
}
