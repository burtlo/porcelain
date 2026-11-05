package config

import (
	"os"
	"path/filepath"
	"testing"
)

func TestReplaceFile(t *testing.T) {
	dir := t.TempDir()
	p := filepath.Join(dir, "f.txt")
	if err := ReplaceFile(p, []byte("a"), 0o644); err != nil {
		t.Fatal(err)
	}
	b, err := os.ReadFile(p)
	if err != nil || string(b) != "a" {
		t.Fatalf("got %q err %v", b, err)
	}
	if err := ReplaceFile(p, []byte("bb"), 0o644); err != nil {
		t.Fatal(err)
	}
	b, _ = os.ReadFile(p)
	if string(b) != "bb" {
		t.Fatalf("got %q", b)
	}
}

func TestCommitRoutingAndChimera_rollbackChimera(t *testing.T) {
	dir := t.TempDir()
	routePath := filepath.Join(dir, "routing-policy.yaml")
	chimeraPath := filepath.Join(dir, "chimera.yaml")
	if err := os.WriteFile(routePath, []byte("route-v1"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(chimeraPath, []byte("gw-v1"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.Remove(chimeraPath); err != nil {
		t.Fatal(err)
	}
	if err := os.Mkdir(chimeraPath, 0o755); err != nil {
		t.Fatal(err)
	}
	err := CommitRoutingAndChimera(routePath, []byte("route-v2"), 0o644, chimeraPath, []byte("gw-v2"), 0o644)
	if err == nil {
		t.Fatal("expected error when chimera path is not a writable file")
	}
	rb, _ := os.ReadFile(routePath)
	if string(rb) != "route-v1" {
		t.Fatalf("routing file not rolled back: %q", rb)
	}
}

func TestCommitRoutingAndChimera_success(t *testing.T) {
	dir := t.TempDir()
	routePath := filepath.Join(dir, "routing-policy.yaml")
	chimeraPath := filepath.Join(dir, "chimera.yaml")
	err := CommitRoutingAndChimera(routePath, []byte("r2"), 0o644, chimeraPath, []byte("g2"), 0o644)
	if err != nil {
		t.Fatal(err)
	}
	rb, _ := os.ReadFile(routePath)
	gb, _ := os.ReadFile(chimeraPath)
	if string(rb) != "r2" || string(gb) != "g2" {
		t.Fatalf("r=%q g=%q", rb, gb)
	}
}
