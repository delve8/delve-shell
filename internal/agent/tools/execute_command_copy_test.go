package tools

import (
	"context"
	"encoding/json"
	"os"
	"strings"
	"testing"

	hiltypes "delve-shell/internal/hil/types"
	"delve-shell/internal/history"
)

func TestExecuteCommandTool_CopyRequestedStoresSuggestedOnly(t *testing.T) {
	sess, err := history.NewSession()
	if err != nil {
		t.Fatalf("NewSession error: %v", err)
	}
	defer func() {
		_ = sess.Close()
		_ = os.Remove(sess.Path())
	}()

	tool := &ExecuteCommandTool{
		Session: sess,
		RequestApproval: func(command, summary, reason, riskLevel, skillName string, autoHL []hiltypes.AutoApproveHighlightSpan) hiltypes.ApprovalResponse {
			return hiltypes.ApprovalResponse{CopyRequested: true}
		},
		OfflineMode:      func() bool { return false },
		ExecutorProvider: nil,
	}

	out, err := tool.InvokableRun(context.Background(), `{"command":"kubectl get pods","reason":"debug","risk_level":"read_only"}`)
	if err != nil {
		t.Fatalf("InvokableRun error: %v", err)
	}
	if !strings.Contains(out, "copied the command") {
		t.Fatalf("unexpected tool output: %q", out)
	}

	data, err := os.ReadFile(sess.Path())
	if err != nil {
		t.Fatalf("read history: %v", err)
	}
	lines := strings.Split(strings.TrimSpace(string(data)), "\n")
	if len(lines) != 1 {
		t.Fatalf("history lines=%d want 1\n%s", len(lines), string(data))
	}
	var ev history.Event
	if err := json.Unmarshal([]byte(lines[0]), &ev); err != nil {
		t.Fatalf("unmarshal event: %v", err)
	}
	if ev.Type != history.EventTypeCommand {
		t.Fatalf("event type=%q want %q", ev.Type, history.EventTypeCommand)
	}
	var payload map[string]interface{}
	if err := json.Unmarshal(ev.Payload, &payload); err != nil {
		t.Fatalf("unmarshal payload: %v", err)
	}
	if payload["suggested"] != true {
		t.Fatalf("suggested=%v want true", payload["suggested"])
	}
	if payload["approved"] != false {
		t.Fatalf("approved=%v want false", payload["approved"])
	}
}

func TestExecuteCommandTool_GuidanceKeepsTurnAndStoresGuidance(t *testing.T) {
	sess, err := history.NewSession()
	if err != nil {
		t.Fatalf("NewSession error: %v", err)
	}
	defer func() {
		_ = sess.Close()
		_ = os.Remove(sess.Path())
	}()

	tool := &ExecuteCommandTool{
		Session: sess,
		RequestApproval: func(command, summary, reason, riskLevel, skillName string, autoHL []hiltypes.AutoApproveHighlightSpan) hiltypes.ApprovalResponse {
			return hiltypes.ApprovalResponse{Guidance: "check logs first"}
		},
		OfflineMode:      func() bool { return false },
		ExecutorProvider: nil,
	}

	out, err := tool.InvokableRun(context.Background(), `{"command":"systemctl restart app","reason":"debug","risk_level":"high"}`)
	if err != nil {
		t.Fatalf("InvokableRun error: %v", err)
	}
	if !strings.Contains(out, "added guidance: check logs first") {
		t.Fatalf("unexpected tool output: %q", out)
	}

	data, err := os.ReadFile(sess.Path())
	if err != nil {
		t.Fatalf("read history: %v", err)
	}
	lines := strings.Split(strings.TrimSpace(string(data)), "\n")
	if len(lines) != 1 {
		t.Fatalf("history lines=%d want 1\n%s", len(lines), string(data))
	}
	var ev history.Event
	if err := json.Unmarshal([]byte(lines[0]), &ev); err != nil {
		t.Fatalf("unmarshal event: %v", err)
	}
	var payload map[string]interface{}
	if err := json.Unmarshal(ev.Payload, &payload); err != nil {
		t.Fatalf("unmarshal payload: %v", err)
	}
	if payload["guidance"] != "check logs first" {
		t.Fatalf("guidance=%v want %q", payload["guidance"], "check logs first")
	}
}
