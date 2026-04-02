package uipresenter

import (
	"errors"
	"testing"

	tea "github.com/charmbracelet/bubbletea"

	"delve-shell/internal/hil/types"
	"delve-shell/internal/i18n"
	"delve-shell/internal/remote"
	"delve-shell/internal/ui"
	"delve-shell/internal/ui/uivm"
)

type recordSender struct {
	msgs []tea.Msg
}

func (r *recordSender) Send(msg tea.Msg) {
	if msg == nil {
		return
	}
	r.msgs = append(r.msgs, msg)
}

func TestPresenter_ShowHistoryPreviewDialog_emitsOverlayMsg(t *testing.T) {
	var r recordSender
	p := New(&r)
	i18n.SetLang("en")
	p.ShowHistoryPreviewDialog([]uivm.Line{
		{Kind: uivm.LineUser, Text: "hi"},
	}, "abc123")
	if len(r.msgs) != 1 {
		t.Fatalf("want 1 msg, got %d", len(r.msgs))
	}
	ov, ok := r.msgs[0].(ui.HistoryPreviewOverlayMsg)
	if !ok || ov.SessionID != "abc123" || ov.Title == "" || ov.Content == "" {
		t.Fatalf("want HistoryPreviewOverlayMsg, got %#v", r.msgs[0])
	}
}

func TestPresenter_ApplyHistorySwitchBanner_emitsReplace(t *testing.T) {
	var r recordSender
	p := New(&r)
	i18n.SetLang("en")
	p.ApplyHistorySwitchBanner("abc123")
	if len(r.msgs) != 1 {
		t.Fatalf("want 1 msg, got %d", len(r.msgs))
	}
	if _, ok := r.msgs[0].(ui.TranscriptReplaceMsg); !ok {
		t.Fatalf("want TranscriptReplaceMsg, got %T", r.msgs[0])
	}
}

func TestPresenter_Config(t *testing.T) {
	var r recordSender
	p := New(&r)
	p.ConfigReloaded()
	if len(r.msgs) != 1 {
		t.Fatalf("want 1 msg, got %d", len(r.msgs))
	}
	if _, ok := r.msgs[0].(ui.TranscriptAppendMsg); !ok {
		t.Fatalf("msg type %T", r.msgs[0])
	}
}

func TestPresenter_AgentReply(t *testing.T) {
	var r recordSender
	p := New(&r)
	p.AgentReply("hi", nil)
	p.AgentReply("", errors.New("boom"))
	if len(r.msgs) != 2 {
		t.Fatalf("want 2 msgs, got %d", len(r.msgs))
	}
	if _, ok := r.msgs[0].(ui.TranscriptAppendMsg); !ok {
		t.Fatalf("reply0 type %T", r.msgs[0])
	}
	if _, ok := r.msgs[1].(ui.TranscriptAppendMsg); !ok {
		t.Fatalf("reply1 type %T", r.msgs[1])
	}
}

func TestPresenter_CommandExecutedDirect_usesDirectTag(t *testing.T) {
	var r recordSender
	p := New(&r)
	p.CommandExecutedDirect("ls", "out")
	if len(r.msgs) != 1 {
		t.Fatalf("want 1 msg, got %d", len(r.msgs))
	}
	ta, ok := r.msgs[0].(ui.TranscriptAppendMsg)
	if !ok || len(ta.Lines) < 1 {
		t.Fatalf("want TranscriptAppendMsg with lines, got %#v", r.msgs[0])
	}
	if ta.Lines[0].Text != "Run: ls (direct)" {
		t.Fatalf("want direct tag, got %q", ta.Lines[0].Text)
	}
}

func TestPresenter_DispatchAgentUI(t *testing.T) {
	var r recordSender
	p := New(&r)

	p.DispatchAgentUI(&hiltypes.ApprovalRequest{Command: "ls", ResponseCh: make(chan hiltypes.ApprovalResponse, 1)})
	p.DispatchAgentUI(&hiltypes.SensitiveConfirmationRequest{Command: "cat", ResponseCh: make(chan hiltypes.SensitiveChoice, 1)})
	p.DispatchAgentUI(hiltypes.ExecEvent{Command: "x", Allowed: true, Result: "ok", Sensitive: false, Suggested: false})
	p.DispatchAgentUI(hiltypes.CommandExecutionState{Active: true})

	if len(r.msgs) != 4 {
		t.Fatalf("want 4 msgs, got %d", len(r.msgs))
	}
	if ar, ok := r.msgs[0].(ui.ChoiceCardShowMsg); !ok || ar.PendingApproval == nil || ar.PendingApproval.Command != "ls" {
		t.Fatalf("msg0 %T %+v", r.msgs[0], r.msgs[0])
	}
	if sr, ok := r.msgs[1].(ui.ChoiceCardShowMsg); !ok || sr.PendingSensitive == nil || sr.PendingSensitive.Command != "cat" {
		t.Fatalf("msg1 %T %+v", r.msgs[1], r.msgs[1])
	}
	ta, ok := r.msgs[2].(ui.TranscriptAppendMsg)
	if !ok {
		t.Fatalf("msg2 type %T %+v", r.msgs[2], r.msgs[2])
	}
	if len(ta.Lines) < 1 || ta.Lines[0].Text != "Run: x (allowlist)" {
		t.Fatalf("ExecEvent allowlist tag: got %#v", ta.Lines)
	}
	if _, ok := r.msgs[3].(ui.CommandExecutionStateMsg); !ok {
		t.Fatalf("msg3 type %T", r.msgs[3])
	}
}

func TestPresenter_DispatchAgentUI_StreamedExecTail(t *testing.T) {
	var r recordSender
	p := New(&r)
	p.DispatchAgentUI(hiltypes.ExecEvent{Streamed: true, Sensitive: true, Result: "exit_code: 0"})
	if len(r.msgs) != 1 {
		t.Fatalf("want 1 msg, got %d", len(r.msgs))
	}
	ta := r.msgs[0].(ui.TranscriptAppendMsg)
	if len(ta.Lines) < 3 {
		t.Fatalf("lines: %#v", ta.Lines)
	}
	if ta.Lines[0].Text != "Result contains sensitive data." {
		t.Fatalf("want sensitive line, got %#v", ta.Lines[0])
	}
	if ta.Lines[1].Text != "exit_code: 0" {
		t.Fatalf("want tail, got %#v", ta.Lines[1])
	}
}

func TestPresenter_DispatchAgentUI_ExecStream(t *testing.T) {
	var r recordSender
	p := New(&r)
	p.DispatchAgentUI(hiltypes.ExecStreamStart{Command: "c", Allowed: false})
	p.DispatchAgentUI(hiltypes.ExecStreamLine{Line: "hi", Stderr: false})
	p.DispatchAgentUI(hiltypes.ExecStreamLine{Line: "e", Stderr: true})
	if len(r.msgs) != 3 {
		t.Fatalf("want 3 msgs, got %d", len(r.msgs))
	}
	m0 := r.msgs[0].(ui.TranscriptAppendMsg)
	if m0.Lines[0].Text != "Run: c (approved)" {
		t.Fatalf("run line: %#v", m0.Lines[0])
	}
	m1 := r.msgs[1].(ui.TranscriptAppendMsg)
	if m1.Lines[0].Text != "hi" {
		t.Fatalf("stdout line: %#v", m1.Lines[0])
	}
	m2 := r.msgs[2].(ui.TranscriptAppendMsg)
	if m2.Lines[0].Text != "stderr: e" {
		t.Fatalf("stderr line: %#v", m2.Lines[0])
	}
}

func TestPresenter_Remote(t *testing.T) {
	var r recordSender
	p := New(&r)
	p.RemoteStatus(true, "dev", false)
	p.RemoteConnectDone(false, "", "nope")
	p.RemoteAuthPrompt("h", "e", false)
	if len(r.msgs) != 3 {
		t.Fatalf("want 3 msgs, got %d", len(r.msgs))
	}
	if _, ok := r.msgs[0].(remote.ExecutionChangedMsg); !ok {
		t.Fatalf("msg0 type %T", r.msgs[0])
	}
	if _, ok := r.msgs[1].(remote.ConnectDoneMsg); !ok {
		t.Fatalf("msg1 type %T", r.msgs[1])
	}
	if _, ok := r.msgs[2].(remote.AuthPromptMsg); !ok {
		t.Fatalf("msg2 type %T", r.msgs[2])
	}
}

func TestPresenter_RunCompletionCache(t *testing.T) {
	var r recordSender
	p := New(&r)
	p.RunCompletionCache("r1", []string{"a", "b"})
	m := r.msgs[0].(remote.RunCompletionCacheMsg)
	if m.RemoteLabel != "r1" || len(m.Commands) != 2 {
		t.Fatalf("%+v", m)
	}
}

func TestPresenter_NilSenderNoPanic(t *testing.T) {
	p := New(nil)
	p.ConfigReloaded()
	p.DispatchAgentUI(&hiltypes.ApprovalRequest{Command: "x", ResponseCh: make(chan hiltypes.ApprovalResponse, 1)})
}
