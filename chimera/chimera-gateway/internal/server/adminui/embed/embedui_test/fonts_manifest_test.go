package embedui_test

import (
	"bufio"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"
)

var codepointsJSEntry = regexp.MustCompile(`["']([a-z][a-z0-9_]*)["']\s*:\s*0x([0-9a-fA-F]+)`)

func TestFontsIconsTxtMatchesGeneratedAssets(t *testing.T) {
	root := embeduiRoot(t)
	fontsDir := filepath.Join(root, "fonts")
	want := readIconList(t, filepath.Join(fontsDir, "icons.txt"))
	if len(want) == 0 {
		t.Fatal("fonts/icons.txt is empty")
	}
	have := readCodepointNames(t, filepath.Join(fontsDir, "codepoints.js"))
	for name := range want {
		if !have[name] {
			t.Errorf("icons.txt entry %q missing from fonts/codepoints.js — run make adminui-fonts", name)
		}
	}
	for name := range have {
		if !want[name] {
			t.Errorf("codepoints.js has %q not listed in fonts/icons.txt — run make adminui-fonts", name)
		}
	}
	for _, name := range []string{
		"material-symbols-outlined.woff2",
		"material-symbols-outlined.ttf",
		"hanken-grotesk-latin.woff2",
		"hanken-grotesk-latin.ttf",
		"codepoints.js",
	} {
		p := filepath.Join(fontsDir, name)
		st, err := os.Stat(p)
		if err != nil || st.Size() < 50 {
			t.Fatalf("expected font asset %s (run make adminui-fonts)", p)
		}
	}
	css := mustReadFile(t, filepath.Join(root, "fonts.css"))
	if !strings.Contains(css, "material-symbols-outlined.woff2") {
		t.Fatal("fonts.css must reference material-symbols-outlined.woff2")
	}
}

func readIconList(t *testing.T, path string) map[string]bool {
	t.Helper()
	b, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read %s: %v", path, err)
	}
	out := map[string]bool{}
	sc := bufio.NewScanner(strings.NewReader(string(b)))
	for sc.Scan() {
		line := strings.TrimSpace(sc.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		out[line] = true
	}
	return out
}

func readCodepointNames(t *testing.T, path string) map[string]bool {
	t.Helper()
	b, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read %s: %v (run make adminui-fonts)", path, err)
	}
	out := map[string]bool{}
	for _, m := range codepointsJSEntry.FindAllStringSubmatch(string(b), -1) {
		out[m[1]] = true
	}
	return out
}
