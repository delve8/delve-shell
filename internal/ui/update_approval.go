package ui

import (
	"strings"

	"github.com/atotto/clipboard"
	tea "github.com/charmbracelet/bubbletea"

	"delve-shell/internal/hil/approvalflow"
	"delve-shell/internal/hil/approvalview"
	"delve-shell/internal/i18n"
	"delve-shell/internal/teakey"
	"delve-shell/internal/textwrap"
	"delve-shell/internal/ui/uivm"
)

func (m *Model) handlePendingChoiceKey(msg tea.KeyMsg) (*Model, tea.Cmd, bool) {
	// Let the global Esc path run; approvalflow would otherwise handle the key with DecisionNone and swallow it.
	if msg.String() == teakey.Esc {
		return m, nil, false
	}
	cc := approvalview.ChoiceCount(m.ChoiceCard.pending != nil, m.ChoiceCard.pendingSensitive != nil)
	if cc > 0 {
		if m.Interaction.ChoiceIndex < 0 || m.Interaction.ChoiceIndex >= cc {
			m.Interaction.ChoiceIndex = 0
		}
	}
	res := approvalflow.Evaluate(
		msg.String(),
		m.ChoiceCard.pending != nil,
		m.ChoiceCard.pendingSensitive != nil,
		m.Interaction.ChoiceIndex,
		cc,
	)
	if !res.Handled {
		return m, nil, false
	}
	if res.ChoiceChanged {
		m.Interaction.ChoiceIndex = res.ChoiceIndex
		return m, nil, true
	}
	return m.applyApprovalDecision(res.Decision)
}

func (m *Model) appendDecisionLines(decision approvalview.DecisionKind) {
	lines, ok := approvalview.BuildDecision(m.contentWidth(), m.ChoiceCard.pending, m.ChoiceCard.pendingSensitive, decision, textwrap.WrapString)
	if !ok {
		return
	}
	for _, line := range lines {
		rendered := line.Text
		switch line.Kind {
		case approvalview.LineSpacer:
			rendered = ""
		case approvalview.LineHeader:
			rendered = approvalHeaderStyle.Render(line.Text)
		case approvalview.LineExec:
			rendered = execStyle.Render(line.Text)
		case approvalview.LineSuggest:
			switch decision {
			case approvalview.DecisionApprove:
				if line.Text == i18n.T(i18n.KeyApprovalDecisionApproved) {
					rendered = approvalDecisionApprovedStyle.Render(line.Text)
				} else {
					rendered = suggestStyle.Render(line.Text)
				}
			case approvalview.DecisionGuided:
				if line.Text == i18n.T(i18n.KeyApprovalDecisionGuided) {
					rendered = approvalDecisionRejectedStyle.Render(line.Text)
				} else {
					rendered = suggestStyle.Render(line.Text)
				}
			case approvalview.DecisionReject:
				if line.Text == i18n.T(i18n.KeyApprovalDecisionRejected) {
					rendered = approvalDecisionRejectedStyle.Render(line.Text)
				} else {
					rendered = suggestStyle.Render(line.Text)
				}
			case approvalview.DecisionDismiss:
				if line.Text == i18n.T(i18n.KeyChoiceDismiss) {
					rendered = approvalDecisionDismissStyle.Render(line.Text)
				} else {
					rendered = suggestStyle.Render(line.Text)
				}
			case approvalview.DecisionSensitiveRefuse, approvalview.DecisionSensitiveRunStore, approvalview.DecisionSensitiveRunNoStore:
				if strings.HasPrefix(line.Text, i18n.T(i18n.KeySensitiveChoice1)) ||
					strings.HasPrefix(line.Text, i18n.T(i18n.KeySensitiveChoice2)) ||
					strings.HasPrefix(line.Text, i18n.T(i18n.KeySensitiveChoice3)) {
					rendered = suggestHi.Render(line.Text)
				} else {
					rendered = suggestStyle.Render(line.Text)
				}
			default:
				rendered = suggestStyle.Render(line.Text)
			}
		case approvalview.LineRiskReadOnly:
			rendered = riskReadOnlyStyle.Render(line.Text)
		case approvalview.LineRiskLow:
			rendered = riskLowStyle.Render(line.Text)
		case approvalview.LineRiskHigh:
			rendered = riskHighStyle.Render(line.Text)
		case approvalview.LineMetaLabel:
			rendered = metaLabelStyle.Render(line.Text)
		case approvalview.LineMetaDetail:
			rendered = metaDetailStyle.Render(line.Text)
		}
		m.messages = append(m.messages, rendered)
	}
}

func (m *Model) applyApprovalDecision(d approvalflow.Decision) (*Model, tea.Cmd, bool) {
	switch d {
	case approvalflow.DecisionSensitiveRefuse, approvalflow.DecisionSensitiveRunStore, approvalflow.DecisionSensitiveRunNoStore:
		if m.ChoiceCard.pendingSensitive == nil {
			return m, nil, true
		}
		var kind approvalview.DecisionKind
		var choice uivm.SensitiveChoice
		switch d {
		case approvalflow.DecisionSensitiveRunStore:
			kind = approvalview.DecisionSensitiveRunStore
			choice = uivm.SensitiveRunAndStore
		case approvalflow.DecisionSensitiveRunNoStore:
			kind = approvalview.DecisionSensitiveRunNoStore
			choice = uivm.SensitiveRunNoStore
		default:
			kind = approvalview.DecisionSensitiveRefuse
			choice = uivm.SensitiveRefuse
		}
		m.appendDecisionLines(kind)
		if m.ChoiceCard.pendingSensitive.Respond != nil {
			m.ChoiceCard.pendingSensitive.Respond(choice)
		}
		m.ChoiceCard.pendingSensitive = nil
		m.Interaction.ChoiceIndex = 0
		return m, m.printTranscriptCmd(false), true

	case approvalflow.DecisionGuide:
		if m.ChoiceCard.pending == nil {
			return m, nil, true
		}
		return m, m.startApprovalGuidanceInput(), true

	case approvalflow.DecisionApprove, approvalflow.DecisionReject, approvalflow.DecisionDismiss, approvalflow.DecisionCopy:
		if m.ChoiceCard.pending == nil {
			return m, nil, true
		}
		var kind approvalview.DecisionKind
		resp := uivm.ApprovalResponse{Approved: false, CopyRequested: false}
		waitingClear := false
		doCopy := false
		switch d {
		case approvalflow.DecisionApprove:
			kind = approvalview.DecisionApprove
			resp.Approved = true
		case approvalflow.DecisionReject:
			kind = approvalview.DecisionReject
			waitingClear = true
		case approvalflow.DecisionDismiss:
			kind = approvalview.DecisionDismiss
			waitingClear = true
		case approvalflow.DecisionCopy:
			kind = approvalview.DecisionReject
			resp.CopyRequested = true
			doCopy = true
		}
		m.appendDecisionLines(kind)
		if doCopy {
			_ = clipboard.WriteAll(m.ChoiceCard.pending.Command)
			m.appendSuggestedLine(m.ChoiceCard.pending.Command)
			m.appendSemanticTranscriptLines(uivm.Line{Kind: uivm.LineSystemSuggest, Text: i18n.T(i18n.KeySuggestedCopied)})
		}
		if m.ChoiceCard.pending.Respond != nil {
			m.ChoiceCard.pending.Respond(resp)
		}
		m.ChoiceCard.pending = nil
		if waitingClear {
			m.Interaction.WaitingForAI = false
		}
		m.Interaction.ChoiceIndex = 0
		return m, m.printTranscriptCmd(false), true
	default:
		return m, nil, true
	}
}
