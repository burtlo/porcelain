// Command genslugindex writes docs/generated/log-slug-index.md from operatorcopy/messages.yaml.
package main

import (
	"flag"
	"fmt"
	"os"
	"path/filepath"

	"github.com/lynn/porcelain/internal/operatorcopy"
	"github.com/lynn/porcelain/internal/operatorcopy/genslugindex"
)

func main() {
	out := flag.String("out", "", "output path (default: repo "+genslugindex.DefaultLogSlugIndexPath+")")
	flag.Parse()

	root, err := findRepoRoot()
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	reg, err := operatorcopy.LoadEmbedded()
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}

	target := *out
	if target == "" {
		target = filepath.Join(root, filepath.FromSlash(genslugindex.DefaultLogSlugIndexPath))
	} else if !filepath.IsAbs(target) {
		target = filepath.Join(root, target)
	}
	if err := os.MkdirAll(filepath.Dir(target), 0o755); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}

	f, err := os.Create(target)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	defer f.Close()

	if err := genslugindex.WriteLogSlugIndexMarkdown(f, reg); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	fmt.Printf("genslugindex: wrote %s (%d slugs)\n", target, len(reg.Messages))
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
			return "", fmt.Errorf("genslugindex: go.mod not found")
		}
		dir = parent
	}
}
