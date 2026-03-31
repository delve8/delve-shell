package ui

import (
	tea "github.com/charmbracelet/bubbletea"

	"delve-shell/internal/input/lifecycletype"
	"delve-shell/internal/slash/reg"
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
			out = append(out, uiregistry.SlashOption{Cmd: o.Cmd, Desc: o.Desc, FillValue: o.FillValue})
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
			out = append(out, uiregistry.SlashOption{Cmd: o.Cmd, Desc: o.Desc, FillValue: o.FillValue})
		}
		return out
	})
}

// OverlayKeyProvider can handle key input when an overlay is active.
// When handled==true, the returned model/cmd should be used by ui.
type OverlayKeyProvider func(m Model, key string, msg tea.KeyMsg) (Model, tea.Cmd, bool)

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

// OverlayContentProvider can provide overlay content for a model.
// When handled==true, returned content should be used by ui overlay renderer.
type OverlayContentProvider func(m Model) (content string, handled bool)

// OverlayOpenRequest describes a structured request to open an overlay feature.
type OverlayOpenRequest struct {
	Key     string
	Params  map[string]string
	Title   string
	Content string
}

// OverlayOpenProvider handles a structured overlay-open request.
type OverlayOpenProvider func(m Model, req OverlayOpenRequest) (Model, tea.Cmd, bool)

// OverlayFeature groups the standard overlay-related integration points for one feature.
// Features register once by stable key and may implement any subset of the lifecycle hooks.
type OverlayFeature struct {
	KeyID   string
	Open    OverlayOpenProvider
	Key     OverlayKeyProvider
	Content OverlayContentProvider
	Event   OverlayEventProvider
	Close   OverlayCloseHook
	Startup StartupOverlayProvider
}

type registeredOverlayFeature struct {
	key     string
	feature OverlayFeature
}

var overlayFeatureRegistry = slashreg.NewProviderChain[registeredOverlayFeature]()

// RegisterOverlayFeature registers a bundled overlay feature contract.
func RegisterOverlayFeature(f OverlayFeature) {
	if f.KeyID == "" {
		return
	}
	overlayFeatureRegistry.Add(registeredOverlayFeature{key: f.KeyID, feature: f}, func(x registeredOverlayFeature) bool { return x.key == "" })
}

// OverlayCloseHook resets feature-specific model fields when an overlay is dismissed
// (Esc or programmatic close).
type OverlayCloseHook func(m Model, activeKey string) Model

// TitleBarFragmentProvider supplies the leading title-bar segment (remote / context label),
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

func overlayFeatures() []registeredOverlayFeature {
	return overlayFeatureRegistry.List()
}

func overlayFeatureByKey(key string) (OverlayFeature, bool) {
	for _, entry := range overlayFeatures() {
		if entry.key == key {
			return entry.feature, true
		}
	}
	return OverlayFeature{}, false
}

// SlashExecutionRequest is the normalized slash work item exposed to feature packages.
type SlashExecutionRequest struct {
	RawText       string
	InputLine     string
	SelectedIndex int
	CommandSender CommandSender
	// OfflineExecutionMode when true: slash handlers should not start skill/remote execution flows that require in-process tools.
	OfflineExecutionMode bool
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
