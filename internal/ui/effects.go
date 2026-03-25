package ui

import (
	"delve-shell/internal/host/app"
	"delve-shell/internal/host/route"
)

// Effects isolates side-effectful UI outputs from host read-model access.
// The UI package calls this interface for submit/relay/trace behaviors.
type Effects interface {
	Submit(text string) bool
	TryRelaySlashSubmit(p route.SlashSubmitPayload) bool
	RequestSlashDispatch(line string)
	TraceSlashEntered(line string)
	TakeOpenConfigLLMOnFirstLayout() bool
}

type hostEffects struct {
	host app.Host
}

func (h hostEffects) Submit(text string) bool {
	if h.host == nil {
		return false
	}
	return h.host.Submit(text)
}

func (h hostEffects) TryRelaySlashSubmit(p route.SlashSubmitPayload) bool {
	if h.host == nil {
		return false
	}
	return h.host.TryRelaySlashSubmit(p)
}

func (h hostEffects) RequestSlashDispatch(line string) {
	if h.host == nil {
		return
	}
	h.host.RequestSlashDispatch(line)
}

func (h hostEffects) TraceSlashEntered(line string) {
	if h.host == nil {
		return
	}
	h.host.TraceSlashEntered(line)
}

func (h hostEffects) TakeOpenConfigLLMOnFirstLayout() bool {
	if h.host == nil {
		return false
	}
	return h.host.TakeOpenConfigLLMOnFirstLayout()
}

func (m Model) submitEffect(text string) bool {
	if m.Effects == nil {
		return false
	}
	return m.Effects.Submit(text)
}

func (m Model) relaySlashSubmitEffect(p route.SlashSubmitPayload) bool {
	if m.Effects == nil {
		return false
	}
	return m.Effects.TryRelaySlashSubmit(p)
}

func (m Model) requestSlashDispatchEffect(line string) {
	if m.Effects == nil {
		return
	}
	m.Effects.RequestSlashDispatch(line)
}

func (m Model) traceSlashEnteredEffect(line string) {
	if m.Effects == nil {
		return
	}
	m.Effects.TraceSlashEntered(line)
}

func (m Model) takeOpenConfigLLMOnFirstLayoutEffect() bool {
	if m.Effects == nil {
		return false
	}
	return m.Effects.TakeOpenConfigLLMOnFirstLayout()
}

