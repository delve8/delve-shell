package maininput

import "fmt"

type ChoiceOption struct {
	Num   int
	Label string
}

type HighlightLine struct {
	Text      string
	Highlight bool
}

func BuildChoiceLines(options []ChoiceOption, selectedIndex int) []HighlightLine {
	lines := make([]HighlightLine, 0, len(options))
	for i, o := range options {
		lines = append(lines, HighlightLine{
			Text:      fmt.Sprintf("%d  %s", o.Num, o.Label),
			Highlight: i == selectedIndex,
		})
	}
	return lines
}

func WaitingHint(waitingForAI bool, inChoice bool, hint string) string {
	if waitingForAI && !inChoice {
		return "\n" + hint
	}
	return ""
}
