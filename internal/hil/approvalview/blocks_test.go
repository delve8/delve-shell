package approvalview

import (
	"strings"
	"testing"

	hiltypes "delve-shell/internal/hil/types"
	"delve-shell/internal/i18n"
	"delve-shell/internal/ui/uivm"
)

func TestBuildSensitive(t *testing.T) {
	i18n.SetLang("en")
	lines, ok := Build(80, nil, &uivm.PendingSensitive{Command: "rm -rf /"}, nil)
	if !ok {
		t.Fatal("expected sensitive block")
	}
	if len(lines) < 3 {
		t.Fatalf("expected multiple lines, got %d", len(lines))
	}
	if lines[0].Kind != LineHeader || !strings.Contains(strings.ToLower(lines[0].Text), "sensitive") {
		t.Fatalf("unexpected header line: %#v", lines[0])
	}
}

func TestBuildApprovalRiskAndSummary(t *testing.T) {
	i18n.SetLang("en")
	lines, ok := Build(80, &uivm.PendingApproval{
		Command:   "kubectl get pods",
		RiskLevel: hiltypes.RiskLevelReadOnly,
		SkillName: "k8s",
		Summary:   "list pods",
		Reason:    "debug",
	}, nil, nil)
	if !ok {
		t.Fatal("expected approval block")
	}
	var hasRisk, hasExecCmd bool
	for _, l := range lines {
		if l.Kind == LineRiskReadOnly {
			hasRisk = true
		}
		if l.Kind == LineExec && strings.Contains(l.Text, "kubectl get pods") {
			hasExecCmd = true
		}
	}
	if !hasRisk {
		t.Fatalf("expected read-only risk line, got %#v", lines)
	}
	if !hasExecCmd {
		t.Fatalf("expected command on separate exec line, got %#v", lines)
	}
}

func TestBuildApprovalAutoApproveReasonLines(t *testing.T) {
	i18n.SetLang("en")
	lines, ok := Build(80, &uivm.PendingApproval{
		Command:   "echo hi",
		RiskLevel: hiltypes.RiskLevelReadOnly,
		AutoApproveHighlight: []hiltypes.AutoApproveHighlightSpan{
			{Kind: hiltypes.AutoApproveHighlightRisk, Reason: "first reason"},
			{Kind: hiltypes.AutoApproveHighlightRisk, Reason: "first reason"},
			{Kind: hiltypes.AutoApproveHighlightRisk, Reason: "second reason"},
		},
	}, nil, nil)
	if !ok {
		t.Fatal("expected approval block")
	}
	var j int = -1
	for i := range lines {
		if lines[i].Kind == LineMetaDetail && strings.Contains(lines[i].Text, "first reason") {
			j = i
			break
		}
	}
	if j < 0 {
		t.Fatalf("expected LineMetaDetail with policy reasons, got %#v", lines)
	}
	wantTitle := i18n.T(i18n.KeyApprovalAutoApprovePolicy)
	if j < 2 || lines[j-1].Kind != LineMetaLabel || lines[j-1].Text != wantTitle {
		t.Fatalf("expected meta label %q before policy detail, got %#v", wantTitle, lines[j-1])
	}
	if lines[j-2].Kind != LineSpacer {
		t.Fatalf("expected spacer before meta sections, got %#v", lines[j-2])
	}
	want := "first reason\nsecond reason"
	if lines[j].Text != want {
		t.Fatalf("reason text: got %q want %q", lines[j].Text, want)
	}
}

func TestBuildApprovalSummaryPurposeMetaSections(t *testing.T) {
	i18n.SetLang("en")
	lines, ok := Build(80, &uivm.PendingApproval{
		Command:   "true",
		RiskLevel: hiltypes.RiskLevelReadOnly,
		Summary:   "do thing",
		Reason:    "because",
	}, nil, nil)
	if !ok {
		t.Fatal("expected approval block")
	}
	var sawSummaryLabel, sawSummaryDetail, sawPurposeLabel, sawPurposeDetail bool
	for _, l := range lines {
		switch {
		case l.Kind == LineMetaLabel && l.Text == i18n.T(i18n.KeyApprovalSummary):
			sawSummaryLabel = true
		case l.Kind == LineMetaDetail && l.Text == "do thing":
			sawSummaryDetail = true
		case l.Kind == LineMetaLabel && l.Text == i18n.T(i18n.KeyApprovalWhy):
			sawPurposeLabel = true
		case l.Kind == LineMetaDetail && l.Text == "because":
			sawPurposeDetail = true
		}
	}
	if !sawSummaryLabel || !sawSummaryDetail || !sawPurposeLabel || !sawPurposeDetail {
		t.Fatalf("expected split Summary/Purpose meta lines, got %#v", lines)
	}
}

func TestBuildApprovalRiskHintBlankLineBeforePurpose(t *testing.T) {
	i18n.SetLang("en")
	lines, ok := Build(80, &uivm.PendingApproval{
		Command:   "kubectl get pods",
		RiskLevel: hiltypes.RiskLevelReadOnly,
		AutoApproveHighlight: []hiltypes.AutoApproveHighlightSpan{
			{Kind: hiltypes.AutoApproveHighlightRisk, Reason: "policy mismatch"},
		},
		Reason: "fix cluster check",
	}, nil, nil)
	if !ok {
		t.Fatal("expected approval block")
	}
	wantPurpose := i18n.T(i18n.KeyApprovalWhy)
	var iPurpose int
	for i := range lines {
		if lines[i].Kind == LineMetaLabel && lines[i].Text == wantPurpose {
			iPurpose = i
			break
		}
	}
	if iPurpose < 2 {
		t.Fatalf("expected Purpose label, got %#v", lines)
	}
	if lines[iPurpose-1].Kind != LineSpacer {
		t.Fatalf("expected blank line before Purpose after Risk Hint, got %#v", lines[iPurpose-1])
	}
}
