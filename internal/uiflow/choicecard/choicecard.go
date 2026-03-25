package choicecard

import (
	"delve-shell/internal/approvalflow"
	"delve-shell/internal/approvalview"
	"delve-shell/internal/uivm"
)

type EvalResult = approvalflow.Result
type Decision = approvalflow.Decision
type DecisionKind = approvalview.DecisionKind
type Line = approvalview.Line
type LineKind = approvalview.LineKind
type ChoiceOption = approvalview.ChoiceOption

const (
	DecisionApprove             = approvalflow.DecisionApprove
	DecisionReject              = approvalflow.DecisionReject
	DecisionCopy                = approvalflow.DecisionCopy
	DecisionDismiss             = approvalflow.DecisionDismiss
	DecisionSensitiveRefuse     = approvalflow.DecisionSensitiveRefuse
	DecisionSensitiveRunStore   = approvalflow.DecisionSensitiveRunStore
	DecisionSensitiveRunNoStore = approvalflow.DecisionSensitiveRunNoStore

	DecisionKindApprove             = approvalview.DecisionApprove
	DecisionKindReject              = approvalview.DecisionReject
	DecisionKindDismiss             = approvalview.DecisionDismiss
	DecisionKindSensitiveRefuse     = approvalview.DecisionSensitiveRefuse
	DecisionKindSensitiveRunStore   = approvalview.DecisionSensitiveRunStore
	DecisionKindSensitiveRunNoStore = approvalview.DecisionSensitiveRunNoStore

	LineHeader       = approvalview.LineHeader
	LineExec         = approvalview.LineExec
	LineSuggest      = approvalview.LineSuggest
	LineRiskReadOnly = approvalview.LineRiskReadOnly
	LineRiskLow      = approvalview.LineRiskLow
	LineRiskHigh     = approvalview.LineRiskHigh
)

func EvaluateKey(
	key string,
	hasPending bool,
	hasPendingSensitive bool,
	allowlistAutoRunEnabled bool,
	choiceIndex int,
	choiceCount int,
) EvalResult {
	return approvalflow.Evaluate(key, hasPending, hasPendingSensitive, allowlistAutoRunEnabled, choiceIndex, choiceCount)
}

func BuildPendingLines(
	lang string,
	width int,
	pending *uivm.PendingApproval,
	pendingSensitive *uivm.PendingSensitive,
	wrap func(string, int) string,
) ([]Line, bool) {
	return approvalview.Build(lang, width, pending, pendingSensitive, wrap)
}

func BuildDecisionLines(
	lang string,
	width int,
	pending *uivm.PendingApproval,
	pendingSensitive *uivm.PendingSensitive,
	decision DecisionKind,
	wrap func(string, int) string,
) ([]Line, bool) {
	return approvalview.BuildDecision(lang, width, pending, pendingSensitive, decision, wrap)
}

func ChoiceCount(hasPending bool, hasPendingSensitive bool, allowlistAutoRunEnabled bool) int {
	return approvalview.ChoiceCount(hasPending, hasPendingSensitive, allowlistAutoRunEnabled)
}

func ChoiceOptions(lang string, hasPending bool, hasPendingSensitive bool, allowlistAutoRunEnabled bool) []ChoiceOption {
	return approvalview.ChoiceOptions(lang, hasPending, hasPendingSensitive, allowlistAutoRunEnabled)
}

func InputPlaceholder(lang string, hasPending bool, hasPendingSensitive bool, allowlistAutoRunEnabled bool) string {
	return approvalview.InputPlaceholder(lang, hasPending, hasPendingSensitive, allowlistAutoRunEnabled)
}

