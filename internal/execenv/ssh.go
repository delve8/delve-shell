package execenv

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"net"
	"os"
	"path/filepath"
	"strings"
	"time"

	"golang.org/x/crypto/ssh"
	"golang.org/x/crypto/ssh/agent"
	"golang.org/x/crypto/ssh/knownhosts"
)

// SSHExecutor runs commands on a remote host via SSH.
// It keeps a single SSH client connection and opens a new session per Run.
type SSHExecutor struct {
	client *ssh.Client
	user   string
	addr   string // host:port
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

// NewSSHExecutor creates an SSHExecutor for target (user@host or host[:port]).
// identityFile, when non-empty, is used as a private key; when empty, ~/.ssh/id_rsa
// is tried first (like OpenSSH client), then SSH agent.
func NewSSHExecutor(target, identityFile string) (*SSHExecutor, string, error) {
	user, hostPort, err := parseUserHost(target)
	if err != nil {
		return nil, "", err
	}

	hostKeyCallback, err := loadKnownHosts()
	if err != nil {
		// Fallback to accepting any host key; caller should tighten this in high-security environments.
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

	authMethods, err := buildAuthMethods(identityFile, "")
	if err != nil {
		return nil, "", err
	}
	if len(authMethods) == 0 {
		return nil, "", errors.New("no SSH authentication methods available (identity file, agent, or default keys)")
	}

	clientConfig := &ssh.ClientConfig{
		User:            user,
		Auth:            authMethods,
		HostKeyCallback: hostKeyCallback,
		Timeout:         10 * time.Second,
	}

	client, err := ssh.Dial("tcp", hostPort, clientConfig)
	if err != nil {
		return nil, "", err
	}

	label := user + "@" + hostPort
	return &SSHExecutor{
		client: client,
		user:   user,
		addr:   hostPort,
	}, label, nil
}

// NewSSHExecutorWithPassword creates an SSHExecutor using password-based auth in addition
// to optional identityFile and SSH agent.
func NewSSHExecutorWithPassword(target, identityFile, password string) (*SSHExecutor, string, error) {
	user, hostPort, err := parseUserHost(target)
	if err != nil {
		return nil, "", err
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
		return nil, "", err
	}
	if len(authMethods) == 0 {
		return nil, "", errors.New("no SSH authentication methods available (identity file, agent, or password) ")
	}

	clientConfig := &ssh.ClientConfig{
		User:            user,
		Auth:            authMethods,
		HostKeyCallback: hostKeyCallback,
		Timeout:         10 * time.Second,
	}

	client, err := ssh.Dial("tcp", hostPort, clientConfig)
	if err != nil {
		return nil, "", err
	}

	label := user + "@" + hostPort
	return &SSHExecutor{
		client: client,
		user:   user,
		addr:   hostPort,
	}, label, nil
}

// Run implements CommandExecutor by executing the command via "sh -c" on the remote host.
func (e *SSHExecutor) Run(ctx context.Context, command string) (stdout, stderr string, exitCode int, err error) {
	if e.client == nil {
		return "", "", -1, errors.New("ssh client is not connected")
	}

	session, err := e.client.NewSession()
	if err != nil {
		return "", "", -1, err
	}
	defer session.Close()

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
		_ = session.Signal(ssh.SIGINT)
		runErr = ctx.Err()
	case runErr = <-done:
	}

	if exitErr, ok := runErr.(*ssh.ExitError); ok {
		exitCode = exitErr.ExitStatus()
	} else if runErr != nil {
		exitCode = -1
	}

	return outBuf.String(), errBuf.String(), exitCode, runErr
}

// Close closes the underlying SSH client connection.
func (e *SSHExecutor) Close() error {
	if e.client != nil {
		return e.client.Close()
	}
	return nil
}

// parseUserHost parses "user@host[:port]" or "host[:port]" into user and host:port.
// When user is missing, it falls back to $USER.
func parseUserHost(target string) (string, string, error) {
	target = strings.TrimSpace(target)
	if target == "" {
		return "", "", errors.New("empty SSH target")
	}
	user := os.Getenv("USER")
	hostPart := target
	if strings.Contains(target, "@") {
		parts := strings.SplitN(target, "@", 2)
		if parts[0] != "" {
			user = parts[0]
		}
		hostPart = parts[1]
	}
	if user == "" {
		return "", "", errors.New("cannot determine SSH user (no user in target and $USER is empty)")
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

// sshEscape escapes double quotes in a shell command used inside sh -c "...".
func sshEscape(command string) string {
	// Conservative escaping: wrap in double quotes and escape existing ones.
	return `"` + strings.ReplaceAll(command, `"`, `\"`) + `"`
}
