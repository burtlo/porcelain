// Command routes writes docs/generated/gateway-http-routes.md from gateway route registrations.
package main

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/lynn/porcelain/internal/docgen/routes"
)

func main() {
	root, err := findRepoRoot()
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	out := filepath.Join(root, filepath.FromSlash(routes.DefaultMarkdownPath))
	if err := os.MkdirAll(filepath.Dir(out), 0o755); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	collected, err := routes.Collect(root)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	f, err := os.Create(out)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	defer f.Close()
	if err := routes.WriteMarkdown(f, collected); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	fmt.Printf("wrote %s\n", out)
}

func findRepoRoot() (string, error) {
	dir, err := os.Getwd()
	if err != nil {
		return "", err
	}
	for {
		if _, err := os.Stat(filepath.Join(dir, "go.mod")); err == nil {
			return dir, nil
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			return "", fmt.Errorf("routes cmd: go.mod not found (started at %s)", dir)
		}
		dir = parent
	}
}
