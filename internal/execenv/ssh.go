package execenv

import (
	"bytes"
	"context"
	"errors"
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
