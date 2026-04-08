package ui

import "regexp"

// CommonMark emphasis rules require ** to be in a "left-flanking" position to open strong
// emphasis. A pattern like 个**"quote" fails because ** is followed by punctuation " but not
// preceded by whitespace/punctuation, so goldmark leaves ** as literal text.
//
// We only add a space *after* CJK and *before* ** when ** is immediately followed by a quote or
// opener — not when ** is a closer after a CJK word (e.g. 状况** must not become 状况 **).
//
// For *closing* ** before CJK (e.g. ..."**专), we add a space after ** only when ** is preceded
// by ASCII alnum or closing punctuation, not by CJK (so "1. **按" stays valid).
var (
	reCJKBeforeStrongOpenQuote = regexp.MustCompile(
		`([\p{Han}\p{Hiragana}\p{Katakana}\p{Hangul}])(\*\*)(["'\x{201c}\x{201d}\(\[\{])`,
	)
	reStrongCloserBeforeCJK = regexp.MustCompile(
		`([a-zA-Z0-9_"\x{201c}\x{201d}'\)\]\}>])(\*\*)([\p{Han}\p{Hiragana}\p{Katakana}\p{Hangul}])`,
	)
)

// relaxMarkdownStrongAdjacentCJK adjusts spacing so **...** can parse as strong emphasis next to
// CJK (goldmark/CommonMark delimiter rules).
func relaxMarkdownStrongAdjacentCJK(s string) string {
	s = reCJKBeforeStrongOpenQuote.ReplaceAllString(s, "$1 $2$3")
	s = reStrongCloserBeforeCJK.ReplaceAllString(s, "$1$2 $3")
	return s
}
