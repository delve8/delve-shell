package slashdispatch

import (
	"strings"

	"delve-shell/internal/input/maininput"
	"delve-shell/internal/slash/flow"
	"delve-shell/internal/slash/view"
	"delve-shell/internal/uiflow/enterflow"
)

type Runtime[M any, C any] struct {
}

func NewRuntime[M any, C any]() *Runtime[M, C] {
	return &Runtime[M, C]{}
}

type Hooks[M any, C any] struct {
	BeforeDispatch func(string)
	AfterDispatch  func(string)
	ClearInput     func(M) M
}

type SuggestionContext func(input string) (visible []int, options []slashview.Option)

type ExecDeps[M any, C any] struct {
	Hooks               Hooks[M, C]
	SuggestionContext   SuggestionContext
	SlashSuggestIndex   func(M) int
	FillInput           func(M, string) M
	AppendSessionNone   func(M) M
	AppendDelRemoteNone func(M) M
	AppendUnknownSlash  func(M) M
	EchoSubmitted       func(M, string) M
	EmitChat            func(M, string) M
	SessionNoneMsg      string
	DelRemoteNoneMsg    string
}

func (r *Runtime[M, C]) ExecuteSubmission(m M, rawText string, selectedIndex int, deps ExecDeps[M, C]) (M, C) {
	var zero C
	text := strings.TrimSpace(rawText)
	if text == "" {
		return m, zero
	}

	if deps.SuggestionContext == nil {
		return deps.EmitChat(m, text), zero
	}
	vis, viewOpts := deps.SuggestionContext(text)
	plan := enterflow.PlanAfterSlashDispatches(text, selectedIndex, viewOpts, vis, deps.SessionNoneMsg, deps.DelRemoteNoneMsg)
	switch plan.Kind {
	case maininput.MainEnterShowSessionNone:
		return deps.AppendSessionNone(m), zero
	case maininput.MainEnterShowDelRemoteNone:
		return deps.AppendDelRemoteNone(m), zero
	case maininput.MainEnterResolveSelected:
		return deps.FillInput(m, slashview.ChosenToInputValue(plan.Selected)), zero
	case maininput.MainEnterUnknownSlash:
		return deps.AppendUnknownSlash(m), zero
	}

	return deps.EmitChat(m, text), zero
}

func (r *Runtime[M, C]) ExecuteEarlySubmission(m M, inputLine string, deps ExecDeps[M, C]) (M, C, bool) {
	var zero C
	trimmed := strings.TrimSpace(inputLine)
	if trimmed == "" || deps.SuggestionContext == nil || deps.SlashSuggestIndex == nil {
		return m, zero, false
	}
	vis, viewOpts := deps.SuggestionContext(inputLine)
	selected, ok := slashview.SelectedByVisibleIndex(viewOpts, vis, deps.SlashSuggestIndex(m))
	result := slashflow.EvaluateSlashEnter(inputLine, trimmed, selected, ok)
	switch result.Action {
	case slashflow.EnterKeyFillOnly:
		return deps.FillInput(m, result.Fill), zero, true
	}
	return m, zero, false
}
