package executormgr

import (
	"context"
	"errors"
	"testing"

	"delve-shell/internal/execenv"
	"delve-shell/internal/ui"
)

type fakeExec struct{}

func (fakeExec) Run(_ context.Context, _ string) (string, string, int, error) {
	return "", "", 0, nil
}

func TestConnect_UsesCachedCredential_First(t *testing.T) {
	m := New()
	called := 0
	m.SetSSHFactories(
		func(target, identity string) (execenv.CommandExecutor, string, error) {
			called++
			return fakeExec{}, "", nil
		},
		func(target, identity, password string) (execenv.CommandExecutor, string, error) {
			t.Fatalf("unexpected password ssh factory")
			return nil, "", nil
		},
	)
	m.PutCachedCred("example.com", "identity", "root", "/tmp/id_rsa")

	res := m.Connect("example.com", "lbl", "")
	if !res.Connected || res.Executor == nil {
		t.Fatalf("expected connected with executor")
	}
	if called != 1 {
		t.Fatalf("expected ssh factory called once, got %d", called)
	}
}

func TestConnect_CachedCredentialFailure_DropsCache(t *testing.T) {
	m := New()
	m.SetSSHFactories(
		func(target, identity string) (execenv.CommandExecutor, string, error) {
			return nil, "", errors.New("fail")
		},
		func(target, identity, password string) (execenv.CommandExecutor, string, error) {
			return nil, "", errors.New("fail")
		},
	)
	m.PutCachedCred("example.com", "password", "root", "pw")
	res := m.Connect("example.com", "", "")
	if res.Connected {
		t.Fatalf("expected not connected")
	}
	if _, _, _, ok := m.GetCachedCred("example.com"); ok {
		t.Fatalf("expected cached cred to be dropped on failure")
	}
}

func TestConnect_ConfigIdentity_Failure_ReturnsPrompt(t *testing.T) {
	m := New()
	m.SetSSHFactories(
		func(target, identity string) (execenv.CommandExecutor, string, error) {
			return nil, "", errors.New("bad key")
		},
		nil,
	)
	res := m.Connect("root@example.com", "mylabel", "/tmp/key")
	if res.Connected {
		t.Fatalf("expected not connected")
	}
	if res.AuthPrompt == nil {
		t.Fatalf("expected auth prompt")
	}
	if res.AuthPrompt.Target != "root@example.com" {
		t.Fatalf("unexpected target: %q", res.AuthPrompt.Target)
	}
}

func TestHandleRemoteAuthResponse_Success_CachesAndSetsExecutor(t *testing.T) {
	m := New()
	m.SetSSHFactories(
		func(target, identity string) (execenv.CommandExecutor, string, error) {
			return fakeExec{}, "", nil
		},
		func(target, identity, password string) (execenv.CommandExecutor, string, error) {
			return fakeExec{}, "", nil
		},
	)
	label, err := m.HandleRemoteAuthResponse(ui.RemoteAuthResponse{
		Target:   "root@example.com",
		Username: "root",
		Kind:     "password",
		Password: "pw",
	})
	if err != nil {
		t.Fatalf("unexpected err: %v", err)
	}
	if label == "" {
		t.Fatalf("expected non-empty label")
	}
	if m.Get() == nil {
		t.Fatalf("expected executor to be set")
	}
	if _, _, _, ok := m.GetCachedCred("example.com"); !ok {
		t.Fatalf("expected cred cached for hostOnly")
	}
}

