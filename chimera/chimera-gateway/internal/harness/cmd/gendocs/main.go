// Command gendocs writes docs/generated/harness-stages.md from harness runners.
package main

import (
	"flag"
	"fmt"
	"os"
	"path/filepath"

	"github.com/lynn/porcelain/chimera/chimera-gateway/internal/harness/gendocs"
)

func main() {
	out := flag.String("out", "", "output path (default: repo-relative "+gendocs.DefaultHarnessStagesPath+")")
	flag.Parse()

	root, err := findRepoRoot()
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}

	target := *out
	if target == "" {
		target = filepath.Join(root, filepath.FromSlash(gendocs.DefaultHarnessStagesPath))
	} else if !filepath.IsAbs(target) {
		target = filepath.Join(root, target)
	}

	f, err := os.Create(target)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	defer f.Close()

	if err := gendocs.WriteHarnessStagesMarkdown(f); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
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
			return "", fmt.Errorf("gendocs: go.mod not found (started at %s)", dir)
		}
		dir = parent
	}
}
