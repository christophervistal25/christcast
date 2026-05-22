// Package trie implements a rune-keyed prefix trie over normalized
// basenames. It supports insertion, deletion, and prefix-walk traversal
// used by the search orchestrator to gather candidate FileIDs for short
// or prefix-anchored queries.
package trie
