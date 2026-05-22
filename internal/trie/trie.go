package trie

import "github.com/chriscast/chriscast/internal/store"

// Simple rune-keyed trie. Each node holds FileIDs of basenames that pass
// through (for prefix walks). Phase 1 — not patricia-compressed yet.
type node struct {
	children map[rune]*node
	ids      []store.FileID // file IDs whose basename equals path-to-here
}

type Trie struct {
	root *node
}

func New() *Trie { return &Trie{root: &node{children: map[rune]*node{}}} }

func (t *Trie) Delete(key string, id store.FileID) {
	n := t.root
	for _, r := range key {
		c, ok := n.children[r]
		if !ok {
			return
		}
		n = c
	}
	for i, v := range n.ids {
		if v == id {
			n.ids = append(n.ids[:i], n.ids[i+1:]...)
			return
		}
	}
}

func (t *Trie) Insert(key string, id store.FileID) {
	n := t.root
	for _, r := range key {
		c, ok := n.children[r]
		if !ok {
			c = &node{children: map[rune]*node{}}
			n.children[r] = c
		}
		n = c
	}
	n.ids = append(n.ids, id)
}

// PrefixIDs returns all FileIDs whose key starts with prefix (DFS gather).
// Bounded by `limit` to avoid runaway collection.
func (t *Trie) PrefixIDs(prefix string, limit int) []store.FileID {
	n := t.root
	for _, r := range prefix {
		c, ok := n.children[r]
		if !ok {
			return nil
		}
		n = c
	}
	out := make([]store.FileID, 0, 64)
	var walk func(*node) bool
	walk = func(x *node) bool {
		out = append(out, x.ids...)
		if len(out) >= limit {
			return true
		}
		for _, c := range x.children {
			if walk(c) {
				return true
			}
		}
		return false
	}
	walk(n)
	if len(out) > limit {
		out = out[:limit]
	}
	return out
}
