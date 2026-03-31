// Package teakey holds string spellings returned by tea.KeyMsg.String() for common non-rune keys
// (charmbracelet/bubbletea keyNames). Compare routed key strings to these constants instead of literals.
//
// Chord names (ShiftEnter, etc.) match github.com/charmbracelet/bubbles/key binding spellings for key.WithKeys.
package teakey

const (
	Enter = "enter"
	Esc   = "esc"
	Tab   = "tab"
	Up    = "up"
	Down  = "down"
)

// Chords for textarea / paste newline bindings (bubbles key package).
const (
	ShiftEnter = "shift+enter"
	AltEnter   = "alt+enter"
	CtrlJ      = "ctrl+j"
)

// InsertNewlineBindingHelp is the first argument to key.WithHelp for the main input InsertNewline binding.
const InsertNewlineBindingHelp = ShiftEnter + " / " + AltEnter + " / " + CtrlJ
