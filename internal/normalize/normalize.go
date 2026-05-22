package normalize

import (
	"strings"
	"unicode"

	"golang.org/x/text/unicode/norm"
)

// Fold normalizes to NFC and lowercases. Used for indexed keys.
func Fold(s string) string {
	s = norm.NFC.String(s)
	return strings.ToLower(s)
}

// SmartCaseQuery returns:
//   - lookupKey: always lowercased (for trie/trigram index lookup).
//   - scoreQuery: original case if the input has any uppercase rune,
//     otherwise lowercase. Used by the scorer.
//   - caseSensitive: true when the input had uppercase.
func SmartCaseQuery(q string) (lookupKey, scoreQuery string, caseSensitive bool) {
	q = norm.NFC.String(q)
	for _, r := range q {
		if unicode.IsUpper(r) {
			caseSensitive = true
			break
		}
	}
	lookupKey = strings.ToLower(q)
	if caseSensitive {
		return lookupKey, q, true
	}
	return lookupKey, lookupKey, false
}
