package trie

import (
	"sort"
	"testing"

	"github.com/chriscast/chriscast/internal/store"
)

func TestEmptyTrieReturnsNil(t *testing.T) {
	tr := New()
	if got := tr.PrefixIDs("anything", 100); got != nil {
		t.Fatalf("expected nil from empty trie, got %v", got)
	}
}

func TestSingleInsertAndPrefixWalk(t *testing.T) {
	tr := New()
	tr.Insert("readme", store.FileID(1))
	ids := tr.PrefixIDs("read", 100)
	if len(ids) != 1 || ids[0] != 1 {
		t.Fatalf("expected [1], got %v", ids)
	}
	if got := tr.PrefixIDs("xyz", 100); got != nil {
		t.Fatalf("expected nil for missing prefix, got %v", got)
	}
}

func TestMultiInsertSharingPrefix(t *testing.T) {
	tr := New()
	tr.Insert("foo", store.FileID(1))
	tr.Insert("foobar", store.FileID(2))
	tr.Insert("foobaz", store.FileID(3))
	tr.Insert("other", store.FileID(4))

	ids := tr.PrefixIDs("foo", 100)
	sort.Slice(ids, func(i, j int) bool { return ids[i] < ids[j] })
	if len(ids) != 3 || ids[0] != 1 || ids[1] != 2 || ids[2] != 3 {
		t.Fatalf("expected [1 2 3] under 'foo', got %v", ids)
	}
}

func TestDeleteRemovesOnlySpecifiedID(t *testing.T) {
	tr := New()
	tr.Insert("foo", store.FileID(1))
	tr.Insert("foo", store.FileID(2))
	tr.Delete("foo", store.FileID(1))
	ids := tr.PrefixIDs("foo", 100)
	if len(ids) != 1 || ids[0] != 2 {
		t.Fatalf("expected [2] after deleting id 1, got %v", ids)
	}
}

func TestDeleteNonexistentKeyNoOp(t *testing.T) {
	tr := New()
	tr.Insert("foo", store.FileID(1))
	tr.Delete("bar", store.FileID(99)) // missing key
	tr.Delete("foo", store.FileID(99)) // present key, missing id
	ids := tr.PrefixIDs("foo", 100)
	if len(ids) != 1 || ids[0] != 1 {
		t.Fatalf("expected [1] unchanged, got %v", ids)
	}
}
