package approvalexec

import (
	"testing"

	"delve-shell/internal/agent"
	"delve-shell/internal/approvalflow"
	"delve-shell/internal/approvalview"
)

func TestOutcomeForDecision_sensitiveRefuse(t *testing.T) {
	t.Helper()
	o, ok := OutcomeForDecision(approvalflow.DecisionSensitiveRefuse, nil, &agent.SensitiveConfirmationRequest{})
	if !ok || !o.HasSensitiveSend || o.SensitiveChoice != agent.SensitiveRefuse || o.LinesKind != approvalview.DecisionSensitiveRefuse {
		t.Fatalf("unexpected outcome: %+v", o)
	}
}

func TestOutcomeForDecision_copyRequiresPending(t *testing.T) {
	t.Helper()
	_, ok := OutcomeForDecision(approvalflow.DecisionCopy, nil, nil)
	if ok {
		t.Fatal("expected false without pending approval")
	}
}

func TestOutcomeForDecision_copy(t *testing.T) {
	t.Helper()
	p := &agent.ApprovalRequest{Command: "echo hi"}
	o, ok := OutcomeForDecision(approvalflow.DecisionCopy, p, nil)
	if !ok || !o.DoCopyWorkflow || o.CopyCommand != "echo hi" || !o.HasApprovalSend || !o.ApprovalResponse.CopyRequested {
		t.Fatalf("unexpected outcome: %+v", o)
	}
}
