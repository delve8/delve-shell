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
