package maininput

import (
	"strings"

	"delve-shell/internal/slashview"
	"delve-shell/internal/textwrap"
)

type CaptureInput struct {
	InputVal     string
	Text         string
	SuggestIndex int
	Selected     slashview.Option
	HasSelected  bool
}

type CaptureResult struct {
	FillOnly      bool
	FillValue     string
	SelectedIndex int
}

func CaptureSlashSelection(in CaptureInput) CaptureResult {
	res := CaptureResult{SelectedIndex: -1}
	if !strings.HasPrefix(in.InputVal, "/") || !in.HasSelected {
		return res
	}
	if slashview.ShouldFillOnly(in.Selected.Cmd, in.Text) {
		res.FillOnly = true
		res.FillValue = slashview.ChosenToInputValue(in.Selected.Cmd)
		return res
	}
	res.SelectedIndex = in.SuggestIndex
	return res
}

type SyncInput struct {
	InputVal            string
	CurrentSuggestIndex int
	VisibleCount        int
}

func SyncSlashSuggestIndex(in SyncInput) int {
	if !strings.HasPrefix(in.InputVal, "/") {
		return in.CurrentSuggestIndex
	}
	next := 0
	if in.VisibleCount > 0 && next >= in.VisibleCount {
		next = 0
	}
	return next
}

func IsNewSessionCommand(text string) bool {
	return text == "/new"
}

func AppendUserInputLines(messages []string, userLabel string, text string, width int, sepLine string) []string {
	userLine := userLabel + text
	if len(messages) > 0 && messages[len(messages)-1] != sepLine {
		messages = append(messages, sepLine)
	}
	messages = append(messages, textwrap.WrapString(userLine, width))
	messages = append(messages, "")
	return messages
}
