package slashdispatch

import (
	"strings"

	"delve-shell/internal/maininput"
	"delve-shell/internal/slashflow"
	"delve-shell/internal/slashreg"
	"delve-shell/internal/slashview"
	"delve-shell/internal/uiflow/enterflow"
)

type ExactEntry[M any, C any] struct {
	Handle     func(M) (M, C)
	ClearInput bool
}

type PrefixEntry[M any, C any] struct {
	Prefix string
	Handle func(M, string) (M, C, bool)
}

type SelectedProvider[M any, C any] func(M, string) (M, C, bool)

type Runtime[M any, C any] struct {
	exact    *slashreg.ExactRegistry[M, C]
	prefix   *slashreg.PrefixRegistry[M, C]
	selected *slashreg.ProviderChain[SelectedProvider[M, C]]
}

func NewRuntime[M any, C any]() *Runtime[M, C] {
	return &Runtime[M, C]{
		exact:    slashreg.NewExactRegistry[M, C](),
		prefix:   slashreg.NewPrefixRegistry[M, C](),
		selected: slashreg.NewProviderChain[SelectedProvider[M, C]](),
	}
}

func (r *Runtime[M, C]) RegisterExact(cmd string, entry ExactEntry[M, C]) {
	if r == nil || cmd == "" {
		return
	}
	r.exact.Set(cmd, slashreg.ExactEntry[M, C]{
		Handle:     entry.Handle,
		ClearInput: entry.ClearInput,
	})
}

func (r *Runtime[M, C]) RegisterPrefix(prefix string, entry PrefixEntry[M, C]) {
	if r == nil || prefix == "" {
		return
	}
	if entry.Prefix == "" {
		entry.Prefix = prefix
	}
	r.prefix.Set(prefix, slashreg.PrefixEntry[M, C]{
		Prefix: entry.Prefix,
		Handle: entry.Handle,
	})
}

func (r *Runtime[M, C]) RegisterSelected(p SelectedProvider[M, C]) {
	if r == nil || p == nil {
		return
	}
	r.selected.Add(p, func(x SelectedProvider[M, C]) bool { return x == nil })
}

func (r *Runtime[M, C]) HasExact(cmd string) bool {
	if r == nil || cmd == "" {
		return false
	}
	_, ok := r.exact.Get(cmd)
	return ok
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

	if m2, cmd, handled := r.dispatchExact(m, text, deps.Hooks); handled {
		return m2, cmd
	}
	if m2, cmd, handled := r.dispatchPrefix(m, text, deps.Hooks); handled {
		return m2, cmd
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
		if m2, cmd, handled := r.dispatchSelected(m, plan.Chosen, deps.Hooks); handled {
			return m2, cmd
		}
		if m2, cmd, handled := r.dispatchExact(m, plan.Chosen, deps.Hooks); handled {
			return m2, cmd
		}
		if m2, cmd, handled := r.dispatchPrefix(m, plan.Chosen, deps.Hooks); handled {
			return m2, cmd
		}
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
	case slashflow.EnterKeyDispatchExactChosen:
		if r.HasExact(selected.Cmd) {
			m = deps.EchoSubmitted(m, trimmed)
		}
		if m2, cmd, handled := r.dispatchExact(m, selected.Cmd, deps.Hooks); handled {
			return m2, cmd, true
		}
	case slashflow.EnterKeyFillOnly:
		return deps.FillInput(m, result.Fill), zero, true
	}
	if r.HasExact(trimmed) {
		m = deps.EchoSubmitted(m, trimmed)
	}
	if m2, cmd, handled := r.dispatchExact(m, trimmed, deps.Hooks); handled {
		return m2, cmd, true
	}
	return m, zero, false
}

func (r *Runtime[M, C]) dispatchExact(m M, cmd string, hooks Hooks[M, C]) (M, C, bool) {
	var zero C
	entry, ok := r.exact.Get(cmd)
	if !ok {
		return m, zero, false
	}
	if hooks.BeforeDispatch != nil {
		hooks.BeforeDispatch(cmd)
	}
	m, outCmd := entry.Handle(m)
	if entry.ClearInput && hooks.ClearInput != nil {
		m = hooks.ClearInput(m)
	}
	if hooks.AfterDispatch != nil {
		hooks.AfterDispatch(cmd)
	}
	return m, outCmd, true
}

func (r *Runtime[M, C]) dispatchPrefix(m M, text string, hooks Hooks[M, C]) (M, C, bool) {
	var zero C
	for _, e := range r.prefix.Entries() {
		if strings.HasPrefix(text, e.Prefix) {
			rest := strings.TrimPrefix(text, e.Prefix)
			if hooks.BeforeDispatch != nil {
				hooks.BeforeDispatch(text)
			}
			m2, outCmd, handled := e.Handle(m, rest)
			if handled {
				if hooks.ClearInput != nil {
					m2 = hooks.ClearInput(m2)
				}
				if hooks.AfterDispatch != nil {
					hooks.AfterDispatch(text)
				}
			}
			return m2, outCmd, handled
		}
	}
	return m, zero, false
}

func (r *Runtime[M, C]) dispatchSelected(m M, chosen string, hooks Hooks[M, C]) (M, C, bool) {
	var zero C
	if hooks.BeforeDispatch != nil {
		hooks.BeforeDispatch(chosen)
	}
	for _, p := range r.selected.List() {
		if m2, cmd, handled := p(m, chosen); handled {
			if hooks.AfterDispatch != nil {
				hooks.AfterDispatch(chosen)
			}
			return m2, cmd, true
		}
	}
	return m, zero, false
}
