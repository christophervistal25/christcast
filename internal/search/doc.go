// Package search orchestrates query execution against the index. It
// gathers candidate FileIDs from the trie and trigram indexes, scores
// them with the fzf matcher in the score package, applies recency and
// extension biases, and selects the top-K results through a bounded
// heap. It also implements a directory-browse mode that combines
// in-index entries with a live readdir of the target directory.
package search
