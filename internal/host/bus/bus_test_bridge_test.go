package bus

import (
	"testing"
	"time"

	"delve-shell/internal/hil/types"
	"delve-shell/internal/input/lifecycletype"
	"delve-shell/internal/remote/auth"
)

func TestBridgeInputs_Submit(t *testing.T) {
	stop := make(chan struct{})
	defer close(stop)

	b := New(8)
	in := NewInputPorts()
	BridgeInputs(stop, b, in)

	in.SubmissionChan <- inputlifecycletype.InputSubmission{
		Kind:    inputlifecycletype.SubmissionChat,
		Source:  inputlifecycletype.SourceProgrammatic,
		RawText: "hello",
	}
	ev := mustRecvEvent(t, b.Events())
	if ev.Kind != KindUserChatSubmitted || ev.UserText != "hello" {
		t.Fatalf("unexpected event: %+v", ev)
	}
}

func TestBridgeInputs_ConfigUpdated(t *testing.T) {
	stop := make(chan struct{})
	defer close(stop)

	b := New(8)
	in := NewInputPorts()
	BridgeInputs(stop, b, in)

	in.ConfigUpdatedChan <- struct{}{}
	ev := mustRecvEvent(t, b.Events())
	if ev.Kind != KindConfigUpdated {
		t.Fatalf("unexpected event: %+v", ev)
	}
}

func TestBridgeInputs_Cancel(t *testing.T) {
	stop := make(chan struct{})
	defer close(stop)

	b := New(8)
	in := NewInputPorts()
	BridgeInputs(stop, b, in)

	in.CancelRequestChan <- struct{}{}
	ev := mustRecvEvent(t, b.Events())
	if ev.Kind != KindCancelRequested {
		t.Fatalf("unexpected event: %+v", ev)
	}
}

func TestBridgeInputs_ExecDirect(t *testing.T) {
	stop := make(chan struct{})
	defer close(stop)

	b := New(8)
	in := NewInputPorts()
	BridgeInputs(stop, b, in)

	in.ExecDirectChan <- "uname -a"
	ev := mustRecvEvent(t, b.Events())
	if ev.Kind != KindExecDirectRequested || ev.Command != "uname -a" {
		t.Fatalf("unexpected event: %+v", ev)
	}
}

func TestBridgeInputs_RemoteOn(t *testing.T) {
	stop := make(chan struct{})
	defer close(stop)

	b := New(8)
	in := NewInputPorts()
	BridgeInputs(stop, b, in)

	in.RemoteOnChan <- "prod"
	ev := mustRecvEvent(t, b.Events())
	if ev.Kind != KindRemoteOnRequested || ev.RemoteTarget != "prod" {
		t.Fatalf("unexpected event: %+v", ev)
	}
}

func TestBridgeInputs_RemoteOff(t *testing.T) {
	stop := make(chan struct{})
	defer close(stop)

	b := New(8)
	in := NewInputPorts()
	BridgeInputs(stop, b, in)

	in.RemoteOffChan <- struct{}{}
	ev := mustRecvEvent(t, b.Events())
	if ev.Kind != KindRemoteOffRequested {
		t.Fatalf("unexpected event: %+v", ev)
	}
}

func TestBridgeInputs_RemoteAuthResponse(t *testing.T) {
	stop := make(chan struct{})
	defer close(stop)

	b := New(8)
	in := NewInputPorts()
	BridgeInputs(stop, b, in)

	resp := remoteauth.Response{
		Target:   "root@1.2.3.4",
		Username: "root",
		Kind:     "password",
		Password: "secret",
	}
	in.RemoteAuthRespChan <- resp
	ev := mustRecvEvent(t, b.Events())
	if ev.Kind != KindRemoteAuthResponseSubmitted {
		t.Fatalf("unexpected event kind: %+v", ev)
	}
	if ev.RemoteAuthResponse.Target != resp.Target || ev.RemoteAuthResponse.Kind != resp.Kind {
		t.Fatalf("unexpected auth payload: %+v", ev.RemoteAuthResponse)
	}
}

func TestBridgeInputs_AgentUI(t *testing.T) {
	stop := make(chan struct{})
	defer close(stop)

	b := New(8)
	in := NewInputPorts()
	BridgeInputs(stop, b, in)

	payload := map[string]string{"k": "v"}
	in.AgentUIChan <- payload
	ev := mustRecvEvent(t, b.Events())
	if ev.Kind != KindAgentUnknown {
		t.Fatalf("unexpected event kind: %+v", ev)
	}
	m, ok := ev.AgentUI.(map[string]string)
	if !ok || m["k"] != "v" {
		t.Fatalf("unexpected agent payload: %#v", ev.AgentUI)
	}
}

func TestBridgeInputs_AgentUI_Approval(t *testing.T) {
	stop := make(chan struct{})
	defer close(stop)
	b := New(8)
	in := NewInputPorts()
	BridgeInputs(stop, b, in)
	req := &hiltypes.ApprovalRequest{Command: "ls"}
	in.AgentUIChan <- req
	ev := mustRecvEvent(t, b.Events())
	if ev.Kind != KindApprovalRequested || ev.Approval != req {
		t.Fatalf("unexpected event: %+v", ev)
	}
}

func TestBridgeInputs_AgentUI_ExecEvent(t *testing.T) {
	stop := make(chan struct{})
	defer close(stop)
	b := New(8)
	in := NewInputPorts()
	BridgeInputs(stop, b, in)
	ex := hiltypes.ExecEvent{Command: "echo", Allowed: true, Result: "hi"}
	in.AgentUIChan <- ex
	ev := mustRecvEvent(t, b.Events())
	if ev.Kind != KindAgentExecEvent || ev.AgentExec.Command != "echo" {
		t.Fatalf("unexpected event: %+v", ev)
	}
}

func TestBridgeInputs_Stop(t *testing.T) {
	stop := make(chan struct{})
	b := New(8)
	in := NewInputPorts()
	BridgeInputs(stop, b, in)
	close(stop)

	time.Sleep(30 * time.Millisecond)

	select {
	case in.SubmissionChan <- inputlifecycletype.InputSubmission{
		Kind:    inputlifecycletype.SubmissionChat,
		Source:  inputlifecycletype.SourceProgrammatic,
		RawText: "ignored",
	}:
	default:
	}
}
