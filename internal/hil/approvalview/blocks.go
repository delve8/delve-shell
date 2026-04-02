package approvalview

import (
	"strings"

	"delve-shell/internal/hil/types"
	"delve-shell/internal/i18n"
	"delve-shell/internal/ui/uivm"
)

type LineKind int

const (
	LineHeader LineKind = iota
	LineExec
	LineSuggest
	LineRiskReadOnly
	LineRiskLow
	LineRiskHigh
)

type Line struct {
	Kind LineKind
	Text string
}

type DecisionKind int

const (
	DecisionApprove DecisionKind = iota
	DecisionReject
	DecisionDismiss
	DecisionSensitiveRefuse
	DecisionSensitiveRunStore
	DecisionSensitiveRunNoStore
)

// Build returns ordered approval/sensitive lines for viewport rendering.
func Build(
	width int,
	pending *uivm.PendingApproval,
	pendingSensitive *uivm.PendingSensitive,
	wrap func(string, int) string,
) ([]Line, bool) {
	w := func(s string) string {
		if wrap == nil {
			return s
		}
		if width <= 0 {
			return s
		}
		return wrap(s, width)
	}

	if pendingSensitive != nil {
		lines := []Line{
			{Kind: LineHeader, Text: i18n.T(i18n.KeySensitivePrompt)},
			{Kind: LineExec, Text: w(pendingSensitive.Command)},
			{Kind: LineSuggest, Text: i18n.T(i18n.KeySensitiveChoice1)},
			{Kind: LineSuggest, Text: i18n.T(i18n.KeySensitiveChoice2)},
			{Kind: LineSuggest, Text: i18n.T(i18n.KeySensitiveChoice3)},
		}
		return lines, true
	}

	if pending == nil {
		return nil, false
	}

	lines := []Line{
		{Kind: LineHeader, Text: i18n.T(i18n.KeyApprovalPrompt)},
	}
	if sn := strings.TrimSpace(pending.SkillName); sn != "" {
		lines = append(lines, Line{Kind: LineSuggest, Text: w(i18n.Tf(i18n.KeySkillLine, sn))})
	}
	switch pending.RiskLevel {
	case hiltypes.RiskLevelReadOnly:
		lines = append(lines,
			Line{Kind: LineRiskReadOnly, Text: "[" + i18n.T(i18n.KeyRiskReadOnly) + "]"},
			Line{Kind: LineExec, Text: w(pending.Command)},
		)
	case hiltypes.RiskLevelLow:
		lines = append(lines,
			Line{Kind: LineRiskLow, Text: "[" + i18n.T(i18n.KeyRiskLow) + "]"},
			Line{Kind: LineExec, Text: w(pending.Command)},
		)
	case hiltypes.RiskLevelHigh:
		lines = append(lines,
			Line{Kind: LineRiskHigh, Text: "[" + i18n.T(i18n.KeyRiskHigh) + "]"},
			Line{Kind: LineExec, Text: w(pending.Command)},
		)
	default:
		lines = append(lines, Line{Kind: LineExec, Text: w(pending.Command)})
	}
	if pending.Summary != "" {
		lines = append(lines, Line{Kind: LineSuggest, Text: i18n.T(i18n.KeyApprovalSummary) + " " + pending.Summary})
	}
	if pending.Reason != "" {
		lines = append(lines, Line{Kind: LineSuggest, Text: i18n.T(i18n.KeyApprovalWhy) + " " + pending.Reason})
	}
	return lines, true
}

// BuildDecision returns ordered lines for persisted decision summary blocks.
func BuildDecision(
	width int,
	pending *uivm.PendingApproval,
	pendingSensitive *uivm.PendingSensitive,
	decision DecisionKind,
	wrap func(string, int) string,
) ([]Line, bool) {
	w := func(s string) string {
		if wrap == nil || width <= 0 {
			return s
		}
		return wrap(s, width)
	}

	if pendingSensitive != nil {
		label := i18n.T(i18n.KeySensitiveChoice1)
		switch decision {
		case DecisionSensitiveRunStore:
			label = i18n.T(i18n.KeySensitiveChoice2)
		case DecisionSensitiveRunNoStore:
			label = i18n.T(i18n.KeySensitiveChoice3)
		}
		return []Line{
			{Kind: LineHeader, Text: i18n.T(i18n.KeySensitivePrompt)},
			{Kind: LineExec, Text: pendingSensitive.Command},
			{Kind: LineSuggest, Text: label},
		}, true
	}

	if pending == nil {
		return nil, false
	}
	base, _ := Build(0, pending, nil, nil)
	switch decision {
	case DecisionApprove:
		base = append(base, Line{Kind: LineSuggest, Text: i18n.T(i18n.KeyApprovalDecisionApproved)})
	case DecisionReject:
		base = append(base, Line{Kind: LineSuggest, Text: i18n.T(i18n.KeyApprovalDecisionRejected)})
	case DecisionDismiss:
		base = append(base, Line{Kind: LineSuggest, Text: i18n.T(i18n.KeyChoiceDismiss)})
	}
	for i := range base {
		base[i].Text = w(base[i].Text)
	}
	return base, true
}
