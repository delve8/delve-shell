package e2e

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"regexp"
	"strings"
	"syscall"
	"time"

	"github.com/creack/pty"
)

// ansiStrip removes ANSI escape sequences for stable substring matching on terminal output.
var ansiStrip = regexp.MustCompile(`\x1b\[[0-9;]*[a-zA-Z]|\x1b[\[?][0-9;]*[a-zA-Z]?`)

func stripANSI(s string) string { return ansiStrip.ReplaceAllString(s, "") }

var (
	termQueryCursor = []byte("\x1b[6n")
	termReplyCursor = []byte("\x1b[1;1R")
)

// Spawn starts the binary in a PTY; returns the PTY master and the process. Caller must Close and Kill.
// Sets a non-zero terminal size so the TUI receives WindowSizeMsg and renders (Bubble Tea needs dimensions).
func Spawn(binaryPath string, env []string) (*os.File, *exec.Cmd, error) {
	cmd := exec.Command(binaryPath)
	cmd.Env = env
	ptmx, err := pty.Start(cmd)
	if err != nil {
		return nil, nil, fmt.Errorf("pty start: %w", err)
	}
	if err := pty.Setsize(ptmx, &pty.Winsize{Rows: 24, Cols: 80}); err != nil {
		_ = ptmx.Close()
		_ = cmd.Process.Kill()
		return nil, nil, fmt.Errorf("pty setsize: %w", err)
	}
	if err := syscall.SetNonblock(int(ptmx.Fd()), true); err != nil {
		_ = ptmx.Close()
		_ = cmd.Process.Kill()
		return nil, nil, fmt.Errorf("pty setnonblock: %w", err)
	}
	return ptmx, cmd, nil
}

// ReadUntil reads from rw until output contains substr or timeout. Returns what was read.
// If rw implements SetReadDeadline (e.g. *os.File), it is used for timeout; otherwise polls until deadline.
func ReadUntil(rw io.ReadWriter, substr string, timeout time.Duration) (string, error) {
	var buf bytes.Buffer
	deadline := time.Now().Add(timeout)
	b := make([]byte, 256)
	for {
		if !time.Now().Before(deadline) {
			return buf.String(), nil
		}
		n, err := rw.Read(b)
		if n > 0 {
			buf.Write(b[:n])
			respondToTerminalQueries(rw, buf.Bytes())
			if strings.Contains(stripANSI(buf.String()), substr) {
				return buf.String(), nil
			}
		}
		if err != nil {
			if isRetryableRead(err) {
				time.Sleep(20 * time.Millisecond)
				continue
			}
			return buf.String(), err
		}
		time.Sleep(10 * time.Millisecond)
	}
}

// WriteLine writes one line to w (appends \r\n for TUI Enter).
func WriteLine(w io.Writer, line string) error {
	_, err := w.Write([]byte(line + "\r\n"))
	return err
}

// ReadUntilAny reads from rw until output contains any of substrings or timeout. Returns content and matched index (-1 if timeout).
func ReadUntilAny(rw io.ReadWriter, substrings []string, timeout time.Duration) (string, int, error) {
	var buf bytes.Buffer
	deadline := time.Now().Add(timeout)
	b := make([]byte, 256)
	for {
		if !time.Now().Before(deadline) {
			return buf.String(), -1, nil
		}
		n, err := rw.Read(b)
		if n > 0 {
			buf.Write(b[:n])
			respondToTerminalQueries(rw, buf.Bytes())
			s := stripANSI(buf.String())
			for i, sub := range substrings {
				if strings.Contains(s, sub) {
					return buf.String(), i, nil
				}
			}
		}
		if err != nil {
			if isRetryableRead(err) {
				time.Sleep(20 * time.Millisecond)
				continue
			}
			return buf.String(), -1, err
		}
		time.Sleep(10 * time.Millisecond)
	}
}

func respondToTerminalQueries(w io.Writer, buf []byte) {
	if bytes.Contains(buf, termQueryCursor) {
		_, _ = w.Write(termReplyCursor)
	}
}

func isRetryableRead(err error) bool {
	return errors.Is(err, os.ErrDeadlineExceeded) ||
		errors.Is(err, syscall.EAGAIN) ||
		errors.Is(err, syscall.EWOULDBLOCK) ||
		strings.Contains(err.Error(), "deadline") ||
		strings.Contains(err.Error(), "timeout") ||
		strings.Contains(err.Error(), "resource temporarily unavailable")
}
