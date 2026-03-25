package uipresenter

import (
	"errors"
	"testing"

	tea "github.com/charmbracelet/bubbletea"

	"delve-shell/internal/agent"
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
	if _, ok := r.msgs[0].(ui.ConfigReloadedMsg); !ok {
		t.Fatalf("first msg type %T", r.msgs[0])
	}
	if _, ok := r.msgs[1].(ui.SessionSwitchedMsg); !ok {
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
	m0 := r.msgs[0].(ui.AgentReplyMsg)
	if m0.Reply != "hi" || m0.Err != nil {
		t.Fatalf("reply0: %+v", m0)
	}
	m1 := r.msgs[1].(ui.AgentReplyMsg)
	if m1.Err == nil {
		t.Fatal("expected err")
	}
}

func TestPresenter_DispatchAgentUI(t *testing.T) {
	var r recordSender
	p := New(&r)

	p.DispatchAgentUI(&agent.ApprovalRequest{Command: "ls"})
	p.DispatchAgentUI(&agent.SensitiveConfirmationRequest{Command: "cat"})
	p.DispatchAgentUI(agent.ExecEvent{Command: "x", Allowed: true, Result: "ok", Sensitive: false, Suggested: false})

	if len(r.msgs) != 3 {
		t.Fatalf("want 3 msgs, got %d", len(r.msgs))
	}
	if ar, ok := r.msgs[0].(*agent.ApprovalRequest); !ok || ar.Command != "ls" {
		t.Fatalf("msg0 %T %+v", r.msgs[0], r.msgs[0])
	}
	if sr, ok := r.msgs[1].(*agent.SensitiveConfirmationRequest); !ok || sr.Command != "cat" {
		t.Fatalf("msg1 %T %+v", r.msgs[1], r.msgs[1])
	}
	ce := r.msgs[2].(ui.CommandExecutedMsg)
	if ce.Command != "x" || !ce.Allowed || ce.Direct || ce.Result != "ok" {
		t.Fatalf("msg2 %+v", ce)
	}
}

func TestPresenter_RemoteAndOverlays(t *testing.T) {
	var r recordSender
	p := New(&r)
	p.RemoteStatus(true, "dev")
	p.RemoteConnectDone(false, "", "nope")
	p.RemoteAuthPrompt(ui.RemoteAuthPromptMsg{Target: "h", Err: "e"})
	p.OverlayShow("t", "c")
	p.OverlayClose()
	if len(r.msgs) != 5 {
		t.Fatalf("want 5 msgs, got %d", len(r.msgs))
	}
}

func TestPresenter_RunCompletionCache(t *testing.T) {
	var r recordSender
	p := New(&r)
	p.RunCompletionCache("r1", []string{"a", "b"})
	m := r.msgs[0].(ui.RunCompletionCacheMsg)
	if m.RemoteLabel != "r1" || len(m.Commands) != 2 {
		t.Fatalf("%+v", m)
	}
}

func TestPresenter_NilSenderNoPanic(t *testing.T) {
	p := New(nil)
	p.ConfigReloaded()
	p.DispatchAgentUI(&agent.ApprovalRequest{Command: "x"})
}
