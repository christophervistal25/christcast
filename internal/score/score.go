package score

import "unicode"

// fzf-style scoring constants.
const (
	ScoreMatch               = 16
	ScoreGapStart            = -3
	ScoreGapExtend           = -1
	BonusBoundary            = 8
	BonusNonWord             = 8
	BonusCamel               = 7
	BonusConsecutive         = 4
	BonusFirstCharMultiplier = 2
)

type charClass int

const (
	classNonWord charClass = iota
	classLower
	classUpper
	classDigit
)

func classOf(r rune) charClass {
	switch {
	case unicode.IsLower(r):
		return classLower
	case unicode.IsUpper(r):
		return classUpper
	case unicode.IsDigit(r):
		return classDigit
	default:
		return classNonWord
	}
}

func bonusFor(prev, cur charClass) int {
	if prev == classNonWord && cur != classNonWord {
		return BonusBoundary
	}
	if (prev == classLower && cur == classUpper) ||
		(prev != classDigit && cur == classDigit) {
		return BonusCamel
	}
	if cur == classNonWord {
		return BonusNonWord
	}
	return 0
}

// Match runs a greedy fzf v1-style fuzzy match. Returns -1 if pattern
// chars cannot be matched in order. Both inputs should already be folded
// for case handling.
func Match(text, pattern string) int {
	if pattern == "" {
		return 0
	}
	tr := []rune(text)
	pr := []rune(pattern)
	// forward scan to find an end index covering all pattern chars
	pidx := 0
	sidx := -1
	eidx := -1
	for i, r := range tr {
		if r == pr[pidx] {
			if sidx < 0 {
				sidx = i
			}
			pidx++
			if pidx == len(pr) {
				eidx = i + 1
				break
			}
		}
	}
	if eidx < 0 {
		return -1
	}
	// tighten left: scan back from eidx
	pidx = len(pr) - 1
	for i := eidx - 1; i >= sidx; i-- {
		if tr[i] == pr[pidx] {
			if pidx == 0 {
				sidx = i
				break
			}
			pidx--
		}
	}
	return calc(tr, pr, sidx, eidx)
}

func calc(text, pattern []rune, sidx, eidx int) int {
	pidx := 0
	score := 0
	inGap := false
	consecutive := 0
	firstBonus := 0
	prevClass := classNonWord
	if sidx > 0 {
		prevClass = classOf(text[sidx-1])
	}
	for i := sidx; i < eidx; i++ {
		r := text[i]
		cls := classOf(r)
		if pidx < len(pattern) && r == pattern[pidx] {
			score += ScoreMatch
			b := bonusFor(prevClass, cls)
			if consecutive == 0 {
				firstBonus = b
			} else {
				if b >= BonusBoundary && b > firstBonus {
					firstBonus = b
				}
				if b < firstBonus {
					b = firstBonus
				}
				if BonusConsecutive > b {
					b = BonusConsecutive
				}
			}
			if pidx == 0 {
				score += b * BonusFirstCharMultiplier
			} else {
				score += b
			}
			inGap = false
			consecutive++
			pidx++
		} else {
			if inGap {
				score += ScoreGapExtend
			} else {
				score += ScoreGapStart
			}
			inGap = true
			consecutive = 0
			firstBonus = 0
		}
		prevClass = cls
	}
	return score
}
