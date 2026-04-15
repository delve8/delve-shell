package execenv

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"net"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"syscall"
	"time"

	"delve-shell/internal/config"

	"golang.org/x/crypto/ssh"
	"golang.org/x/crypto/ssh/agent"
	"golang.org/x/crypto/ssh/knownhosts"
)

// SSHExecutor runs commands on a remote host via SSH.
// It keeps a single SSH client connection and opens a new session per Run.
type SSHExecutor struct {
	mu               sync.Mutex
	reconnectMu      sync.Mutex
	client           *ssh.Client
	clientClose      func() error
	keepAliveStop    chan struct{}
	recoveryStop     chan struct{}
	closed           bool
	transportIssue   string
	transportIssueFn func(string)
	user             string
	addr             string // host:port
	target           string
	auth             sshAuthConfig
}

type sshAuthConfig struct {
	identityFile string
	password     string
}

type sshClientHandle struct {
	client *ssh.Client
	close  func() error
}

type sshConnectTarget struct {
	user     string
	hostPort string
}

const (
	sshDialTimeout            = 10 * time.Second
	sshTCPKeepAliveIdle       = 10 * time.Second
	sshTCPKeepAliveInterval   = 10 * time.Second
	sshTCPKeepAliveCount      = 3
	sshAppKeepAliveInterval   = 10 * time.Second
	sshAppKeepAliveReqTimeout = 5 * time.Second
	sshRecoveryRetryInterval  = 5 * time.Second

	sshConnectionIssueSummary = "disconnected"
)

// SSHConnectionError reports that the SSH transport failed, usually due to network interruption or a dropped session.
type SSHConnectionError struct {
	Op               string
	Err              error
	ReconnectTried   bool
	ReconnectSuccess bool
}

func (e *SSHConnectionError) Error() string {
	if e == nil {
		return "ssh connection lost"
	}
	msg := "ssh connection lost"
	if e.Op != "" {
		msg += " during " + e.Op
	}
	if e.Err != nil {
		msg += ": " + e.Err.Error()
	}
	switch {
	case e.ReconnectSuccess:
		msg += " (reconnected; retry the command)"
	case e.ReconnectTried:
		msg += " (automatic reconnect failed)"
	}
	return msg
}

func (e *SSHConnectionError) Unwrap() error { return e.Err }

// IsSSHConnectionError reports whether err represents a dropped SSH transport/session rather than command exit status.
func IsSSHConnectionError(err error) bool {
	var connErr *SSHConnectionError
	return errors.As(err, &connErr)
}

// SSHConnectionIssueSummary returns the short UI status for SSH transport errors.
func SSHConnectionIssueSummary(err error) string {
	if !IsSSHConnectionError(err) {
		return ""
	}
	return sshConnectionIssueSummary
}

// HostKeyMismatchError indicates the server presented a host key that conflicts
// with entries already recorded in known_hosts.
type HostKeyMismatchError struct {
	Hostname    string
	Fingerprint string
	Key         ssh.PublicKey
	UnknownHost bool
	Cause       error
}

func (e *HostKeyMismatchError) Error() string {
	if e == nil {
		return "ssh host key mismatch"
	}
	if e.UnknownHost {
		if e.Fingerprint == "" {
			return "ssh host key unknown"
		}
		return fmt.Sprintf("ssh host key unknown (%s)", e.Fingerprint)
	}
	if e.Fingerprint == "" {
		return "ssh host key mismatch"
	}
	return fmt.Sprintf("ssh host key mismatch (%s)", e.Fingerprint)
}

// NewSSHExecutor creates an SSHExecutor for target (user@host[:port]).
// identityFile, when non-empty, is used as a private key; when empty, ~/.ssh/id_rsa
// is tried first (like OpenSSH client), then SSH agent.
func NewSSHExecutor(target, identityFile string) (*SSHExecutor, string, error) {
	handle, user, hostPort, err := dialSSHClient(target, identityFile, "")
	if err != nil {
		return nil, "", err
	}

	label := user + "@" + hostPort
	exec := &SSHExecutor{
		user:   user,
		addr:   hostPort,
		target: target,
		auth:   sshAuthConfig{identityFile: identityFile},
	}
	if err := exec.replaceClient(handle); err != nil {
		return nil, "", err
	}
	return exec, label, nil
}

// NewSSHExecutorWithPassword creates an SSHExecutor using password-based auth in addition
// to optional identityFile and SSH agent.
func NewSSHExecutorWithPassword(target, identityFile, password string) (*SSHExecutor, string, error) {
	handle, user, hostPort, err := dialSSHClient(target, identityFile, password)
	if err != nil {
		return nil, "", err
	}

	label := user + "@" + hostPort
	exec := &SSHExecutor{
		user:   user,
		addr:   hostPort,
		target: target,
		auth:   sshAuthConfig{identityFile: identityFile, password: password},
	}
	if err := exec.replaceClient(handle); err != nil {
		return nil, "", err
	}
	return exec, label, nil
}

// SetTransportIssueHandler registers a callback for asynchronous SSH transport issue changes.
// The handler is called from background keepalive goroutines and must not block indefinitely.
func (e *SSHExecutor) SetTransportIssueHandler(fn func(string)) {
	if e == nil {
		return
	}
	e.mu.Lock()
	e.transportIssueFn = fn
	issue := e.transportIssue
	e.mu.Unlock()
	if fn != nil && issue != "" {
		fn(issue)
	}
}

// CopyLocalFileToRemote uploads a local file using the SCP protocol over the existing SSH
// session (remote runs scp -t). Parent directories must exist (e.g. mkdir -p) before calling.
func (e *SSHExecutor) CopyLocalFileToRemote(ctx context.Context, localPath, remotePath string) error {
	client, err := e.ensureClient("reconnecting before scp upload")
	if err != nil {
		return err
	}
	err = scpUpload(ctx, client, localPath, remotePath)
	if isSSHTransportError(err) {
		reErr := e.reconnect()
		if reErr != nil {
			e.markClientDisconnected(client)
		}
		return &SSHConnectionError{Op: "scp upload", Err: err, ReconnectTried: true, ReconnectSuccess: reErr == nil}
	}
	return err
}

// interruptRemoteSession asks the SSH server to signal the remote process. Non-interactive "sh -c"
// often ignores SIGINT alone; TERM/KILL improve teardown before [ssh.Session.Close] runs via defer.
func interruptRemoteSession(s *ssh.Session) {
	if s == nil {
		return
	}
	_ = s.Signal(ssh.SIGINT)
	_ = s.Signal(ssh.SIGTERM)
	_ = s.Signal(ssh.SIGKILL)
}

// Run implements CommandExecutor by executing the command via "sh -c" on the remote host.
func (e *SSHExecutor) Run(ctx context.Context, command string) (stdout, stderr string, exitCode int, err error) {
	client, err := e.ensureClient("reconnecting before remote command")
	if err != nil {
		return "", "", -1, err
	}

	session, err := client.NewSession()
	if err != nil {
		if isSSHTransportError(err) {
			if e.reconnect() == nil {
				client = e.currentClient()
				session, err = client.NewSession()
			} else {
				e.markClientDisconnected(client)
			}
		}
		if err != nil {
			return "", "", -1, wrapSSHConnectionError("opening remote session", err, false)
		}
	}
	var closeOnce sync.Once
	closeSession := func() {
		closeOnce.Do(func() { _ = session.Close() })
	}
	defer closeSession()

	var outBuf, errBuf bytes.Buffer
	session.Stdout = &outBuf
	session.Stderr = &errBuf

	done := make(chan error, 1)
	go func() {
		// Always use sh -c so behavior matches local executor.
		done <- session.Run("sh -c " + sshEscape(command))
	}()

	var runErr error
	select {
	case <-ctx.Done():
		interruptRemoteSession(session)
		closeSession()
		runErr = ctx.Err()
	case runErr = <-done:
	}

	if exitErr, ok := runErr.(*ssh.ExitError); ok {
		exitCode = exitErr.ExitStatus()
	} else if runErr != nil {
		exitCode = -1
	}
	if runErr != nil && isSSHTransportError(runErr) {
		reErr := e.reconnect()
		if reErr != nil {
			e.markClientDisconnected(client)
		}
		return outBuf.String(), errBuf.String(), exitCode, &SSHConnectionError{
			Op:               "running remote command",
			Err:              runErr,
			ReconnectTried:   true,
			ReconnectSuccess: reErr == nil,
		}
	}

	return outBuf.String(), errBuf.String(), exitCode, runErr
}

// RunStreaming runs the command on the remote host like [SSHExecutor.Run] but copies stdout/stderr to the given writers as data arrives.
func (e *SSHExecutor) RunStreaming(ctx context.Context, command string, stdout, stderr io.Writer) (exitCode int, err error) {
	client, err := e.ensureClient("reconnecting before streamed remote command")
	if err != nil {
		return -1, err
	}
	session, err := client.NewSession()
	if err != nil {
		if isSSHTransportError(err) {
			if e.reconnect() == nil {
				client = e.currentClient()
				session, err = client.NewSession()
			} else {
				e.markClientDisconnected(client)
			}
		}
		if err != nil {
			return -1, wrapSSHConnectionError("opening streamed remote session", err, false)
		}
	}
	var closeOnce sync.Once
	closeSession := func() {
		closeOnce.Do(func() { _ = session.Close() })
	}
	defer closeSession()
	session.Stdout = stdout
	session.Stderr = stderr

	done := make(chan error, 1)
	go func() {
		done <- session.Run("sh -c " + sshEscape(command))
	}()

	var runErr error
	select {
	case <-ctx.Done():
		interruptRemoteSession(session)
		closeSession()
		runErr = ctx.Err()
	case runErr = <-done:
	}

	if exitErr, ok := runErr.(*ssh.ExitError); ok {
		exitCode = exitErr.ExitStatus()
	} else if runErr != nil {
		exitCode = -1
	}
	if runErr != nil && isSSHTransportError(runErr) {
		reErr := e.reconnect()
		if reErr != nil {
			e.markClientDisconnected(client)
		}
		return exitCode, &SSHConnectionError{
			Op:               "running streamed remote command",
			Err:              runErr,
			ReconnectTried:   true,
			ReconnectSuccess: reErr == nil,
		}
	}
	return exitCode, runErr
}

// Close closes the underlying SSH client connection.
func (e *SSHExecutor) Close() error {
	e.mu.Lock()
	client := e.client
	closeFn := e.clientClose
	stop := e.keepAliveStop
	recoveryStop := e.recoveryStop
	e.client = nil
	e.clientClose = nil
	e.keepAliveStop = nil
	e.recoveryStop = nil
	e.closed = true
	e.mu.Unlock()
	if stop != nil {
		close(stop)
	}
	if recoveryStop != nil {
		close(recoveryStop)
	}
	if closeFn != nil {
		return closeFn()
	}
	if client != nil {
		return client.Close()
	}
	return nil
}

func (e *SSHExecutor) currentClient() *ssh.Client {
	if e == nil {
		return nil
	}
	e.mu.Lock()
	defer e.mu.Unlock()
	return e.client
}

func (e *SSHExecutor) isClosed() bool {
	if e == nil {
		return true
	}
	e.mu.Lock()
	defer e.mu.Unlock()
	return e.closed
}

func (e *SSHExecutor) ensureClient(op string) (*ssh.Client, error) {
	client := e.currentClient()
	if client != nil {
		return client, nil
	}
	if e.isClosed() {
		return nil, errors.New("ssh client is not connected")
	}
	if err := e.reconnect(); err != nil {
		return nil, &SSHConnectionError{Op: op, Err: err, ReconnectTried: true, ReconnectSuccess: false}
	}
	client = e.currentClient()
	if client == nil {
		return nil, &SSHConnectionError{Op: op, Err: errors.New("ssh client is not connected"), ReconnectTried: true, ReconnectSuccess: false}
	}
	return client, nil
}

func (e *SSHExecutor) reconnect() error {
	if e == nil {
		return errors.New("ssh executor is nil")
	}
	e.reconnectMu.Lock()
	defer e.reconnectMu.Unlock()
	if e.isClosed() {
		return errors.New("ssh executor is closed")
	}
	handle, _, _, err := dialSSHClient(e.target, e.auth.identityFile, e.auth.password)
	if err != nil {
		return err
	}
	if err := e.replaceClient(handle); err != nil {
		return err
	}
	e.reportTransportIssue("")
	return nil
}

func (e *SSHExecutor) replaceClient(handle sshClientHandle) error {
	if e == nil {
		return errors.New("ssh executor is nil")
	}
	client := handle.client
	if client == nil {
		return errors.New("ssh client is nil")
	}
	closeFn := handle.close
	if closeFn == nil {
		closeFn = client.Close
	}
	stop := make(chan struct{})
	e.mu.Lock()
	if e.closed {
		e.mu.Unlock()
		_ = closeFn()
		return errors.New("ssh executor is closed")
	}
	old := e.client
	oldClose := e.clientClose
	e.client = client
	e.clientClose = closeFn
	oldStop := e.keepAliveStop
	oldRecoveryStop := e.recoveryStop
	e.keepAliveStop = stop
	e.recoveryStop = nil
	e.mu.Unlock()
	if oldStop != nil {
		close(oldStop)
	}
	if oldRecoveryStop != nil {
		close(oldRecoveryStop)
	}
	if oldClose != nil {
		_ = oldClose()
	} else if old != nil {
		_ = old.Close()
	}
	go e.keepAliveLoop(client, stop)
	return nil
}

func (e *SSHExecutor) keepAliveLoop(client *ssh.Client, stop <-chan struct{}) {
	ticker := time.NewTicker(sshAppKeepAliveInterval)
	defer ticker.Stop()
	for {
		select {
		case <-stop:
			return
		case <-ticker.C:
		}
		if err := sendSSHKeepAlive(client, sshAppKeepAliveReqTimeout); err != nil {
			select {
			case <-stop:
				return
			default:
			}
			e.markKeepAliveFailed(client, err)
			return
		}
		e.reportTransportIssue("")
	}
}

func sendSSHKeepAlive(client *ssh.Client, timeout time.Duration) error {
	if client == nil {
		return errors.New("ssh client is nil")
	}
	done := make(chan error, 1)
	go func() {
		_, _, err := client.SendRequest("keepalive@openssh.com", true, nil)
		done <- err
	}()
	timer := time.NewTimer(timeout)
	defer timer.Stop()
	select {
	case err := <-done:
		return err
	case <-timer.C:
		_ = client.Close()
		return fmt.Errorf("ssh keepalive response timed out after %s", timeout)
	}
}

func (e *SSHExecutor) markKeepAliveFailed(client *ssh.Client, err error) {
	_ = err
	e.markClientDisconnected(client)
}

func (e *SSHExecutor) markClientDisconnected(client *ssh.Client) {
	if e == nil {
		return
	}
	e.mu.Lock()
	if e.client != client {
		e.mu.Unlock()
		return
	}
	stop := e.keepAliveStop
	e.client = nil
	e.keepAliveStop = nil
	e.mu.Unlock()
	if stop != nil {
		close(stop)
	}
	_ = client.Close()
	e.reportTransportIssue(sshConnectionIssueSummary)
	e.startRecoveryLoop()
}

func (e *SSHExecutor) startRecoveryLoop() {
	if e == nil {
		return
	}
	e.mu.Lock()
	if e.closed || e.client != nil || e.recoveryStop != nil {
		e.mu.Unlock()
		return
	}
	stop := make(chan struct{})
	e.recoveryStop = stop
	e.mu.Unlock()
	go e.recoveryLoop(stop)
}

func (e *SSHExecutor) recoveryLoop(stop <-chan struct{}) {
	for {
		timer := time.NewTimer(sshRecoveryRetryInterval)
		select {
		case <-stop:
			timer.Stop()
			return
		case <-timer.C:
		}
		if err := e.reconnect(); err == nil {
			return
		}
		select {
		case <-stop:
			return
		default:
		}
	}
}

func (e *SSHExecutor) reportTransportIssue(issue string) {
	if e == nil {
		return
	}
	issue = strings.TrimSpace(issue)
	e.mu.Lock()
	if e.transportIssue == issue {
		e.mu.Unlock()
		return
	}
	e.transportIssue = issue
	fn := e.transportIssueFn
	e.mu.Unlock()
	if fn != nil {
		fn(issue)
	}
}

func dialSSHClient(target, identityFile, password string) (sshClientHandle, string, string, error) {
	connectTarget, err := parseSSHConnectTarget(target, false)
	if err != nil {
		return sshClientHandle{}, "", "", err
	}
	clientConfig, err := newSSHClientConfig(connectTarget.user, identityFile, password)
	if err != nil {
		return sshClientHandle{}, "", "", err
	}
	proxyJump := resolveTargetProxyJump(target)
	if strings.TrimSpace(proxyJump) == "" {
		handle, err := dialDirectSSHClient(connectTarget.hostPort, clientConfig)
		if err != nil {
			return sshClientHandle{}, "", "", err
		}
		return handle, connectTarget.user, connectTarget.hostPort, nil
	}
	jumpTarget, jumpIdentityFile, err := resolveProxyJumpTarget(proxyJump)
	if err != nil {
		return sshClientHandle{}, "", "", err
	}
	jumpConnectTarget, err := parseSSHConnectTarget(jumpTarget, true)
	if err != nil {
		return sshClientHandle{}, "", "", err
	}
	jumpConfig, err := newSSHClientConfig(jumpConnectTarget.user, jumpIdentityFile, "")
	if err != nil {
		return sshClientHandle{}, "", "", err
	}
	jumpHandle, err := dialDirectSSHClient(jumpConnectTarget.hostPort, jumpConfig)
	if err != nil {
		return sshClientHandle{}, "", "", err
	}
	conn, err := jumpHandle.client.Dial("tcp", connectTarget.hostPort)
	if err != nil {
		_ = jumpHandle.close()
		return sshClientHandle{}, "", "", err
	}
	sshConn, chans, reqs, err := ssh.NewClientConn(conn, connectTarget.hostPort, clientConfig)
	if err != nil {
		_ = conn.Close()
		_ = jumpHandle.close()
		return sshClientHandle{}, "", "", err
	}
	client := ssh.NewClient(sshConn, chans, reqs)
	var closeOnce sync.Once
	closeFn := func() error {
		var closeErr error
		closeOnce.Do(func() {
			if err := client.Close(); err != nil {
				closeErr = err
			}
			if err := jumpHandle.close(); err != nil && closeErr == nil {
				closeErr = err
			}
		})
		return closeErr
	}
	return sshClientHandle{client: client, close: closeFn}, connectTarget.user, connectTarget.hostPort, nil
}

func newSSHClientConfig(user, identityFile, password string) (*ssh.ClientConfig, error) {
	hostKeyCallback, err := loadKnownHosts()
	if err != nil {
		hostKeyCallback = ssh.InsecureIgnoreHostKey()
	} else {
		knownHostsCB := hostKeyCallback
		hostKeyCallback = func(hostname string, remote net.Addr, key ssh.PublicKey) error {
			if cbErr := knownHostsCB(hostname, remote, key); cbErr != nil {
				var keyErr *knownhosts.KeyError
				if errors.As(cbErr, &keyErr) && len(keyErr.Want) > 0 {
					return &HostKeyMismatchError{
						Hostname:    hostname,
						Fingerprint: ssh.FingerprintSHA256(key),
						Key:         key,
						Cause:       cbErr,
					}
				}
				if errors.As(cbErr, &keyErr) && len(keyErr.Want) == 0 {
					return &HostKeyMismatchError{
						Hostname:    hostname,
						Fingerprint: ssh.FingerprintSHA256(key),
						Key:         key,
						UnknownHost: true,
						Cause:       cbErr,
					}
				}
				return cbErr
			}
			return nil
		}
	}

	authMethods, err := buildAuthMethods(identityFile, password)
	if err != nil {
		return nil, err
	}
	if len(authMethods) == 0 {
		if password != "" {
			return nil, errors.New("no SSH authentication methods available (identity file, agent, or password)")
		}
		return nil, errors.New("no SSH authentication methods available (identity file, agent, or default keys)")
	}

	return &ssh.ClientConfig{
		User:            user,
		Auth:            authMethods,
		HostKeyCallback: hostKeyCallback,
		Timeout:         sshDialTimeout,
	}, nil
}

func dialDirectSSHClient(hostPort string, clientConfig *ssh.ClientConfig) (sshClientHandle, error) {
	dialer := net.Dialer{
		Timeout: sshDialTimeout,
		KeepAliveConfig: net.KeepAliveConfig{
			Enable:   true,
			Idle:     sshTCPKeepAliveIdle,
			Interval: sshTCPKeepAliveInterval,
			Count:    sshTCPKeepAliveCount,
		},
	}
	conn, err := dialer.DialContext(context.Background(), "tcp", hostPort)
	if err != nil {
		return sshClientHandle{}, err
	}
	_ = conn.SetDeadline(time.Now().Add(sshDialTimeout))
	sshConn, chans, reqs, err := ssh.NewClientConn(conn, hostPort, clientConfig)
	if err != nil {
		_ = conn.Close()
		return sshClientHandle{}, err
	}
	_ = conn.SetDeadline(time.Time{})
	client := ssh.NewClient(sshConn, chans, reqs)
	return sshClientHandle{client: client, close: client.Close}, nil
}

func resolveTargetProxyJump(target string) string {
	sshHost, ok, err := config.ResolveSSHConfigHost(target)
	if err != nil || !ok {
		return ""
	}
	return strings.TrimSpace(sshHost.ProxyJump)
}

func resolveProxyJumpTarget(raw string) (target string, identityFile string, err error) {
	raw = strings.TrimSpace(raw)
	switch {
	case raw == "":
		return "", "", nil
	case strings.Contains(raw, ","):
		return "", "", fmt.Errorf("ssh ProxyJump chain %q is not supported", raw)
	}
	if sshHost, ok, err := config.ResolveSSHConfigHost(raw); err == nil && ok {
		if strings.TrimSpace(sshHost.ProxyJump) != "" {
			name := strings.TrimSpace(sshHost.Alias)
			if name == "" {
				name = sshHost.Target
			}
			return "", "", fmt.Errorf("ssh ProxyJump chain via %q is not supported", name)
		}
		return strings.TrimSpace(sshHost.Target), strings.TrimSpace(sshHost.IdentityFile), nil
	}
	connectTarget, err := parseSSHConnectTarget(raw, true)
	if err != nil {
		return "", "", fmt.Errorf("invalid ssh ProxyJump target %q: %w", raw, err)
	}
	return formatSSHConnectTarget(connectTarget), "", nil
}

func wrapSSHConnectionError(op string, err error, reconnectSuccess bool) error {
	if !isSSHTransportError(err) {
		return err
	}
	return &SSHConnectionError{Op: op, Err: err, ReconnectSuccess: reconnectSuccess}
}

func isSSHTransportError(err error) bool {
	if err == nil {
		return false
	}
	if errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded) {
		return false
	}
	var exitErr *ssh.ExitError
	if errors.As(err, &exitErr) {
		return false
	}
	if errors.Is(err, io.EOF) || errors.Is(err, io.ErrUnexpectedEOF) {
		return true
	}
	if errors.Is(err, net.ErrClosed) || errors.Is(err, syscall.EPIPE) || errors.Is(err, syscall.ECONNRESET) || errors.Is(err, syscall.ECONNABORTED) || errors.Is(err, syscall.ETIMEDOUT) {
		return true
	}
	var netErr net.Error
	if errors.As(err, &netErr) {
		return true
	}
	s := strings.ToLower(err.Error())
	return strings.Contains(s, "broken pipe") ||
		strings.Contains(s, "connection reset") ||
		strings.Contains(s, "connection aborted") ||
		strings.Contains(s, "connection refused") ||
		strings.Contains(s, "connection lost") ||
		strings.Contains(s, "use of closed network connection") ||
		strings.Contains(s, "transport is closing") ||
		strings.Contains(s, "client connection lost") ||
		strings.Contains(s, "handshake failed: eof") ||
		s == "eof"
}

func parseSSHConnectTarget(target string, allowImplicitUser bool) (sshConnectTarget, error) {
	if !allowImplicitUser || strings.Contains(target, "@") {
		user, hostPort, err := parseUserHost(target)
		if err != nil {
			return sshConnectTarget{}, err
		}
		return sshConnectTarget{user: user, hostPort: hostPort}, nil
	}
	target = strings.TrimSpace(target)
	if target == "" {
		return sshConnectTarget{}, errors.New("empty SSH target")
	}
	user := strings.TrimSpace(os.Getenv("USER"))
	if user == "" {
		user = strings.TrimSpace(os.Getenv("LOGNAME"))
	}
	if user == "" {
		return sshConnectTarget{}, errors.New("ssh target must include username (user@host or user@host:port)")
	}
	if !strings.Contains(target, ":") {
		target = net.JoinHostPort(target, "22")
	}
	return sshConnectTarget{user: user, hostPort: target}, nil
}

func formatSSHConnectTarget(target sshConnectTarget) string {
	if target.hostPort == "" {
		return ""
	}
	host, port, err := net.SplitHostPort(target.hostPort)
	if err != nil {
		return target.user + "@" + target.hostPort
	}
	if port == "22" {
		return target.user + "@" + host
	}
	return target.user + "@" + net.JoinHostPort(host, port)
}

// parseUserHost parses "user@host[:port]" into user and host:port.
// The user must be explicit; it does not fall back to environment defaults.
func parseUserHost(target string) (string, string, error) {
	target = strings.TrimSpace(target)
	if target == "" {
		return "", "", errors.New("empty SSH target")
	}
	if !strings.Contains(target, "@") {
		return "", "", errors.New("ssh target must include username (user@host or user@host:port)")
	}
	parts := strings.SplitN(target, "@", 2)
	user := strings.TrimSpace(parts[0])
	hostPart := strings.TrimSpace(parts[1])
	if user == "" || hostPart == "" {
		return "", "", errors.New("ssh target must include username (user@host or user@host:port)")
	}
	if !strings.Contains(hostPart, ":") {
		hostPart = net.JoinHostPort(hostPart, "22")
	}
	return user, hostPart, nil
}

func loadKnownHosts() (ssh.HostKeyCallback, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return nil, err
	}
	path := filepath.Join(home, ".ssh", "known_hosts")
	return knownhosts.New(path)
}

// UpdateKnownHost replaces entries for hostname and writes the given public key to known_hosts.
func UpdateKnownHost(hostname string, key ssh.PublicKey) error {
	if strings.TrimSpace(hostname) == "" || key == nil {
		return errors.New("hostname and key are required")
	}
	path, err := knownHostsPath()
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(path), 0700); err != nil {
		return err
	}
	raw, err := os.ReadFile(path)
	if err != nil && !os.IsNotExist(err) {
		return err
	}
	lines := strings.Split(string(raw), "\n")
	out := make([]string, 0, len(lines)+1)
	for _, line := range lines {
		trimmed := strings.TrimSpace(line)
		if trimmed == "" || strings.HasPrefix(trimmed, "#") {
			if trimmed != "" {
				out = append(out, line)
			}
			continue
		}
		fields := strings.Fields(line)
		if len(fields) == 0 {
			continue
		}
		hosts := strings.Split(fields[0], ",")
		matched := false
		for _, h := range hosts {
			if strings.TrimSpace(h) == hostname {
				matched = true
				break
			}
		}
		if matched {
			continue
		}
		out = append(out, line)
	}
	out = append(out, knownhosts.Line([]string{hostname}, key))
	content := strings.Join(out, "\n")
	if !strings.HasSuffix(content, "\n") {
		content += "\n"
	}
	return os.WriteFile(path, []byte(content), 0600)
}

func knownHostsPath() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(home, ".ssh", "known_hosts"), nil
}

// defaultIdentityPath is the path tried when identityFile is empty, matching OpenSSH client behavior.
const defaultIdentityPath = "~/.ssh/id_rsa"

func buildAuthMethods(identityFile, password string) ([]ssh.AuthMethod, error) {
	var methods []ssh.AuthMethod

	// Identity file (private key): use given path, or when empty try default ~/.ssh/id_rsa like ssh client
	pathToTry := identityFile
	if pathToTry == "" {
		pathToTry = defaultIdentityPath
	}
	if pathToTry != "" {
		keyPath := pathToTry
		if strings.HasPrefix(keyPath, "~") {
			home, err := os.UserHomeDir()
			if err == nil {
				keyPath = filepath.Join(home, strings.TrimPrefix(keyPath, "~"))
			}
		}
		if data, err := os.ReadFile(keyPath); err == nil {
			if signer, err := ssh.ParsePrivateKey(data); err == nil {
				methods = append(methods, ssh.PublicKeys(signer))
			}
		}
	}

	// SSH agent
	if sock := os.Getenv("SSH_AUTH_SOCK"); sock != "" {
		if conn, err := net.Dial("unix", sock); err == nil {
			ag := agent.NewClient(conn)
			methods = append(methods, ssh.PublicKeysCallback(ag.Signers))
		}
	}

	// Password
	if password != "" {
		methods = append(methods, ssh.Password(password))
	}

	return methods, nil
}

// sshEscape wraps a command as one single-quoted shell word for use in `sh -c <arg>`.
// This prevents the outer shell from expanding `$VAR`, `$(...)`, backticks, etc. before
// the inner shell receives the original script.
func sshEscape(command string) string {
	return `'` + strings.ReplaceAll(command, `'`, `'"'"'`) + `'`
}
