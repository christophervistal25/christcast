package store

import "testing"

func TestAddReturnsSequentialIDs(t *testing.T) {
	s := New()
	id1 := s.Add(FileInfo{Path: "/a", Base: "a"})
	id2 := s.Add(FileInfo{Path: "/b", Base: "b"})
	id3 := s.Add(FileInfo{Path: "/c", Base: "c"})
	if id1 != 1 || id2 != 2 || id3 != 3 {
		t.Fatalf("expected sequential 1,2,3 — got %d,%d,%d", id1, id2, id3)
	}
	if s.Len() != 3 {
		t.Fatalf("expected Len 3, got %d", s.Len())
	}
}

func TestGetReturnsNilForTombstonedAndOutOfRange(t *testing.T) {
	s := New()
	id := s.Add(FileInfo{Path: "/a", Base: "a"})
	if s.Get(id) == nil {
		t.Fatalf("expected non-nil before tombstone")
	}
	s.Tombstone(id)
	if s.Get(id) != nil {
		t.Fatalf("expected nil after tombstone")
	}
	if s.Get(FileID(0)) != nil {
		t.Fatalf("expected nil for id=0")
	}
	if s.Get(FileID(9999)) != nil {
		t.Fatalf("expected nil for out-of-range id")
	}
}

func TestIDByPath(t *testing.T) {
	s := New()
	id := s.Add(FileInfo{Path: "/foo/bar", Base: "bar"})
	got, ok := s.IDByPath("/foo/bar")
	if !ok || got != id {
		t.Fatalf("expected id=%d ok=true, got id=%d ok=%v", id, got, ok)
	}
	if _, ok := s.IDByPath("/nope"); ok {
		t.Fatalf("expected lookup of missing path to return ok=false")
	}
}

// Simulate a Store created via msgpack Load — Files populated, byPath nil.
func TestEnsureMapAfterLiteralConstruction(t *testing.T) {
	s := &Store{
		Files: []FileInfo{
			{ID: 1, Path: "/x", Base: "x"},
			{ID: 2, Path: "/y", Base: "y"},
		},
	}
	// byPath nil; IDByPath should ensureMap on demand.
	got, ok := s.IDByPath("/x")
	if !ok || got != 1 {
		t.Fatalf("expected id=1 ok=true after ensureMap, got id=%d ok=%v", got, ok)
	}
	got2, ok2 := s.IDByPath("/y")
	if !ok2 || got2 != 2 {
		t.Fatalf("expected id=2 ok=true, got id=%d ok=%v", got2, ok2)
	}
}
