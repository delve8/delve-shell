package app

import "testing"

func TestParseRemoteDisplayLabel(t *testing.T) {
	n, h := ParseRemoteDisplayLabel("jump (10.0.0.1)")
	if n != "jump" || h != "10.0.0.1" {
		t.Fatalf("got name=%q host=%q", n, h)
	}
	n, h = ParseRemoteDisplayLabel("10.0.0.1")
	if n != "" || h != "10.0.0.1" {
		t.Fatalf("host only: got name=%q host=%q", n, h)
	}
	n, h = ParseRemoteDisplayLabel("")
	if n != "" || h != "" {
		t.Fatalf("empty: got name=%q host=%q", n, h)
	}
}

func TestRuntimeExecContextForLLM(t *testing.T) {
	r := NewRuntime()
	if got := r.ExecContextForLLM(); got != "Local" {
		t.Fatalf("fresh runtime: got %q", got)
	}

	r.SetRemoteExecution(true, "jump (10.0.0.1)", "10.0.0.1", "jump")
	if got := r.ExecContextForLLM(); got != "Remote: jump @ 10.0.0.1" {
		t.Fatalf("remote with name+host: got %q", got)
	}

	r.SetRemoteExecution(true, "10.0.0.1", "10.0.0.1", "")
	if got := r.ExecContextForLLM(); got != "Remote: 10.0.0.1" {
		t.Fatalf("remote host only: got %q", got)
	}

	r.SetRemoteExecution(true, "jump (10.0.0.1)", "", "")
	if got := r.ExecContextForLLM(); got != "Remote: jump @ 10.0.0.1" {
		t.Fatalf("parse label fallback: got %q", got)
	}

	r.SetRemoteExecution(true, "  ", "", "")
	if got := r.ExecContextForLLM(); got != "Remote" {
		t.Fatalf("blank label: got %q", got)
	}

	r.SetOffline(true)
	if got := r.ExecContextForLLM(); got != "Offline (manual relay)" {
		t.Fatalf("offline: got %q", got)
	}
}
