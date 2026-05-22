package index_test

import (
	"testing"

	"github.com/chriscast/chriscast/internal/index"
	"github.com/chriscast/chriscast/internal/search"
	"github.com/chriscast/chriscast/internal/store"
)

func TestUpsertRemove(t *testing.T) {
	ix := index.New()
	ix.Upsert(store.FileInfo{Path: "/a/readme.md", Base: "readme.md"})
	ix.Upsert(store.FileInfo{Path: "/b/project.go", Base: "project.go"})

	if got := search.Search(ix, "readme", 10); len(got) != 1 || got[0].File.Base != "readme.md" {
		t.Fatalf("expected readme.md, got %+v", got)
	}
	if got := search.Search(ix, "proj", 10); len(got) != 1 || got[0].File.Base != "project.go" {
		t.Fatalf("expected project.go, got %+v", got)
	}

	// rename: same path, new base
	ix.Upsert(store.FileInfo{Path: "/a/readme.md", Base: "README.MD"})
	if got := search.Search(ix, "readme", 10); len(got) != 1 || got[0].File.Base != "README.MD" {
		t.Fatalf("after upsert: expected README.MD, got %+v", got)
	}

	ix.Remove("/a/readme.md")
	if got := search.Search(ix, "readme", 10); len(got) != 0 {
		t.Fatalf("after remove: expected 0, got %+v", got)
	}
}
