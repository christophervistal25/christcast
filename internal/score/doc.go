// Package score implements an fzf v1 style greedy fuzzy matcher. It
// returns a numeric score that rewards word boundaries, camelCase
// transitions, and consecutive character matches, and is used by the
// search package to rank candidate FileIDs.
package score
