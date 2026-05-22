package trigram

import (
	"testing"

	"github.com/chriscast/chriscast/internal/store"
)

func TestIntersect(t *testing.T) {
	ix := New()
	ix.Insert("readme", 1)
	ix.Insert("readable", 2)
	ix.Insert("apple", 3)
	got := ix.Candidates("read", 100)
	if len(got) != 2 {
		t.Fatalf("want 2 candidates, got %d: %v", len(got), got)
	}
	gotSet := map[store.FileID]bool{}
	for _, id := range got {
		gotSet[id] = true
	}
	if !gotSet[1] || !gotSet[2] {
		t.Errorf("missing expected ids: %v", gotSet)
	}
}
