package executormgr

import (
	"context"
	"crypto/ed25519"
	"crypto/rand"
	"errors"
	"testing"

	"delve-shell/internal/remote/auth"
	"delve-shell/internal/remote/execenv"
	"golang.org/x/crypto/ssh"
)

type fakeExec struct{}

func (fakeExec) Run(_ context.Context, _ string) (string, string, int, error) {
	return "", "", 0, nil
}

type fakeIssueExec struct {
	handler func(string)
}

func (f *fakeIssueExec) Run(_ context.Context, _ string) (string, string, int, error) {
	return "", "", 0, nil
}

func (f *fakeIssueExec) SetTransportIssueHandler(fn func(string)) {
	f.handler = fn
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
	m.PutCachedCred("example.com", remoteauth.ResponseKindIdentity, "root", "/tmp/id_rsa")

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
	m.PutCachedCred("example.com", remoteauth.ResponseKindPassword, "root", "pw")
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
			return nil, "", errors.New("ssh: handshake failed: ssh: unable to authenticate, attempted methods [none publickey], no supported methods remain")
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

func TestConnect_TransportFailure_DoesNotPromptForAuth(t *testing.T) {
	m := New()
	m.SetSSHFactories(
		func(target, identity string) (execenv.CommandExecutor, string, error) {
			return nil, "", errors.New("dial tcp 10.0.0.1:22: connect: connection refused")
		},
		nil,
	)
	res := m.Connect("root@example.com", "lbl", "")
	if res.Connected {
		t.Fatal("expected not connected")
	}
	if res.AuthPrompt != nil {
		t.Fatalf("unexpected auth prompt: %+v", res.AuthPrompt)
	}
	if res.ErrText == "" {
		t.Fatal("expected plain connection error text")
	}
}

func TestConnect_AuthFailure_PromptsForCredentials(t *testing.T) {
	m := New()
	m.SetSSHFactories(
		func(target, identity string) (execenv.CommandExecutor, string, error) {
			return nil, "", errors.New("ssh: handshake failed: ssh: unable to authenticate, attempted methods [none], no supported methods remain")
		},
		nil,
	)
	res := m.Connect("root@example.com", "lbl", "")
	if res.Connected {
		t.Fatal("expected not connected")
	}
	if res.AuthPrompt == nil {
		t.Fatal("expected auth prompt")
	}
	if res.ErrText != "" {
		t.Fatalf("unexpected errText: %q", res.ErrText)
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
	label, err := m.HandleRemoteAuthResponse(remoteauth.Response{
		Target:   "root@example.com",
		Username: "root",
		Kind:     remoteauth.ResponseKindPassword,
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

func TestConnect_HostKeyMismatch_ReturnsVerifyPrompt(t *testing.T) {
	m := New()
	m.SetSSHFactories(
		func(target, identity string) (execenv.CommandExecutor, string, error) {
			return nil, "", &execenv.HostKeyMismatchError{
				Hostname:    "[example.com]:22",
				Fingerprint: "SHA256:test",
			}
		},
		nil,
	)
	res := m.Connect("root@example.com", "lbl", "")
	if res.Connected {
		t.Fatalf("expected not connected")
	}
	if res.AuthPrompt == nil || !res.AuthPrompt.HostKeyVerify {
		t.Fatalf("expected host-key verify prompt")
	}
	if res.AuthPrompt.HostKeyFingerprint == "" {
		t.Fatalf("expected fingerprint in host-key verify prompt")
	}
}

func TestConnect_UnknownHostKey_ReturnsVerifyPrompt(t *testing.T) {
	m := New()
	m.SetSSHFactories(
		func(target, identity string) (execenv.CommandExecutor, string, error) {
			return nil, "", &execenv.HostKeyMismatchError{
				Hostname:    "[example.com]:22",
				Fingerprint: "SHA256:unknown",
				UnknownHost: true,
			}
		},
		nil,
	)
	res := m.Connect("root@example.com", "lbl", "")
	if res.Connected {
		t.Fatalf("expected not connected")
	}
	if res.AuthPrompt == nil || !res.AuthPrompt.HostKeyVerify {
		t.Fatalf("expected host-key verify prompt")
	}
	if res.AuthPrompt.HostKeyFingerprint != "SHA256:unknown" {
		t.Fatalf("unexpected fingerprint: %q", res.AuthPrompt.HostKeyFingerprint)
	}
	if res.AuthPrompt.Err == "" {
		t.Fatalf("expected host-key decision message")
	}
}

func TestResolveHostKeyDecision_Accept_Reconnects(t *testing.T) {
	t.Setenv("HOME", t.TempDir())
	m := New()
	pub, _, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		t.Fatalf("generate key: %v", err)
	}
	sshKey, err := ssh.NewPublicKey(pub)
	if err != nil {
		t.Fatalf("new public key: %v", err)
	}
	calls := 0
	m.SetSSHFactories(
		func(target, identity string) (execenv.CommandExecutor, string, error) {
			calls++
			if calls == 1 {
				return nil, "", &execenv.HostKeyMismatchError{
					Hostname:    "[example.com]:22",
					Fingerprint: "SHA256:test",
					Key:         sshKey,
				}
			}
			return fakeExec{}, "", nil
		},
		nil,
	)
	first := m.Connect("root@example.com", "lbl", "")
	if first.AuthPrompt == nil || !first.AuthPrompt.HostKeyVerify {
		t.Fatalf("expected host-key verify prompt from first connect")
	}
	res := m.ResolveHostKeyDecision("root@example.com", true)
	if !res.Connected {
		t.Fatalf("expected connected after accepting host key")
	}
}

func TestSetRemoteIssueHandler_AttachesToExecutor(t *testing.T) {
	m := New()
	exec := &fakeIssueExec{}
	var got string
	m.SetRemoteIssueHandler(func(issue string) {
		got = issue
	})
	m.Set(exec)
	if exec.handler == nil {
		t.Fatal("expected transport issue handler to be attached")
	}
	exec.handler("lost")
	if got != "lost" {
		t.Fatalf("issue=%q want %q", got, "lost")
	}
}
