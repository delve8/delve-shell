//go:build !windows

package execenv

import (
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"os/signal"
	"syscall"

	"golang.org/x/crypto/ssh"
	"golang.org/x/sys/unix"
	"golang.org/x/term"
)

// RunInteractiveSSHShell runs an interactive remote shell on an existing SSH connection.
// It attaches os.Stdin/os.Stdout to a new session with a PTY. If exec is not *SSHExecutor, returns an error.
func RunInteractiveSSHShell(ctx context.Context, exec CommandExecutor) error {
	sshExec, ok := exec.(*SSHExecutor)
	if !ok {
		return fmt.Errorf("remote subshell requires an active SSH connection")
	}
	return sshExec.runInteractiveShell(ctx)
}

func (e *SSHExecutor) runInteractiveShell(ctx context.Context) error {
	client, err := e.ensureClient("reconnecting before remote interactive shell")
	if err != nil {
		return err
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
			return wrapSSHConnectionError("opening remote interactive shell", err, false)
		}
	}
	defer session.Close()

	fd := int(os.Stdin.Fd())
	if !term.IsTerminal(fd) {
		return errors.New("stdin is not a terminal")
	}

	cols, rows := termSize(fd)
	modes := ssh.TerminalModes{
		ssh.ECHO:          1,
		ssh.TTY_OP_ISPEED: 14400,
		ssh.TTY_OP_OSPEED: 14400,
	}
	// RequestPty(term, height, width, modes)
	if err := session.RequestPty("xterm-256color", rows, cols, modes); err != nil {
		return err
	}

	stdinPipe, err := session.StdinPipe()
	if err != nil {
		return err
	}
	stdoutPipe, err := session.StdoutPipe()
	if err != nil {
		return err
	}

	oldState, err := term.MakeRaw(fd)
	if err != nil {
		return err
	}
	defer func() { _ = term.Restore(fd, oldState) }()

	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	sigwinch := make(chan os.Signal, 8)
	signal.Notify(sigwinch, syscall.SIGWINCH)
	defer signal.Stop(sigwinch)

	go func() {
		for {
			select {
			case <-ctx.Done():
				return
			case <-sigwinch:
				c, r := termSize(fd)
				_ = session.WindowChange(r, c)
			}
		}
	}()

	go func() {
		_, _ = io.Copy(stdinPipe, os.Stdin)
		_ = stdinPipe.Close()
	}()

	go func() {
		_, _ = io.Copy(os.Stdout, stdoutPipe)
	}()

	if err := session.Start("bash -l"); err != nil {
		return err
	}

	waitDone := make(chan error, 1)
	go func() {
		waitDone <- session.Wait()
	}()

	select {
	case <-ctx.Done():
		_ = session.Close()
		return ctx.Err()
	case err := <-waitDone:
		if err != nil {
			var exitErr *ssh.ExitError
			if errors.As(err, &exitErr) {
				return nil
			}
		}
		return err
	}
}

func termSize(fd int) (cols, rows int) {
	ws, err := unix.IoctlGetWinsize(fd, unix.TIOCGWINSZ)
	if err != nil {
		return 80, 24
	}
	return int(ws.Col), int(ws.Row)
}
