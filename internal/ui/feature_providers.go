package ui

import (
	tea "github.com/charmbracelet/bubbletea"

	"delve-shell/internal/inputlifecycletype"
	"delve-shell/internal/slashdispatch"
	"delve-shell/internal/slashreg"
	"delve-shell/internal/uiregistry"
)

// SlashOptionsProvider can provide slash suggestion options for a given input.
// When handled==true, the returned options should override the default ui logic.
//
// Registration is stored in [uiregistry] so feature packages do not couple to ui.Model.
type SlashOptionsProvider func(inputVal string, lang string) (opts []SlashOption, handled bool)

// RegisterSlashOptionsProvider registers a slash options provider.
// Providers are executed in registration order; the first one that returns handled=true wins.
func RegisterSlashOptionsProvider(p SlashOptionsProvider) {
	if p == nil {
		return
	}
	uiregistry.RegisterSlashOptionsProvider(func(inputVal string, lang string) ([]uiregistry.SlashOption, bool) {
		opts, handled := p(inputVal, lang)
		if !handled || len(opts) == 0 {
			return nil, handled
		}
		out := make([]uiregistry.SlashOption, 0, len(opts))
		for _, o := range opts {
			out = append(out, uiregistry.SlashOption{Cmd: o.Cmd, Desc: o.Desc})
		}
		return out, true
	})
}

// RegisterRootSlashOptionProvider registers a provider for top-level slash options.
// Providers are concatenated in registration order.
func RegisterRootSlashOptionProvider(p func(lang string) []SlashOption) {
	if p == nil {
		return
	}
	uiregistry.RegisterRootSlashOptionProvider(func(lang string) []uiregistry.SlashOption {
		raw := p(lang)
		out := make([]uiregistry.SlashOption, 0, len(raw))
		for _, o := range raw {
			out = append(out, uiregistry.SlashOption{Cmd: o.Cmd, Desc: o.Desc})
		}
		return out
	})
}

// SlashSelectedProvider handles Enter on a chosen slash suggestion when the
// command is not executed via exact/prefix dispatch (e.g. fill-only hints).
type SlashSelectedProvider func(m Model, chosen string) (Model, tea.Cmd, bool)

// RegisterSlashSelectedProvider registers a slash-selected handler.
// Providers run in registration order; the first that returns handled=true wins.
func RegisterSlashSelectedProvider(p SlashSelectedProvider) {
	if p == nil {
		return
	}
	slashRuntime.RegisterSelected(slashdispatch.SelectedProvider[Model, tea.Cmd](p))
}

// OverlayKeyProvider can handle key input when an overlay is active.
// When handled==true, the returned model/cmd should be used by ui.
type OverlayKeyProvider func(m Model, key string, msg tea.KeyMsg) (Model, tea.Cmd, bool)

var overlayKeyProviderChain = slashreg.NewProviderChain[OverlayKeyProvider]()

// RegisterOverlayKeyProvider registers an overlay key provider.
// Providers are executed in registration order; the first one that returns handled=true wins.
func RegisterOverlayKeyProvider(p OverlayKeyProvider) {
	if p == nil {
		return
	}
	overlayKeyProviderChain.Add(p, func(x OverlayKeyProvider) bool { return x == nil })
}

// StateEventProvider handles non-overlay UI state synchronization events before the default update switch.
// This is intended for global state mirroring, not arbitrary feature business logic.
type StateEventProvider func(m Model, msg tea.Msg) (Model, tea.Cmd, bool)

var stateEventProviderChain = slashreg.NewProviderChain[StateEventProvider]()

// RegisterStateEventProvider registers a state event provider.
func RegisterStateEventProvider(p StateEventProvider) {
	if p == nil {
		return
	}
	stateEventProviderChain.Add(p, func(x StateEventProvider) bool { return x == nil })
}

// OverlayEventProvider handles asynchronous overlay events for the active overlay feature.
// Providers should inspect m.Overlay.Key and return handled=true only for their own overlay key.
type OverlayEventProvider func(m Model, msg tea.Msg) (Model, tea.Cmd, bool)

var overlayEventProviderChain = slashreg.NewProviderChain[OverlayEventProvider]()

// RegisterOverlayEventProvider registers an overlay event provider.
func RegisterOverlayEventProvider(p OverlayEventProvider) {
	if p == nil {
		return
	}
	overlayEventProviderChain.Add(p, func(x OverlayEventProvider) bool { return x == nil })
}

// OverlayContentProvider can provide overlay content for a model.
// When handled==true, returned content should be used by ui overlay renderer.
type OverlayContentProvider func(m Model) (content string, handled bool)

var overlayContentProviderChain = slashreg.NewProviderChain[OverlayContentProvider]()

// RegisterOverlayContentProvider registers an overlay content provider.
// Providers are executed in registration order; the first one that returns handled=true wins.
func RegisterOverlayContentProvider(p OverlayContentProvider) {
	if p == nil {
		return
	}
	overlayContentProviderChain.Add(p, func(x OverlayContentProvider) bool { return x == nil })
}

// OverlayOpenRequest describes a structured request to open an overlay feature.
type OverlayOpenRequest struct {
	Key     string
	Params  map[string]string
	Title   string
	Content string
}

// OverlayOpenProvider handles a structured overlay-open request.
type OverlayOpenProvider func(m Model, req OverlayOpenRequest) (Model, tea.Cmd, bool)

var overlayOpenProviderChain = slashreg.NewProviderChain[OverlayOpenProvider]()

// RegisterOverlayOpenProvider registers an overlay-open provider.
func RegisterOverlayOpenProvider(p OverlayOpenProvider) {
	if p == nil {
		return
	}
	overlayOpenProviderChain.Add(p, func(x OverlayOpenProvider) bool { return x == nil })
}

// OverlayFeature groups the standard overlay-related integration points for one feature.
// Features can register any subset that they need.
type OverlayFeature struct {
	Open    OverlayOpenProvider
	Key     OverlayKeyProvider
	Content OverlayContentProvider
	Event   OverlayEventProvider
	Close   OverlayCloseHook
	Startup StartupOverlayProvider
}

// RegisterOverlayFeature registers a bundled overlay feature contract.
func RegisterOverlayFeature(f OverlayFeature) {
	if f.Open != nil {
		RegisterOverlayOpenProvider(f.Open)
	}
	if f.Key != nil {
		RegisterOverlayKeyProvider(f.Key)
	}
	if f.Event != nil {
		RegisterOverlayEventProvider(f.Event)
	}
	if f.Content != nil {
		RegisterOverlayContentProvider(f.Content)
	}
	if f.Close != nil {
		RegisterOverlayCloseHook(f.Close)
	}
	if f.Startup != nil {
		RegisterStartupOverlayProvider(f.Startup)
	}
}

// OverlayCloseHook resets feature-specific model fields when an overlay is dismissed
// (Esc or programmatic close). Hooks run after generic overlay chrome is cleared.
type OverlayCloseHook func(m Model, activeKey string) Model

var overlayCloseHookChain = slashreg.NewProviderChain[OverlayCloseHook]()

// RegisterOverlayCloseHook registers an overlay-dismiss reset hook.
// Hooks run in registration order; each receives and returns the model by value.
func RegisterOverlayCloseHook(h OverlayCloseHook) {
	if h == nil {
		return
	}
	overlayCloseHookChain.Add(h, func(x OverlayCloseHook) bool { return x == nil })
}

// TitleBarFragmentProvider supplies the leading title-bar segment (before " | " auto-run),
// e.g. "Local" or "Remote" with an optional label. Providers run in registration order;
// the first that returns ok=true wins. If none return ok, ui uses the default "Local" segment.
type TitleBarFragmentProvider func(m Model) (segment string, ok bool)

var titleBarFragmentProviderChain = slashreg.NewProviderChain[TitleBarFragmentProvider]()

// RegisterTitleBarFragmentProvider registers a title-bar leading-segment provider.
func RegisterTitleBarFragmentProvider(p TitleBarFragmentProvider) {
	if p == nil {
		return
	}
	titleBarFragmentProviderChain.Add(p, func(x TitleBarFragmentProvider) bool { return x == nil })
}

// StartupOverlayProvider can open a feature overlay on startup after first layout.
// Providers run in registration order; the first one that returns handled=true wins.
type StartupOverlayProvider func(m Model) (Model, tea.Cmd, bool)

var startupOverlayProviderChain = slashreg.NewProviderChain[StartupOverlayProvider]()

// RegisterStartupOverlayProvider registers a startup overlay provider.
func RegisterStartupOverlayProvider(p StartupOverlayProvider) {
	if p == nil {
		return
	}
	startupOverlayProviderChain.Add(p, func(x StartupOverlayProvider) bool { return x == nil })
}

// SlashExecutionRequest is the normalized slash work item exposed to feature packages.
type SlashExecutionRequest struct {
	RawText       string
	InputLine     string
	SelectedIndex int
	CommandSender CommandSender
}

// SlashExecutionProvider handles slash execution outside the ui package.
type SlashExecutionProvider func(req SlashExecutionRequest) (inputlifecycletype.ProcessResult, bool, error)

var slashExecutionProviderChain = slashreg.NewProviderChain[SlashExecutionProvider]()

// RegisterSlashExecutionProvider registers a slash execution provider.
func RegisterSlashExecutionProvider(p SlashExecutionProvider) {
	if p == nil {
		return
	}
	slashExecutionProviderChain.Add(p, func(x SlashExecutionProvider) bool { return x == nil })
}
