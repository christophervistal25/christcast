package trigram

import "github.com/chriscast/chriscast/internal/store"

// Sentinel pads short strings so 1-2 char basenames still get keys.
const sentinel = '\x00'

type Index struct {
	m map[string][]store.FileID
}

func New() *Index { return &Index{m: map[string][]store.FileID{}} }

// runes builds padded rune slice: $ + s + $.
func runes(s string) []rune {
	out := make([]rune, 0, len(s)+2)
	out = append(out, sentinel)
	for _, r := range s {
		out = append(out, r)
	}
	out = append(out, sentinel)
	return out
}

// indexGrams pads with sentinels — captures word-start/word-end anchors.
func indexGrams(s string) []string {
	rs := runes(s)
	if len(rs) < 3 {
		return nil
	}
	out := make([]string, 0, len(rs)-2)
	for i := 0; i <= len(rs)-3; i++ {
		out = append(out, string(rs[i:i+3]))
	}
	return out
}

// queryGrams does NOT pad — query is treated as a substring, not anchored.
func queryGrams(s string) []string {
	rs := []rune(s)
	if len(rs) < 3 {
		return nil
	}
	out := make([]string, 0, len(rs)-2)
	for i := 0; i <= len(rs)-3; i++ {
		out = append(out, string(rs[i:i+3]))
	}
	return out
}

func (ix *Index) Insert(s string, id store.FileID) {
	for _, g := range indexGrams(s) {
		ix.m[g] = append(ix.m[g], id)
	}
}

func (ix *Index) Delete(s string, id store.FileID) {
	for _, g := range indexGrams(s) {
		lst := ix.m[g]
		for i, v := range lst {
			if v == id {
				ix.m[g] = append(lst[:i], lst[i+1:]...)
				break
			}
		}
		if len(ix.m[g]) == 0 {
			delete(ix.m, g)
		}
	}
}

// Candidates returns FileIDs whose trigrams intersect all query trigrams.
// Returns at most `limit` candidates.
func (ix *Index) Candidates(q string, limit int) []store.FileID {
	gs := queryGrams(q)
	if len(gs) == 0 {
		return nil
	}
	// pick rarest gram first
	var sets [][]store.FileID
	for _, g := range gs {
		s := ix.m[g]
		if len(s) == 0 {
			return nil
		}
		sets = append(sets, s)
	}
	// intersect via counting (multiset)
	counts := map[store.FileID]int{}
	for _, s := range sets {
		seen := map[store.FileID]bool{}
		for _, id := range s {
			if !seen[id] {
				seen[id] = true
				counts[id]++
			}
		}
	}
	need := len(sets)
	out := make([]store.FileID, 0, 64)
	for id, c := range counts {
		if c == need {
			out = append(out, id)
			if len(out) >= limit {
				break
			}
		}
	}
	return out
}
