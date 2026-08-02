package tokenbudget

import "strings"

// EstimateConservative returns the shared deterministic token estimate used by
// domain compilation and application runtime budgeting. ASCII runes are
// estimated at four characters per token, while every non-ASCII rune counts as
// one token.
func EstimateConservative(value string) int {
	trimmed := strings.TrimSpace(value)
	if trimmed == "" {
		return 0
	}
	ascii := 0
	nonASCII := 0
	for _, r := range trimmed {
		if r <= 0x7f {
			ascii++
		} else {
			nonASCII++
		}
	}
	return max((ascii+3)/4+nonASCII, 1)
}
