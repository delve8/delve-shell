package interactive

import (
	"testing"
	"time"

	"delve-shell/internal/hostbus"
	"delve-shell/internal/hostnotify"
	"delve-shell/internal/remote"
	"delve-shell/internal/run"
	"delve-shell/internal/ui"
)

// WireHostChannels mutates package-level globals in hostnotify/run/remote.
// Do not use t.Parallel() here.

func TestWireHostChannels_SubmitDelivered(t *testing.T) {
	ports := hostbus.NewInputPorts()
	WireHostChannels(ports, make(chan []string, 1))

	done := make(chan string, 1)
	go func() {
		done <- <-ports.SubmitChan
	}()

	if !hostnotify.Submit("ping") {
		t.Fatal("Submit returned false")
	}
	select {
	case v := <-done:
		if v != "ping" {
			t.Fatalf("want ping, got %q", v)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("timeout waiting for submit")
	}
}

func TestWireHostChannels_ConfigUpdated(t *testing.T) {
	ports := hostbus.NewInputPorts()
	WireHostChannels(ports, make(chan []string, 1))

	go func() { hostnotify.NotifyConfigUpdated() }()
	select {
	case <-ports.ConfigUpdatedChan:
	case <-time.After(2 * time.Second):
		t.Fatal("timeout on config updated")
	}
}

func TestWireHostChannels_ExecDirectPublish(t *testing.T) {
	ports := hostbus.NewInputPorts()
	WireHostChannels(ports, make(chan []string, 1))

	done := make(chan string, 1)
	go func() {
		done <- <-ports.ExecDirectChan
	}()

	go func() { run.PublishExecDirect("echo ok") }()

	select {
	case v := <-done:
		if v != "echo ok" {
			t.Fatalf("want echo ok, got %q", v)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("timeout exec direct")
	}
}

func TestWireHostChannels_CancelPublish(t *testing.T) {
	ports := hostbus.NewInputPorts()
	WireHostChannels(ports, make(chan []string, 1))

	go func() { <-ports.CancelRequestChan }()

	if !run.PublishCancelRequest() {
		t.Fatal("cancel publish failed")
	}
}

func TestWireHostChannels_RemoteOnOffAuth(t *testing.T) {
	ports := hostbus.NewInputPorts()
	WireHostChannels(ports, make(chan []string, 1))

	if !remote.PublishRemoteOnTarget("dev") {
		t.Fatal("remote on publish failed")
	}
	select {
	case v := <-ports.RemoteOnChan:
		if v != "dev" {
			t.Fatalf("want dev got %q", v)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("timeout remote on")
	}

	if !remote.PublishRemoteOff() {
		t.Fatal("remote off publish failed")
	}
	select {
	case <-ports.RemoteOffChan:
	case <-time.After(2 * time.Second):
		t.Fatal("timeout remote off")
	}

	resp := ui.RemoteAuthResponse{Target: "h", Kind: "password", Password: "x"}
	if !remote.PublishRemoteAuthResponse(resp) {
		t.Fatal("auth resp publish failed")
	}
	select {
	case got := <-ports.RemoteAuthRespChan:
		if got.Target != resp.Target || got.Password != resp.Password {
			t.Fatalf("unexpected resp: %+v", got)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("timeout auth resp")
	}
}

func TestWireHostChannels_ShellSnapshotPublish(t *testing.T) {
	ports := hostbus.NewInputPorts()
	shell := make(chan []string, 1)
	WireHostChannels(ports, shell)

	msgs := []string{"a", "b"}
	if !run.PublishShellSnapshot(msgs) {
		t.Fatal("shell snapshot publish failed")
	}
	select {
	case got := <-shell:
		if len(got) != 2 || got[0] != "a" || got[1] != "b" {
			t.Fatalf("unexpected snapshot: %#v", got)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("timeout shell snapshot")
	}
}

func TestNewInputPorts_CapacitiesMatchHostBus(t *testing.T) {
	p := hostbus.NewInputPorts()
	if cap(p.SubmitChan) != 8 || cap(p.AgentUIChan) != 64 {
		t.Fatalf("unexpected capacities: submit=%d agent=%d", cap(p.SubmitChan), cap(p.AgentUIChan))
	}
}

func TestWireHostChannels_SubmitNonBlockingVsFullBuffer(t *testing.T) {
	ports := hostbus.NewInputPorts()
	WireHostChannels(ports, make(chan []string, 1))

	for i := 0; i < cap(ports.SubmitChan); i++ {
		if !hostnotify.TrySubmitNonBlocking("fill") {
			t.Fatalf("unexpected failure filling at %d", i)
		}
	}
	if hostnotify.TrySubmitNonBlocking("overflow") {
		t.Fatal("expected full buffer to reject non-blocking submit")
	}
}

func TestWireHostChannels_ExecDirectEmptyNoBlock(t *testing.T) {
	ports := hostbus.NewInputPorts()
	WireHostChannels(ports, make(chan []string, 1))

	done := make(chan struct{})
	go func() {
		run.PublishExecDirect("")
		close(done)
	}()

	select {
	case <-done:
	case <-time.After(500 * time.Millisecond):
		t.Fatal("empty exec direct should return without blocking")
	}

	select {
	case <-ports.ExecDirectChan:
		t.Fatal("empty command should not enqueue")
	default:
	}
}

func TestWireHostChannels_MultipleWiresLastWins(t *testing.T) {
	p1 := hostbus.NewInputPorts()
	p2 := hostbus.NewInputPorts()
	shell := make(chan []string, 1)
	WireHostChannels(p1, shell)
	WireHostChannels(p2, shell)

	go func() { <-p2.SubmitChan }()
	if !hostnotify.Submit("second") {
		t.Fatal("submit failed")
	}

	select {
	case <-p1.SubmitChan:
		t.Fatal("first port should not receive after re-wire")
	default:
	}
}
