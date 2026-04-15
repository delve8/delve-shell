package ui

import (
	"strings"

	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/bubbles/textarea"
	tea "github.com/charmbracelet/bubbletea"

	"delve-shell/internal/hil/approvalview"
	"delve-shell/internal/i18n"
	"delve-shell/internal/teakey"
	"delve-shell/internal/ui/uivm"
)

func (m *Model) startApprovalGuidanceInput() tea.Cmd {
	if m.ChoiceCard.pending == nil {
		return nil
	}
	guide := textarea.New()
	guide.Placeholder = i18n.T(i18n.KeyApprovalGuidancePlaceholder)
	guide.Prompt = "│ "
	guide.ShowLineNumbers = false
	guide.KeyMap.InsertNewline = key.NewBinding(
		key.WithKeys(teakey.ShiftEnter),
		key.WithHelp(teakey.ShiftEnter, "new line"),
	)
	guide.CharLimit = 0
	guide.SetHeight(inputTextareaMinHeight)
	if m.layout.Width > minInputLayoutWidth {
		guide.SetWidth(m.layout.Width - minInputLayoutWidth)
	} else {
		guide.SetWidth(defaultWidth - 4)
	}
	guide.FocusedStyle.Prompt = inputPromptStyle
	guide.FocusedStyle.Text = inputTextStyle
	guide.FocusedStyle.Placeholder = inputPlaceholderStyle
	guide.BlurredStyle.Prompt = inputPromptStyle
	guide.BlurredStyle.Text = inputTextStyle
	guide.BlurredStyle.Placeholder = inputPlaceholderStyle
	guide.Cursor.Style = inputCursorStyle
	guide.Focus()
	m.Input.Blur()
	m.ChoiceCard.approvalGuidance = &ApprovalGuidanceState{Input: guide}
	m.syncApprovalGuidanceHeight()
	return guide.Focus()
}

func (m *Model) syncApprovalGuidanceHeight() {
	if m.ChoiceCard.approvalGuidance == nil {
		return
	}
	p := &m.ChoiceCard.approvalGuidance.Input
	target := inputTextareaMinHeight
	if p.LineCount() > 1 {
		target = inputTextareaMaxHeight
	}
	if p.Height() != target {
		p.SetHeight(target)
	}
}

func (m *Model) cancelApprovalGuidanceInput() tea.Cmd {
	m.ChoiceCard.approvalGuidance = nil
	m.syncInputHeight()
	if m.ChoiceCard.pending != nil {
		return m.Input.Focus()
	}
	return nil
}

func (m *Model) submitApprovalGuidanceInput(text string) (*Model, tea.Cmd) {
	if m.ChoiceCard.pending == nil {
		return m, nil
	}
	trimmed := strings.TrimSpace(text)
	if trimmed == "" {
		if m.ChoiceCard.approvalGuidance != nil {
			m.ChoiceCard.approvalGuidance.submitFeedback = i18n.T(i18n.KeyApprovalGuidanceEmpty)
		}
		return m, nil
	}

	m.appendDecisionLines(approvalview.DecisionGuided)
	m.appendSemanticTranscriptLines(uivm.Line{
		Kind: uivm.LineSystemSuggest,
		Text: i18n.T(i18n.KeyApprovalUserGuidance) + " " + trimmed,
	})
	if m.ChoiceCard.pending.Respond != nil {
		m.ChoiceCard.pending.Respond(uivm.ApprovalResponse{Guidance: trimmed})
	}
	m.ChoiceCard.approvalGuidance = nil
	m.ChoiceCard.pending = nil
	m.Interaction.ChoiceIndex = 0
	m.syncInputHeight()
	return m, m.printTranscriptCmd(false)
}

func (m *Model) handleApprovalGuidanceKeyMsg(msg tea.KeyMsg) (*Model, tea.Cmd) {
	keyStr := msg.String()
	if m.ChoiceCard.approvalGuidance == nil {
		return m, nil
	}

	switch keyStr {
	case teakey.Esc:
		return m, m.cancelApprovalGuidanceInput()
	case teakey.Enter:
		return m.submitApprovalGuidanceInput(m.ChoiceCard.approvalGuidance.Input.Value())
	default:
		state := m.ChoiceCard.approvalGuidance
		if key.Matches(msg, state.Input.KeyMap.InsertNewline) && state.Input.LineCount() == 1 {
			state.Input.SetHeight(inputTextareaMaxHeight)
		}
		var cmd tea.Cmd
		state.Input, cmd = state.Input.Update(msg)
		if strings.TrimSpace(state.Input.Value()) != "" {
			state.submitFeedback = ""
		}
		m.ChoiceCard.approvalGuidance = state
		m.syncApprovalGuidanceHeight()
		return m, cmd
	}
}
