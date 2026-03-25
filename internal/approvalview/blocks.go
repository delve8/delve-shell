package approvalview

import (
	"strings"

	"delve-shell/internal/i18n"
	"delve-shell/internal/uivm"
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
	lang string,
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
			{Kind: LineHeader, Text: i18n.T(lang, i18n.KeySensitivePrompt)},
			{Kind: LineExec, Text: w(pendingSensitive.Command)},
			{Kind: LineSuggest, Text: i18n.T(lang, i18n.KeySensitiveChoice1)},
			{Kind: LineSuggest, Text: i18n.T(lang, i18n.KeySensitiveChoice2)},
			{Kind: LineSuggest, Text: i18n.T(lang, i18n.KeySensitiveChoice3)},
		}
		return lines, true
	}

	if pending == nil {
		return nil, false
	}

	lines := []Line{
		{Kind: LineHeader, Text: i18n.T(lang, i18n.KeyApprovalPrompt)},
	}
	if sn := strings.TrimSpace(pending.SkillName); sn != "" {
		lines = append(lines, Line{Kind: LineSuggest, Text: w(i18n.Tf(lang, i18n.KeySkillLine, sn))})
	}
	switch pending.RiskLevel {
	case "read_only":
		lines = append(lines, Line{Kind: LineRiskReadOnly, Text: w("[" + i18n.T(lang, i18n.KeyRiskReadOnly) + "] " + pending.Command)})
	case "low":
		lines = append(lines, Line{Kind: LineRiskLow, Text: w("[" + i18n.T(lang, i18n.KeyRiskLow) + "] " + pending.Command)})
	case "high":
		lines = append(lines, Line{Kind: LineRiskHigh, Text: w("[" + i18n.T(lang, i18n.KeyRiskHigh) + "] " + pending.Command)})
	default:
		lines = append(lines, Line{Kind: LineExec, Text: w(pending.Command)})
	}
	if pending.Summary != "" {
		lines = append(lines, Line{Kind: LineSuggest, Text: i18n.T(lang, i18n.KeyApprovalSummary) + " " + pending.Summary})
	}
	if pending.Reason != "" {
		lines = append(lines, Line{Kind: LineSuggest, Text: i18n.T(lang, i18n.KeyApprovalWhy) + " " + pending.Reason})
	}
	return lines, true
}

// BuildDecision returns ordered lines for persisted decision summary blocks.
func BuildDecision(
	lang string,
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
		label := i18n.T(lang, i18n.KeySensitiveChoice1)
		switch decision {
		case DecisionSensitiveRunStore:
			label = i18n.T(lang, i18n.KeySensitiveChoice2)
		case DecisionSensitiveRunNoStore:
			label = i18n.T(lang, i18n.KeySensitiveChoice3)
		}
		return []Line{
			{Kind: LineHeader, Text: i18n.T(lang, i18n.KeySensitivePrompt)},
			{Kind: LineExec, Text: pendingSensitive.Command},
			{Kind: LineSuggest, Text: label},
		}, true
	}

	if pending == nil {
		return nil, false
	}
	base, _ := Build(lang, 0, pending, nil, nil)
	switch decision {
	case DecisionApprove:
		base = append(base, Line{Kind: LineSuggest, Text: i18n.T(lang, i18n.KeyApprovalDecisionApproved)})
	case DecisionReject:
		base = append(base, Line{Kind: LineSuggest, Text: i18n.T(lang, i18n.KeyApprovalDecisionRejected)})
	case DecisionDismiss:
		base = append(base, Line{Kind: LineSuggest, Text: i18n.T(lang, i18n.KeyChoiceDismiss)})
	}
	for i := range base {
		base[i].Text = w(base[i].Text)
	}
	return base, true
}
