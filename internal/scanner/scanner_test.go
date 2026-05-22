package scanner_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/chriscast/chriscast/internal/config"
	"github.com/chriscast/chriscast/internal/scanner"
	"github.com/chriscast/chriscast/internal/store"
)

func writeTree(t *testing.T, root string, entries map[string]string) {
	t.Helper()
	for p, content := range entries {
		full := filepath.Join(root, p)
		if err := os.MkdirAll(filepath.Dir(full), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(full, []byte(content), 0o644); err != nil {
			t.Fatal(err)
		}
	}
}

func TestScanBasic(t *testing.T) {
	root := t.TempDir()
	writeTree(t, root, map[string]string{
		"a.txt":          "x",
		"sub/b.txt":      "x",
		"sub/c/d.txt":    "x",
		"node_modules/x": "x", // should be excluded by name
		".hidden.txt":    "x", // hidden, excluded
	})

	cfg := &config.Config{
		Scopes:   []config.Scope{{Path: root}},
		Excludes: []string{"node_modules"},
	}
	var files []store.FileInfo
	st, err := scanner.Scan(cfg, func(fi store.FileInfo) {
		files = append(files, fi)
	})
	if err != nil {
		t.Fatalf("scan: %v", err)
	}
	if st.Files != 3 {
		t.Errorf("want 3 files, got %d", st.Files)
	}
	// dirs counted should include root + sub + sub/c (3)
	if st.Dirs < 3 {
		t.Errorf("want >=3 dirs, got %d", st.Dirs)
	}
	for _, fi := range files {
		if fi.Base == ".hidden.txt" {
			t.Errorf("hidden file should be excluded")
		}
		if fi.Base == "x" && filepath.Base(filepath.Dir(fi.Path)) == "node_modules" {
			t.Errorf("node_modules contents should be excluded")
		}
	}
}

func TestScanIncludeHidden(t *testing.T) {
	root := t.TempDir()
	writeTree(t, root, map[string]string{
		".dotfile": "x",
		"regular":  "x",
	})
	cfg := &config.Config{
		Scopes:        []config.Scope{{Path: root}},
		IncludeHidden: true,
	}
	count := 0
	if _, err := scanner.Scan(cfg, func(fi store.FileInfo) {
		count++
	}); err != nil {
		t.Fatalf("scan: %v", err)
	}
	if count != 2 {
		t.Errorf("want 2 (hidden included), got %d", count)
	}
}
