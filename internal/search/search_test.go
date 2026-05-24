package search_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/chriscast/chriscast/internal/index"
	"github.com/chriscast/chriscast/internal/search"
	"github.com/chriscast/chriscast/internal/store"
)

func newIxWith(files ...store.FileInfo) *index.Index {
	ix := index.New()
	for _, fi := range files {
		ix.Upsert(fi)
	}
	return ix
}

func TestEmptyQuery(t *testing.T) {
	ix := newIxWith(store.FileInfo{Path: "/a/b.txt", Base: "b.txt"})
	if got := search.Search(ix, "", 10); got != nil {
		t.Fatalf("empty query: want nil, got %v", got)
	}
}

func TestCaseInsensitiveMatch(t *testing.T) {
	ix := newIxWith(store.FileInfo{Path: "/x/README.md", Base: "README.md"})
	got := search.Search(ix, "readme", 10)
	if len(got) != 1 || got[0].File.Base != "README.md" {
		t.Fatalf("want README.md, got %+v", got)
	}
}

func TestSmartCaseSensitivity(t *testing.T) {
	ix := newIxWith(
		store.FileInfo{Path: "/x/README.md", Base: "README.md"},
		store.FileInfo{Path: "/x/readme.md", Base: "readme.md"},
	)
	got := search.Search(ix, "README", 10)
	if len(got) == 0 {
		t.Fatal("uppercase query should still find README.md")
	}
	// at least the uppercase one matches.
	foundUpper := false
	for _, r := range got {
		if r.File.Base == "README.md" {
			foundUpper = true
		}
	}
	if !foundUpper {
		t.Fatalf("uppercase query missed README.md: %+v", got)
	}
}

func TestDirBrowseIndexed(t *testing.T) {
	ix := newIxWith(
		store.FileInfo{Path: "/proj", Base: "proj", IsDir: true},
		store.FileInfo{Path: "/proj/a.txt", Base: "a.txt"},
		store.FileInfo{Path: "/proj/b.txt", Base: "b.txt"},
		store.FileInfo{Path: "/proj/sub", Base: "sub", IsDir: true},
		store.FileInfo{Path: "/other", Base: "other", IsDir: true},
	)
	got := search.Search(ix, "/proj", 10)
	if len(got) != 3 {
		t.Fatalf("want 3 children of /proj, got %d: %+v", len(got), got)
	}
	if !got[0].File.IsDir {
		t.Fatalf("dirs should sort first, got %+v", got[0])
	}
}

func TestLiveReaddirFallback(t *testing.T) {
	dir := t.TempDir()
	for _, n := range []string{"alpha.txt", "beta.go", "gamma.md"} {
		if err := os.WriteFile(filepath.Join(dir, n), nil, 0o644); err != nil {
			t.Fatal(err)
		}
	}
	if err := os.Mkdir(filepath.Join(dir, "sub"), 0o755); err != nil {
		t.Fatal(err)
	}
	ix := index.New() // empty
	got := search.Search(ix, dir, 20)
	if len(got) != 4 {
		t.Fatalf("want 4 entries, got %d: %+v", len(got), got)
	}
	// dirs first
	if !got[0].File.IsDir {
		t.Fatalf("first result should be dir, got %+v", got[0])
	}
}

func TestAppOutranksSameNamedFile(t *testing.T) {
	// An indexed .desktop app (IsApp=true) should rank above a regular file
	// with the same basename for a query that matches that basename.
	ix := newIxWith(
		store.FileInfo{Path: "/home/u/notes/firefox", Base: "firefox"},
		store.FileInfo{Path: "/usr/share/applications/firefox.desktop", Base: "Firefox", IsApp: true},
		store.FileInfo{Path: "/tmp/firefox.log", Base: "firefox.log"},
	)
	got := search.Search(ix, "firefox", 10)
	if len(got) == 0 {
		t.Fatal("expected at least one result for 'firefox'")
	}
	if !got[0].File.IsApp {
		t.Fatalf("app entry should rank first, got %+v (full: %+v)", got[0].File, got)
	}
	if got[0].File.Base != "Firefox" {
		t.Fatalf("first result should be the Firefox app, got base=%q", got[0].File.Base)
	}
}
