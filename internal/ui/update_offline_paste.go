package ui

import (
	"strings"
	"time"

	"github.com/atotto/clipboard"
	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/bubbles/textarea"
	tea "github.com/charmbracelet/bubbletea"

	"delve-shell/internal/i18n"
)

func (m Model) handleOfflinePasteShowMsg(msg OfflinePasteShowMsg) (Model, tea.Cmd) {
	if msg.Pending == nil || msg.Pending.Respond == nil {
		return m, nil
	}
	m.ChoiceCard.pending = nil
	m.ChoiceCard.pendingSensitive = nil
	lang := m.getLang()
	paste := textarea.New()
	paste.Placeholder = i18n.T(lang, i18n.KeyOfflinePastePlaceholder)
	paste.Prompt = "│ "
	paste.ShowLineNumbers = false
	// Enter submits (handled in handleOfflinePasteKeyMsg); Shift+Enter inserts a newline for manual multi-line input.
	paste.KeyMap.InsertNewline = key.NewBinding(
		key.WithKeys("shift+enter"),
		key.WithHelp("shift+enter", "new line"),
	)
	paste.CharLimit = 0
	paste.SetHeight(inputTextareaMaxHeight)
	if m.layout.Width > minInputLayoutWidth {
		paste.SetWidth(m.layout.Width - minInputLayoutWidth)
	} else {
		paste.SetWidth(defaultWidth - 4)
	}
	paste.FocusedStyle.Prompt = inputPromptStyle
	paste.FocusedStyle.Text = inputTextStyle
	paste.FocusedStyle.Placeholder = inputPlaceholderStyle
	paste.BlurredStyle.Prompt = inputPromptStyle
	paste.BlurredStyle.Text = inputTextStyle
	paste.BlurredStyle.Placeholder = inputPlaceholderStyle
	paste.Cursor.Style = inputCursorStyle
	paste.Focus()
	m.Input.Blur()
	m.ChoiceCard.offlinePaste = &OfflinePasteState{
		Command:   msg.Pending.Command,
		Reason:    msg.Pending.Reason,
		RiskLevel: msg.Pending.RiskLevel,
		Paste:     paste,
		Respond:   msg.Pending.Respond,
	}
	m = m.syncOfflinePasteHeight()
	m = m.syncChoiceViewport()
	m, copyCmd := m.offlinePasteWriteCommandToClipboard()
	return m, tea.Batch(paste.Focus(), copyCmd)
}

// offlinePasteWriteCommandToClipboard puts the pending command on the system clipboard and shows
// brief success/failure feedback (cleared after a tick). Called when the offline paste dialog opens.
func (m Model) offlinePasteWriteCommandToClipboard() (Model, tea.Cmd) {
	op := m.ChoiceCard.offlinePaste
	if op == nil {
		return m, nil
	}
	lang := m.getLang()
	if err := clipboard.WriteAll(op.Command); err != nil {
		op.copyFeedback = i18n.T(lang, i18n.KeyOfflinePasteCopyFailed)
	} else {
		op.copyFeedback = i18n.T(lang, i18n.KeySuggestedCopied)
	}
	m.ChoiceCard.offlinePaste = op
	m = m.syncChoiceViewport()
	return m, tea.Tick(2*time.Second, func(time.Time) tea.Msg {
		return offlinePasteCopyAckClearMsg{}
	})
}

func (m Model) syncOfflinePasteHeight() Model {
	if m.ChoiceCard.offlinePaste == nil {
		return m
	}
	p := &m.ChoiceCard.offlinePaste.Paste
	target := inputTextareaMinHeight
	if p.LineCount() > 1 {
		target = inputTextareaMaxHeight
	}
	if p.Height() != target {
		p.SetHeight(target)
	}
	return m
}

// finishOfflinePaste clears offline UI and invokes Respond; refocuses main input.
func (m Model) finishOfflinePaste(text string, cancelled bool) Model {
	if m.ChoiceCard.offlinePaste == nil {
		return m
	}
	resp := m.ChoiceCard.offlinePaste.Respond
	m.ChoiceCard.offlinePaste = nil
	if resp != nil {
		resp(text, cancelled)
	}
	m.Input.Focus()
	return m
}

func (m Model) handleOfflinePasteKeyMsg(msg tea.KeyMsg) (Model, tea.Cmd) {
	keyStr := msg.String()
	if m.ChoiceCard.offlinePaste == nil {
		return m, nil
	}

	switch keyStr {
	case "esc":
		m = m.finishOfflinePaste("", true)
		m = m.syncInputHeight()
		m = m.syncChoiceViewport()
		return m, nil
	case "enter":
		text := strings.TrimSpace(m.ChoiceCard.offlinePaste.Paste.Value())
		m = m.finishOfflinePaste(text, false)
		m = m.syncInputHeight()
		m = m.syncChoiceViewport()
		return m, nil
	default:
		op := m.ChoiceCard.offlinePaste
		if key.Matches(msg, op.Paste.KeyMap.InsertNewline) && op.Paste.LineCount() == 1 {
			op.Paste.SetHeight(inputTextareaMaxHeight)
		}
		var cmd tea.Cmd
		op.Paste, cmd = op.Paste.Update(msg)
		m.ChoiceCard.offlinePaste = op
		m = m.syncOfflinePasteHeight()
		m = m.syncChoiceViewport()
		return m, cmd
	}
}
