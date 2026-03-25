package hostbus

import (
	"sync/atomic"
	"testing"
	"time"

	tea "github.com/charmbracelet/bubbletea"

	"delve-shell/internal/agent/hiltypes"
	"delve-shell/internal/remoteauth"
	"delve-shell/internal/ui"
)

func mustRecvEvent(t *testing.T, ch <-chan Event) Event {
	t.Helper()
	select {
	case e := <-ch:
		return e
	case <-time.After(2 * time.Second):
		t.Fatal("timed out waiting for bus event")
		return Event{}
	}
}

func TestNew_DefaultCapacity(t *testing.T) {
	b := New(0)
	if b == nil {
		t.Fatal("bus is nil")
	}
	if cap(b.events) != 128 {
		t.Fatalf("want default event cap 128, got %d", cap(b.events))
	}
	if cap(b.uiMsgs) != 256 {
		t.Fatalf("want ui cap 256, got %d", cap(b.uiMsgs))
	}
}

func TestPublishAndEvents(t *testing.T) {
	b := New(4)
	ok := b.Publish(Event{Kind: KindUserChatSubmitted, UserText: "hello"})
	if !ok {
		t.Fatal("publish failed unexpectedly")
	}
	ev := mustRecvEvent(t, b.Events())
	if ev.Kind != KindUserChatSubmitted || ev.UserText != "hello" {
		t.Fatalf("unexpected event: %+v", ev)
	}
}

func TestPublishQueueFull(t *testing.T) {
	b := New(1)
	if !b.Publish(Event{Kind: KindConfigUpdated}) {
		t.Fatal("first publish should succeed")
	}
	if b.Publish(Event{Kind: KindConfigUpdated}) {
		t.Fatal("second publish should fail when queue full")
	}
}

func TestPublishBlockingUnblocksAfterReceive(t *testing.T) {
	b := New(1)
	if !b.Publish(Event{Kind: KindConfigUpdated}) {
		t.Fatal("seed publish failed")
	}
	done := make(chan struct{})
	go func() {
		b.PublishBlocking(Event{Kind: KindCancelRequested})
		close(done)
	}()

	select {
	case <-done:
		t.Fatal("publish blocking should not complete before dequeue")
	case <-time.After(80 * time.Millisecond):
	}

	_ = mustRecvEvent(t, b.Events())
	select {
	case <-done:
	case <-time.After(2 * time.Second):
		t.Fatal("publish blocking did not unblock")
	}
}

func TestEnqueueUI_NilMessage(t *testing.T) {
	b := New(2)
	if b.EnqueueUI(nil) {
		t.Fatal("expected false for nil UI message")
	}
}

func TestEnqueueUI_QueueFull(t *testing.T) {
	b := New(2)
	for i := 0; i < cap(b.uiMsgs); i++ {
		if !b.EnqueueUI(ui.SystemNotifyMsg{Text: "x"}) {
			t.Fatalf("enqueue failed unexpectedly at index %d", i)
		}
	}
	if b.EnqueueUI(ui.SystemNotifyMsg{Text: "overflow"}) {
		t.Fatal("enqueue should fail when ui queue full")
	}
}

func TestEnqueueUIBlocking_IgnoresNil(t *testing.T) {
	b := New(2)
	b.EnqueueUIBlocking(nil)
	select {
	case <-b.uiMsgs:
		t.Fatal("nil should not enqueue")
	default:
	}
}

func TestInputPortsDefaults(t *testing.T) {
	in := NewInputPorts()
	if cap(in.SubmitChan) != 8 ||
		cap(in.ConfigUpdatedChan) != 8 ||
		cap(in.CancelRequestChan) != 8 ||
		cap(in.ExecDirectChan) != 8 ||
		cap(in.RemoteOnChan) != 4 ||
		cap(in.RemoteOffChan) != 4 ||
		cap(in.RemoteAuthRespChan) != 4 ||
		cap(in.AgentUIChan) != 64 {
		t.Fatalf("unexpected capacities: %+v", in)
	}
}

func TestBridgeInputs_Submit(t *testing.T) {
	stop := make(chan struct{})
	defer close(stop)

	b := New(8)
	in := NewInputPorts()
	BridgeInputs(stop, b, in)

	in.SubmitChan <- "hello"
	ev := mustRecvEvent(t, b.Events())
	if ev.Kind != KindUserChatSubmitted || ev.UserText != "hello" {
		t.Fatalf("unexpected event: %+v", ev)
	}
}

func TestBridgeInputs_SubmitNewSession(t *testing.T) {
	stop := make(chan struct{})
	defer close(stop)
	b := New(8)
	in := NewInputPorts()
	BridgeInputs(stop, b, in)
	in.SubmitChan <- "/new"
	ev := mustRecvEvent(t, b.Events())
	if ev.Kind != KindSessionNewRequested {
		t.Fatalf("unexpected event: %+v", ev)
	}
}

func TestBridgeInputs_SubmitSwitchSession(t *testing.T) {
	stop := make(chan struct{})
	defer close(stop)
	b := New(8)
	in := NewInputPorts()
	BridgeInputs(stop, b, in)
	in.SubmitChan <- "/sessions  demo-id "
	ev := mustRecvEvent(t, b.Events())
	if ev.Kind != KindSessionSwitchRequested || ev.SessionID != "demo-id" {
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

	// Give bridge goroutine a moment to exit cleanly.
	time.Sleep(30 * time.Millisecond)

	// Sending after stop should not panic and should not necessarily produce events.
	select {
	case in.SubmitChan <- "ignored":
	default:
	}
}

func TestStartUIPump_NoProgram(t *testing.T) {
	stop := make(chan struct{})
	defer close(stop)

	b := New(8)
	var currentP atomic.Pointer[tea.Program]
	StartUIPump(stop, b, &currentP)

	// No active tea program; enqueue should not block or panic.
	if !b.EnqueueUI(ui.SystemNotifyMsg{Text: "hello"}) {
		t.Fatal("enqueue failed unexpectedly")
	}
	time.Sleep(20 * time.Millisecond)
}

func TestStartUIPump_Stop(t *testing.T) {
	stop := make(chan struct{})

	b := New(8)
	var currentP atomic.Pointer[tea.Program]
	StartUIPump(stop, b, &currentP)
	close(stop)
	time.Sleep(20 * time.Millisecond)

	// Queue still accepts data; no receiver guarantee after stop.
	_ = b.EnqueueUI(ui.SystemNotifyMsg{Text: "after-stop"})
}
