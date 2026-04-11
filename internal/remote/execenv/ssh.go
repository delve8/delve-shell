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

	"golang.org/x/crypto/ssh"
	"golang.org/x/crypto/ssh/agent"
	"golang.org/x/crypto/ssh/knownhosts"
)

// SSHExecutor runs commands on a remote host via SSH.
// It keeps a single SSH client connection and opens a new session per Run.
type SSHExecutor struct {
	mu     sync.Mutex
	client *ssh.Client
	user   string
	addr   string // host:port
	target string
	auth   sshAuthConfig
}

type sshAuthConfig struct {
	identityFile string
	password     string
}

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
	client, user, hostPort, err := dialSSHClient(target, identityFile, "")
	if err != nil {
		return nil, "", err
	}

	label := user + "@" + hostPort
	return &SSHExecutor{
		client: client,
		user:   user,
		addr:   hostPort,
		target: target,
		auth:   sshAuthConfig{identityFile: identityFile},
	}, label, nil
}

// NewSSHExecutorWithPassword creates an SSHExecutor using password-based auth in addition
// to optional identityFile and SSH agent.
func NewSSHExecutorWithPassword(target, identityFile, password string) (*SSHExecutor, string, error) {
	client, user, hostPort, err := dialSSHClient(target, identityFile, password)
	if err != nil {
		return nil, "", err
	}

	label := user + "@" + hostPort
	return &SSHExecutor{
		client: client,
		user:   user,
		addr:   hostPort,
		target: target,
		auth:   sshAuthConfig{identityFile: identityFile, password: password},
	}, label, nil
}

// CopyLocalFileToRemote uploads a local file using the SCP protocol over the existing SSH
// session (remote runs scp -t). Parent directories must exist (e.g. mkdir -p) before calling.
func (e *SSHExecutor) CopyLocalFileToRemote(ctx context.Context, localPath, remotePath string) error {
	client := e.currentClient()
	if e == nil || client == nil {
		return errors.New("ssh client is not connected")
	}
	err := scpUpload(ctx, client, localPath, remotePath)
	if isSSHTransportError(err) {
		reErr := e.reconnect()
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
	client := e.currentClient()
	if client == nil {
		return "", "", -1, errors.New("ssh client is not connected")
	}

	session, err := client.NewSession()
	if err != nil {
		if isSSHTransportError(err) && e.reconnect() == nil {
			client = e.currentClient()
			session, err = client.NewSession()
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
	client := e.currentClient()
	if client == nil {
		return -1, errors.New("ssh client is not connected")
	}
	session, err := client.NewSession()
	if err != nil {
		if isSSHTransportError(err) && e.reconnect() == nil {
			client = e.currentClient()
			session, err = client.NewSession()
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
	e.client = nil
	e.mu.Unlock()
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

func (e *SSHExecutor) reconnect() error {
	if e == nil {
		return errors.New("ssh executor is nil")
	}
	e.mu.Lock()
	defer e.mu.Unlock()
	client, _, _, err := dialSSHClient(e.target, e.auth.identityFile, e.auth.password)
	if err != nil {
		return err
	}
	old := e.client
	e.client = client
	if old != nil {
		_ = old.Close()
	}
	return nil
}

func dialSSHClient(target, identityFile, password string) (*ssh.Client, string, string, error) {
	user, hostPort, err := parseUserHost(target)
	if err != nil {
		return nil, "", "", err
	}

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
		return nil, "", "", err
	}
	if len(authMethods) == 0 {
		if password != "" {
			return nil, "", "", errors.New("no SSH authentication methods available (identity file, agent, or password)")
		}
		return nil, "", "", errors.New("no SSH authentication methods available (identity file, agent, or default keys)")
	}

	clientConfig := &ssh.ClientConfig{
		User:            user,
		Auth:            authMethods,
		HostKeyCallback: hostKeyCallback,
		Timeout:         10 * time.Second,
	}

	client, err := ssh.Dial("tcp", hostPort, clientConfig)
	if err != nil {
		return nil, "", "", err
	}
	return client, user, hostPort, nil
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
