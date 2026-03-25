package uipresenter

import (
	"errors"
	"testing"

	tea "github.com/charmbracelet/bubbletea"

	"delve-shell/internal/hiltypes"
	"delve-shell/internal/remote"
	"delve-shell/internal/ui"
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

func TestPresenter_ConfigAndSession(t *testing.T) {
	var r recordSender
	p := New(&r)
	p.ConfigReloaded()
	p.SessionSwitched()
	if len(r.msgs) != 2 {
		t.Fatalf("want 2 msgs, got %d", len(r.msgs))
	}
	if _, ok := r.msgs[0].(ui.TranscriptAppendMsg); !ok {
		t.Fatalf("first msg type %T", r.msgs[0])
	}
	if _, ok := r.msgs[1].(ui.TranscriptAppendMsg); !ok {
		t.Fatalf("second msg type %T", r.msgs[1])
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

func TestPresenter_DispatchAgentUI(t *testing.T) {
	var r recordSender
	p := New(&r)

	p.DispatchAgentUI(&hiltypes.ApprovalRequest{Command: "ls", ResponseCh: make(chan hiltypes.ApprovalResponse, 1)})
	p.DispatchAgentUI(&hiltypes.SensitiveConfirmationRequest{Command: "cat", ResponseCh: make(chan hiltypes.SensitiveChoice, 1)})
	p.DispatchAgentUI(hiltypes.ExecEvent{Command: "x", Allowed: true, Result: "ok", Sensitive: false, Suggested: false})

	if len(r.msgs) != 3 {
		t.Fatalf("want 3 msgs, got %d", len(r.msgs))
	}
	if ar, ok := r.msgs[0].(ui.ChoiceCardShowMsg); !ok || ar.PendingApproval == nil || ar.PendingApproval.Command != "ls" {
		t.Fatalf("msg0 %T %+v", r.msgs[0], r.msgs[0])
	}
	if sr, ok := r.msgs[1].(ui.ChoiceCardShowMsg); !ok || sr.PendingSensitive == nil || sr.PendingSensitive.Command != "cat" {
		t.Fatalf("msg1 %T %+v", r.msgs[1], r.msgs[1])
	}
	if _, ok := r.msgs[2].(ui.TranscriptAppendMsg); !ok {
		t.Fatalf("msg2 type %T %+v", r.msgs[2], r.msgs[2])
	}
}

func TestPresenter_RemoteAndOverlays(t *testing.T) {
	var r recordSender
	p := New(&r)
	p.RemoteStatus(true, "dev")
	p.RemoteConnectDone(false, "", "nope")
	p.RemoteAuthPrompt("h", "e", false)
	p.OverlayShow("t", "c")
	p.OverlayClose()
	if len(r.msgs) != 5 {
		t.Fatalf("want 5 msgs, got %d", len(r.msgs))
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
