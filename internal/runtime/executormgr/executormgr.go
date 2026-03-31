package executormgr

import (
	"errors"
	"fmt"
	"sync"

	"delve-shell/internal/config"
	"delve-shell/internal/remote/auth"
	"delve-shell/internal/remote/execenv"
)

type sshNewFunc func(target, identityFile string) (execenv.CommandExecutor, string, error)
type sshNewWithPasswordFunc func(target, identityFile, password string) (execenv.CommandExecutor, string, error)

// Manager owns the current command executor (local or remote) and remote credential cache.
// It is safe for concurrent use.
type Manager struct {
	mu       sync.Mutex
	executor execenv.CommandExecutor
	pending  *pendingHostKeyDecision

	remoteCredMu sync.Mutex
	remoteCreds  map[string]remoteCred // key: host-only

	newSSH             sshNewFunc
	newSSHWithPassword sshNewWithPasswordFunc
}

type remoteCred struct {
	Kind     string // "password" or "identity"
	Username string
	Secret   string // password or identity file path
}

type pendingHostKeyDecision struct {
	target       string
	label        string
	identityFile string
	mismatch     *execenv.HostKeyMismatchError
}

func New() *Manager {
	return &Manager{
		executor:    execenv.LocalExecutor{},
		remoteCreds: make(map[string]remoteCred),
		newSSH: func(target, identityFile string) (execenv.CommandExecutor, string, error) {
			return execenv.NewSSHExecutor(target, identityFile)
		},
		newSSHWithPassword: func(target, identityFile, password string) (execenv.CommandExecutor, string, error) {
			return execenv.NewSSHExecutorWithPassword(target, identityFile, password)
		},
	}
}

// SetSSHFactories allows tests to stub SSH executor creation.
func (m *Manager) SetSSHFactories(newSSH sshNewFunc, newSSHWithPassword sshNewWithPasswordFunc) {
	if newSSH != nil {
		m.newSSH = newSSH
	}
	if newSSHWithPassword != nil {
		m.newSSHWithPassword = newSSHWithPassword
	}
}

func (m *Manager) Get() execenv.CommandExecutor {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.executor
}

// Set switches the current executor without touching credential cache.
// Callers are responsible for closing any replaced SSH executor when needed.
func (m *Manager) Set(exec execenv.CommandExecutor) {
	m.mu.Lock()
	m.executor = exec
	m.mu.Unlock()
}

func (m *Manager) GetCachedCred(hostOnly string) (kind, username, secret string, ok bool) {
	m.remoteCredMu.Lock()
	defer m.remoteCredMu.Unlock()
	c, ok := m.remoteCreds[hostOnly]
	if !ok {
		return "", "", "", false
	}
	return c.Kind, c.Username, c.Secret, true
}

func (m *Manager) PutCachedCred(hostOnly, kind, username, secret string) {
	if hostOnly == "" || secret == "" {
		return
	}
	if kind != "identity" {
		kind = "password"
	}
	m.remoteCredMu.Lock()
	m.remoteCreds[hostOnly] = remoteCred{Kind: kind, Username: username, Secret: secret}
	m.remoteCredMu.Unlock()
}

func (m *Manager) DeleteCachedCred(hostOnly string) {
	m.remoteCredMu.Lock()
	delete(m.remoteCreds, hostOnly)
	m.remoteCredMu.Unlock()
}

// SwitchToLocal closes any SSH executor, switches to local, and clears credential cache.
func (m *Manager) SwitchToLocal() {
	m.mu.Lock()
	if sshExec, ok := m.executor.(*execenv.SSHExecutor); ok {
		_ = sshExec.Close()
	}
	m.executor = execenv.LocalExecutor{}
	m.mu.Unlock()

	m.remoteCredMu.Lock()
	for k := range m.remoteCreds {
		delete(m.remoteCreds, k)
	}
	m.remoteCredMu.Unlock()
}

// HandleRemoteAuthResponse tries to create an SSH executor using user-provided credentials.
// On success it switches the current executor and caches the credential for the host.
func (m *Manager) HandleRemoteAuthResponse(resp remoteauth.Response) (label string, err error) {
	if resp.Password == "" || resp.Target == "" {
		return "", fmt.Errorf("empty remote auth response")
	}
	targetForSSH := resp.Target
	hostOnly := config.HostFromTarget(resp.Target)
	if resp.Username != "" {
		targetForSSH = resp.Username + "@" + hostOnly
	}

	var sshExec execenv.CommandExecutor
	switch resp.Kind {
	case "identity":
		sshExec, _, err = m.newSSH(targetForSSH, resp.Password)
	default:
		sshExec, _, err = m.newSSHWithPassword(targetForSSH, "", resp.Password)
	}
	if err != nil {
		return "", err
	}

	kind := resp.Kind
	if kind != "identity" {
		kind = "password"
	}
	m.remoteCredMu.Lock()
	m.remoteCreds[hostOnly] = remoteCred{Kind: kind, Username: resp.Username, Secret: resp.Password}
	m.remoteCredMu.Unlock()

	m.Set(sshExec)
	return config.HostFromTarget(targetForSSH), nil
}

type ConnectResult struct {
	Connected  bool
	Label      string
	Executor   execenv.CommandExecutor // non-nil when Connected
	AuthPrompt *remoteauth.Prompt      // when non-nil, UI should open auth prompt / show error
}

// Connect attempts to switch to a remote SSH executor.
//
// - target: ssh target, e.g. "user@host" or "host"
// - label: display label for UI (may differ from hostOnly)
// - identityFile: configured key path for this remote (optional)
//
// Behavior:
//   - If a cached credential exists for hostOnly, try it first. On failure, drop it and continue.
//   - If identityFile is provided, emit an AuthPrompt in "auto identity" mode, then attempt connection.
//     On failure, emit an AuthPrompt with error for interactive auth.
//   - Otherwise, try plain SSH; on failure, emit an AuthPrompt with error.
func (m *Manager) Connect(target, label, identityFile string) ConnectResult {
	hostOnly := config.HostFromTarget(target)
	if label == "" {
		label = hostOnly
	}

	// 1) Cached credential
	kind, cachedUser, cachedSecret, ok := m.GetCachedCred(hostOnly)
	if ok {
		targetForSSH := target
		if cachedUser != "" {
			targetForSSH = cachedUser + "@" + hostOnly
		}
		var exec execenv.CommandExecutor
		var err error
		if kind == "identity" {
			exec, _, err = m.newSSH(targetForSSH, cachedSecret)
		} else {
			exec, _, err = m.newSSHWithPassword(targetForSSH, "", cachedSecret)
		}
		if err == nil && exec != nil {
			m.Set(exec)
			return ConnectResult{Connected: true, Label: label, Executor: exec}
		}
		m.DeleteCachedCred(hostOnly)
	}

	// 2) Configured identity
	if identityFile != "" {
		info := fmt.Sprintf("Using configured SSH key: %s", identityFile)
		prompt := &remoteauth.Prompt{Target: target, Err: info, UseConfiguredIdentity: true}
		exec, _, err := m.newSSH(target, identityFile)
		if err != nil || exec == nil {
			var mismatch *execenv.HostKeyMismatchError
			if errors.As(err, &mismatch) {
				m.mu.Lock()
				m.pending = &pendingHostKeyDecision{
					target:       target,
					label:        label,
					identityFile: identityFile,
					mismatch:     mismatch,
				}
				m.mu.Unlock()
				return ConnectResult{
					Connected: false,
					Label:     label,
					AuthPrompt: &remoteauth.Prompt{
						Target:             target,
						HostKeyVerify:      true,
						HostKeyHost:        mismatch.Hostname,
						HostKeyFingerprint: mismatch.Fingerprint,
						Err:                hostKeyDecisionPrompt(mismatch),
					},
				}
			}
			msg := fmt.Sprintf("Remote connect failed for %s: %v", hostOnly, err)
			return ConnectResult{Connected: false, Label: label, AuthPrompt: &remoteauth.Prompt{Target: target, Err: msg}}
		}
		m.Set(exec)
		_ = prompt
		return ConnectResult{Connected: true, Label: label, Executor: exec, AuthPrompt: prompt}
	}

	// 3) Plain SSH attempt
	exec, _, err := m.newSSH(target, "")
	if err != nil || exec == nil {
		var mismatch *execenv.HostKeyMismatchError
		if errors.As(err, &mismatch) {
			m.mu.Lock()
			m.pending = &pendingHostKeyDecision{
				target:       target,
				label:        label,
				identityFile: identityFile,
				mismatch:     mismatch,
			}
			m.mu.Unlock()
			return ConnectResult{
				Connected: false,
				Label:     label,
				AuthPrompt: &remoteauth.Prompt{
					Target:             target,
					HostKeyVerify:      true,
					HostKeyHost:        mismatch.Hostname,
					HostKeyFingerprint: mismatch.Fingerprint,
					Err:                hostKeyDecisionPrompt(mismatch),
				},
			}
		}
		msg := fmt.Sprintf("Remote connect failed for %s: %v", hostOnly, err)
		return ConnectResult{Connected: false, Label: label, AuthPrompt: &remoteauth.Prompt{Target: target, Err: msg}}
	}
	m.Set(exec)
	return ConnectResult{Connected: true, Label: label, Executor: exec}
}

func hostKeyDecisionPrompt(mismatch *execenv.HostKeyMismatchError) string {
	if mismatch != nil && mismatch.UnknownHost {
		return "Host key is not trusted yet. Accept to add/update known_hosts or reject to abort."
	}
	return "Host key mismatch detected. Accept to update known_hosts or reject to abort."
}

// ResolveHostKeyDecision resolves a pending host-key mismatch decision and retries connection on accept.
func (m *Manager) ResolveHostKeyDecision(target string, accept bool) ConnectResult {
	m.mu.Lock()
	pending := m.pending
	m.pending = nil
	m.mu.Unlock()
	if pending == nil || pending.mismatch == nil {
		return ConnectResult{Connected: false, Label: config.HostFromTarget(target)}
	}
	if !accept {
		return ConnectResult{Connected: false, Label: pending.label}
	}
	if err := execenv.UpdateKnownHost(pending.mismatch.Hostname, pending.mismatch.Key); err != nil {
		return ConnectResult{
			Connected:  false,
			Label:      pending.label,
			AuthPrompt: &remoteauth.Prompt{Target: pending.target, Err: fmt.Sprintf("Failed to update known_hosts: %v", err)},
		}
	}
	exec, _, err := m.newSSH(pending.target, pending.identityFile)
	if err != nil || exec == nil {
		return ConnectResult{
			Connected:  false,
			Label:      pending.label,
			AuthPrompt: &remoteauth.Prompt{Target: pending.target, Err: fmt.Sprintf("Remote connect failed for %s: %v", config.HostFromTarget(pending.target), err)},
		}
	}
	m.Set(exec)
	return ConnectResult{Connected: true, Label: pending.label, Executor: exec}
}
