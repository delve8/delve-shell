package wiring

import (
	"strings"
	"testing"
	"time"

	"delve-shell/internal/host/app"
	"delve-shell/internal/host/bus"
	"delve-shell/internal/inputlifecycletype"
	"delve-shell/internal/remoteauth"
)

func TestBindSendPorts_SubmissionDelivered(t *testing.T) {
	ports := bus.NewInputPorts()
	rt := bindTestPorts(t, ports, make(chan []string, 1))

	done := make(chan inputlifecycletype.InputSubmission, 1)
	go func() {
		done <- <-ports.SubmissionChan
	}()

	sub := inputlifecycletype.InputSubmission{
		Kind:    inputlifecycletype.SubmissionChat,
		Source:  inputlifecycletype.SourceProgrammatic,
		RawText: "ping",
	}
	if !rt.SubmitSubmission(sub) {
		t.Fatal("SubmitSubmission returned false")
	}
	select {
	case v := <-done:
		if v != sub {
			t.Fatalf("want %#v, got %#v", sub, v)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("timeout waiting for submission")
	}
}

func TestBindSendPorts_ConfigUpdated(t *testing.T) {
	ports := bus.NewInputPorts()
	rt := bindTestPorts(t, ports, make(chan []string, 1))

	go func() { rt.NotifyConfigUpdated() }()
	select {
	case <-ports.ConfigUpdatedChan:
	case <-time.After(2 * time.Second):
		t.Fatal("timeout on config updated")
	}
}

func TestBindSendPorts_ExecDirectPublish(t *testing.T) {
	ports := bus.NewInputPorts()
	rt := bindTestPorts(t, ports, make(chan []string, 1))

	done := make(chan string, 1)
	go func() {
		done <- <-ports.ExecDirectChan
	}()

	go func() { rt.PublishExecDirect("echo ok") }()

	select {
	case v := <-done:
		if v != "echo ok" {
			t.Fatalf("want echo ok, got %q", v)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("timeout exec direct")
	}
}

func TestBindSendPorts_CancelPublish(t *testing.T) {
	ports := bus.NewInputPorts()
	rt := bindTestPorts(t, ports, make(chan []string, 1))

	go func() { <-ports.CancelRequestChan }()

	if !rt.PublishCancelRequest() {
		t.Fatal("cancel publish failed")
	}
}

func TestBindSendPorts_RemoteOnOffAuth(t *testing.T) {
	ports := bus.NewInputPorts()
	rt := bindTestPorts(t, ports, make(chan []string, 1))

	if !rt.PublishRemoteOnTarget("dev") {
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

	if !rt.PublishRemoteOff() {
		t.Fatal("remote off publish failed")
	}
	select {
	case <-ports.RemoteOffChan:
	case <-time.After(2 * time.Second):
		t.Fatal("timeout remote off")
	}

	resp := remoteauth.Response{Target: "h", Kind: "password", Password: "x"}
	if !rt.PublishRemoteAuthResponse(resp) {
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

func TestBindSendPorts_ShellSnapshotPublish(t *testing.T) {
	ports := bus.NewInputPorts()
	shell := make(chan []string, 1)
	rt := bindTestPorts(t, ports, shell)

	msgs := []string{"a", "b"}
	if !rt.PublishShellSnapshot(msgs) {
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

func TestBindSendPorts_SubmitNonBlockingVsFullBuffer(t *testing.T) {
	ports := bus.NewInputPorts()
	rt := bindTestPorts(t, ports, make(chan []string, 1))

	sub := inputlifecycletype.InputSubmission{
		Kind:    inputlifecycletype.SubmissionChat,
		Source:  inputlifecycletype.SourceProgrammatic,
		RawText: "fill",
	}
	for i := 0; i < cap(ports.SubmissionChan); i++ {
		if !rt.TrySubmitSubmissionNonBlocking(sub) {
			t.Fatalf("unexpected failure filling at %d", i)
		}
	}
	if rt.TrySubmitSubmissionNonBlocking(sub) {
		t.Fatal("expected full buffer to reject non-blocking submit")
	}
}

func TestBindSendPorts_ExecDirectEmptyNoBlock(t *testing.T) {
	ports := bus.NewInputPorts()
	rt := bindTestPorts(t, ports, make(chan []string, 1))

	done := make(chan struct{})
	go func() {
		rt.PublishExecDirect("")
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

func TestBindSendPorts_MultipleBindsLastWins(t *testing.T) {
	p1 := bus.NewInputPorts()
	p2 := bus.NewInputPorts()
	shell := make(chan []string, 1)
	r1 := app.NewRuntime()
	BindSendPorts(r1, p1, shell)
	r2 := app.NewRuntime()
	BindSendPorts(r2, p2, shell)
	t.Cleanup(func() { r1.Reset(); r2.Reset() })

	go func() { <-p2.SubmissionChan }()
	if !r2.SubmitSubmission(inputlifecycletype.InputSubmission{
		Kind:    inputlifecycletype.SubmissionChat,
		Source:  inputlifecycletype.SourceProgrammatic,
		RawText: "second",
	}) {
		t.Fatal("submit failed")
	}

	select {
	case <-p1.SubmissionChan:
		t.Fatal("first port should not receive after re-bind")
	default:
	}
}

func TestInputPortsCapacitiesDocumented(t *testing.T) {
	p := bus.NewInputPorts()
	type spec struct {
		name string
		cap  int
		got  int
	}
	specs := []spec{
		{"SubmissionChan", 8, cap(p.SubmissionChan)},
		{"ConfigUpdatedChan", 8, cap(p.ConfigUpdatedChan)},
		{"CancelRequestChan", 8, cap(p.CancelRequestChan)},
		{"ExecDirectChan", 8, cap(p.ExecDirectChan)},
		{"RemoteOnChan", 4, cap(p.RemoteOnChan)},
		{"RemoteOffChan", 4, cap(p.RemoteOffChan)},
		{"RemoteAuthRespChan", 4, cap(p.RemoteAuthRespChan)},
		{"AgentUIChan", 64, cap(p.AgentUIChan)},
	}
	for _, s := range specs {
		if s.cap != s.got {
			t.Fatalf("%s: want cap %d got %d", s.name, s.cap, s.got)
		}
	}
}

func TestBindSendPorts_AgentUIChanUnwired(t *testing.T) {
	ports := bus.NewInputPorts()
	bindTestPorts(t, ports, make(chan []string, 1))
	if cap(ports.AgentUIChan) < 1 {
		t.Fatal("agent chan missing capacity")
	}
}

func TestBindSendPorts_ConfigUpdatedNonBlockingDrop(t *testing.T) {
	ports := bus.NewInputPorts()
	rt := bindTestPorts(t, ports, make(chan []string, 1))
	n := cap(ports.ConfigUpdatedChan)
	for i := 0; i < n+20; i++ {
		rt.NotifyConfigUpdated()
	}
	count := 0
	for {
		select {
		case <-ports.ConfigUpdatedChan:
			count++
		default:
			if count != n {
				t.Fatalf("want exactly %d retained events, drained %d", n, count)
			}
			return
		}
	}
}

func TestBindSendPorts_RemoteBuffersIndependent(t *testing.T) {
	ports := bus.NewInputPorts()
	rt := bindTestPorts(t, ports, make(chan []string, 1))

	for i := 0; i < cap(ports.RemoteOnChan); i++ {
		if !rt.PublishRemoteOnTarget("x") {
			t.Fatalf("fill remote on at %d", i)
		}
	}
	if rt.PublishRemoteOnTarget("overflow") {
		t.Fatal("expected full remote on buffer")
	}
}

func TestBindSendPorts_SubmitStressSequential(t *testing.T) {
	ports := bus.NewInputPorts()
	rt := bindTestPorts(t, ports, make(chan []string, 1))

	const total = 200
	go func() {
		for i := range total {
			_ = rt.SubmitSubmission(inputlifecycletype.InputSubmission{
				Kind:    inputlifecycletype.SubmissionChat,
				Source:  inputlifecycletype.SourceProgrammatic,
				RawText: strings.Repeat("a", i%5+1),
			})
		}
	}()
	for range total {
		select {
		case <-ports.SubmissionChan:
		case <-time.After(3 * time.Second):
			t.Fatal("timeout draining submit")
		}
	}
}

func TestBindSendPorts_ShellSnapshotDelivered(t *testing.T) {
	ports := bus.NewInputPorts()
	shell := make(chan []string, 1)
	rt := bindTestPorts(t, ports, shell)

	msgs := []string{"line1", "line2"}
	if !rt.PublishShellSnapshot(msgs) {
		t.Fatal("publish failed")
	}
	got := <-shell
	if len(got) != 2 || got[0] != "line1" {
		t.Fatalf("unexpected snapshot: %#v", got)
	}
}

func TestBindSendPorts_RemoteOnSequential(t *testing.T) {
	ports := bus.NewInputPorts()
	rt := bindTestPorts(t, ports, make(chan []string, 1))

	targets := []string{"a", "b", "c", "d", "e"}
	for _, want := range targets {
		if !rt.PublishRemoteOnTarget(want) {
			t.Fatalf("publish failed for %q", want)
		}
		got := <-ports.RemoteOnChan
		if got != want {
			t.Fatalf("want %q got %q", want, got)
		}
	}
}

func TestBindSendPorts_ExecDirectSequential(t *testing.T) {
	ports := bus.NewInputPorts()
	rt := bindTestPorts(t, ports, make(chan []string, 1))

	cmds := []string{"a", "b", "c", "d", "e", "f", "g", "h", "i", "j"}
	for _, want := range cmds {
		done := make(chan string, 1)
		go func() {
			done <- <-ports.ExecDirectChan
		}()
		rt.PublishExecDirect(want)
		if got := <-done; got != want {
			t.Fatalf("want %q got %q", want, got)
		}
	}
}

func TestBindSendPorts_CancelBurstWithinCapacity(t *testing.T) {
	ports := bus.NewInputPorts()
	rt := bindTestPorts(t, ports, make(chan []string, 1))
	n := cap(ports.CancelRequestChan)
	for i := 0; i < n; i++ {
		if !rt.PublishCancelRequest() {
			t.Fatalf("cancel %d failed", i)
		}
	}
	if rt.PublishCancelRequest() {
		t.Fatal("expected full cancel buffer to reject")
	}
	for i := 0; i < n; i++ {
		<-ports.CancelRequestChan
	}
}

func TestBindSendPorts_AuthResponseTable(t *testing.T) {
	ports := bus.NewInputPorts()
	rt := bindTestPorts(t, ports, make(chan []string, 1))

	variants := []remoteauth.Response{
		{Target: "root@10.0.0.1", Username: "root", Kind: "password", Password: "secret"},
		{Target: "host", Username: "", Kind: "identity", Password: "/path/to/key"},
		{Target: "u@h", Username: "u", Kind: "password", Password: "x"},
		{Target: "single", Username: "", Kind: "password", Password: "y"},
		{Target: "t2", Username: "admin", Kind: "identity", Password: "./id_rsa"},
		{Target: "t3", Username: "", Kind: "password", Password: strings.Repeat("p", 64)},
		{Target: "t4", Username: "longuser", Kind: "password", Password: "z"},
		{Target: "t5", Username: "", Kind: "identity", Password: "~/.ssh/id_ed25519"},
		{Target: "t6", Username: "git", Kind: "password", Password: "token"},
		{Target: "t7", Username: "", Kind: "password", Password: "a\nb"},
		{Target: "t8", Username: "x", Kind: "password", Password: "\t"},
		{Target: "t9", Username: "", Kind: "identity", Password: "/tmp/k"},
		{Target: "t10", Username: "u1", Kind: "password", Password: "1"},
		{Target: "t11", Username: "u2", Kind: "password", Password: "2"},
		{Target: "t12", Username: "u3", Kind: "password", Password: "3"},
		{Target: "t13", Username: "u4", Kind: "password", Password: "4"},
		{Target: "t14", Username: "u5", Kind: "password", Password: "5"},
		{Target: "t15", Username: "u6", Kind: "password", Password: "6"},
		{Target: "t16", Username: "u7", Kind: "password", Password: "7"},
		{Target: "t17", Username: "u8", Kind: "password", Password: "8"},
		{Target: "t18", Username: "u9", Kind: "password", Password: "9"},
		{Target: "t19", Username: "u10", Kind: "password", Password: "10"},
		{Target: "t20", Username: "", Kind: "identity", Password: "/a"},
		{Target: "t21", Username: "", Kind: "identity", Password: "/b"},
		{Target: "t22", Username: "", Kind: "identity", Password: "/c"},
		{Target: "t23", Username: "mix", Kind: "password", Password: "pw"},
		{Target: "t24", Username: "mix2", Kind: "identity", Password: "/k2"},
		{Target: "t25", Username: "", Kind: "password", Password: ""},
	}
	for i, want := range variants {
		if !rt.PublishRemoteAuthResponse(want) {
			t.Fatalf("case %d: publish failed for %+v", i, want)
		}
		got := <-ports.RemoteAuthRespChan
		if got.Target != want.Target || got.Kind != want.Kind || got.Password != want.Password || got.Username != want.Username {
			t.Fatalf("case %d: want %+v got %+v", i, want, got)
		}
	}
}

func TestBindSendPorts_SubmitPayloadTable(t *testing.T) {
	ports := bus.NewInputPorts()
	rt := bindTestPorts(t, ports, make(chan []string, 1))

	payloads := []string{
		"hello",
		"/help",
		"/config llm",
		"",
		"a",
		strings.Repeat("x", 300),
		"line1\nline2",
		"\ttrimmed by UI normally\t",
		"unicode 你好",
		`/run echo "hi"`,
		`/session ` + strings.Repeat("id", 5),
		`/new`,
		`question ?`,
		`exclaim !`,
		`dollar $VAR`,
		`percent %`,
		`caret ^`,
		`ampersand &`,
		`star *`,
		`paren ( )`,
		`bracket [ ]`,
		`brace { }`,
		`angle < >`,
		`pipe |`,
		`backslash \\`,
		`slash /`,
		`colon :`,
		`semi ;`,
		`quote '`,
		`double "`,
		"`",
	}
	for i, want := range payloads {
		recv := make(chan inputlifecycletype.InputSubmission, 1)
		go func() {
			recv <- <-ports.SubmissionChan
		}()
		sub := inputlifecycletype.InputSubmission{
			Kind:    inputlifecycletype.SubmissionChat,
			Source:  inputlifecycletype.SourceProgrammatic,
			RawText: want,
		}
		if !rt.SubmitSubmission(sub) {
			t.Fatalf("case %d: submit failed for %q", i, want)
		}
		select {
		case got := <-recv:
			if got != sub {
				t.Fatalf("case %d: want %#v got %#v", i, sub, got)
			}
		case <-time.After(2 * time.Second):
			t.Fatalf("case %d: timeout", i)
		}
	}
}

func TestBindSendPorts_ExecDirectPayloadTable(t *testing.T) {
	ports := bus.NewInputPorts()
	rt := bindTestPorts(t, ports, make(chan []string, 1))

	cmds := []string{
		"true",
		"false",
		"echo ok",
		`/bin/sh -c 'exit 0'`,
		"uname",
		"pwd",
		strings.Repeat("n", 120),
		"cmd with spaces",
	}
	for i, want := range cmds {
		received := make(chan string, 1)
		go func() {
			received <- <-ports.ExecDirectChan
		}()
		go func() { rt.PublishExecDirect(want) }()
		select {
		case got := <-received:
			if got != want {
				t.Fatalf("case %d: want %q got %q", i, want, got)
			}
		case <-time.After(2 * time.Second):
			t.Fatalf("case %d: timeout", i)
		}
	}
}
