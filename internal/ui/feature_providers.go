package ui

import tea "github.com/charmbracelet/bubbletea"

// SlashOptionsProvider can provide slash suggestion options for a given input.
// When handled==true, the returned options should override the default ui logic.
type SlashOptionsProvider func(
	inputVal string,
	lang string,
	currentSessionPath string,
	localRunCommands []string,
	remoteRunCommands []string,
	remoteActive bool,
) (opts []SlashOption, handled bool)

var slashOptionsProviders []SlashOptionsProvider

// RegisterSlashOptionsProvider registers a slash options provider.
// Providers are executed in registration order; the first one that returns handled=true wins.
func RegisterSlashOptionsProvider(p SlashOptionsProvider) {
	if p == nil {
		return
	}
	slashOptionsProviders = append(slashOptionsProviders, p)
}

// SlashSelectedProvider handles Enter on a chosen slash suggestion when the
// command is not executed via exact/prefix dispatch (e.g. fill-only hints).
type SlashSelectedProvider func(m Model, chosen string) (Model, tea.Cmd, bool)

var slashSelectedProviders []SlashSelectedProvider

// RegisterSlashSelectedProvider registers a slash-selected handler.
// Providers run in registration order; the first that returns handled=true wins.
func RegisterSlashSelectedProvider(p SlashSelectedProvider) {
	if p == nil {
		return
	}
	slashSelectedProviders = append(slashSelectedProviders, p)
}

// OverlayKeyProvider can handle key input when an overlay is active.
// When handled==true, the returned model/cmd should be used by ui.
type OverlayKeyProvider func(m Model, key string, msg tea.KeyMsg) (Model, tea.Cmd, bool)

var overlayKeyProviders []OverlayKeyProvider

// RegisterOverlayKeyProvider registers an overlay key provider.
// Providers are executed in registration order; the first one that returns handled=true wins.
func RegisterOverlayKeyProvider(p OverlayKeyProvider) {
	if p == nil {
		return
	}
	overlayKeyProviders = append(overlayKeyProviders, p)
}

// MessageProvider can handle any tea.Msg before ui's default type switch.
// When handled==true, the returned model/cmd should be used.
type MessageProvider func(m Model, msg tea.Msg) (Model, tea.Cmd, bool)

var messageProviders []MessageProvider

// RegisterMessageProvider registers a message provider.
// Providers are executed in registration order; the first one that returns handled=true wins.
func RegisterMessageProvider(p MessageProvider) {
	if p == nil {
		return
	}
	messageProviders = append(messageProviders, p)
}

// OverlayContentProvider can provide overlay content for a model.
// When handled==true, returned content should be used by ui overlay renderer.
type OverlayContentProvider func(m Model) (content string, handled bool)

var overlayContentProviders []OverlayContentProvider

// RegisterOverlayContentProvider registers an overlay content provider.
// Providers are executed in registration order; the first one that returns handled=true wins.
func RegisterOverlayContentProvider(p OverlayContentProvider) {
	if p == nil {
		return
	}
	overlayContentProviders = append(overlayContentProviders, p)
}

// OverlayCloseHook resets feature-specific model fields when an overlay is dismissed
// (Esc or programmatic close). Hooks run after generic overlay chrome is cleared.
type OverlayCloseHook func(m Model) Model

var overlayCloseHooks []OverlayCloseHook

// RegisterOverlayCloseHook registers an overlay-dismiss reset hook.
// Hooks run in registration order; each receives and returns the model by value.
func RegisterOverlayCloseHook(h OverlayCloseHook) {
	if h == nil {
		return
	}
	overlayCloseHooks = append(overlayCloseHooks, h)
}

// TitleBarFragmentProvider supplies the leading title-bar segment (before " | " auto-run),
// e.g. "Local" or "Remote" with an optional label. Providers run in registration order;
// the first that returns ok=true wins. If none return ok, ui uses the default "Local" segment.
type TitleBarFragmentProvider func(m Model) (segment string, ok bool)

var titleBarFragmentProviders []TitleBarFragmentProvider

// RegisterTitleBarFragmentProvider registers a title-bar leading-segment provider.
func RegisterTitleBarFragmentProvider(p TitleBarFragmentProvider) {
	if p == nil {
		return
	}
	titleBarFragmentProviders = append(titleBarFragmentProviders, p)
}
