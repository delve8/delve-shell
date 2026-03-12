package e2e

import (
	"bytes"
	"fmt"
	"io"
	"os"
	"os/exec"
	"regexp"
	"strings"
	"time"

	"github.com/creack/pty"
)

// ansiStrip removes ANSI escape sequences for stable substring matching on terminal output.
var ansiStrip = regexp.MustCompile(`\x1b\[[0-9;]*[a-zA-Z]|\x1b[\[?][0-9;]*[a-zA-Z]?`)

func stripANSI(s string) string { return ansiStrip.ReplaceAllString(s, "") }

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
	return ptmx, cmd, nil
}

// ReadUntil reads from r until output contains substr or timeout. Returns what was read.
// If r implements SetReadDeadline (e.g. *os.File), it is used for timeout; otherwise polls until deadline.
func ReadUntil(r io.Reader, substr string, timeout time.Duration) (string, error) {
	type deadliner interface{ SetReadDeadline(t time.Time) error }
	var buf bytes.Buffer
	deadline := time.Now().Add(timeout)
	b := make([]byte, 256)
	for time.Now().Before(deadline) {
		if d, ok := r.(deadliner); ok {
			_ = d.SetReadDeadline(time.Now().Add(500 * time.Millisecond))
		}
		n, err := r.Read(b)
		if n > 0 {
			buf.Write(b[:n])
			if strings.Contains(stripANSI(buf.String()), substr) {
				return buf.String(), nil
			}
		}
		if err != nil {
			if !strings.Contains(err.Error(), "deadline") {
				return buf.String(), err
			}
		}
	}
	return buf.String(), nil
}

// WriteLine writes one line to w (appends \r\n for TUI Enter).
func WriteLine(w io.Writer, line string) error {
	_, err := w.Write([]byte(line + "\r\n"))
	return err
}

// ReadUntilAny reads from r until output contains any of substrings or timeout. Returns content and matched index (-1 if timeout).
func ReadUntilAny(r io.Reader, substrings []string, timeout time.Duration) (string, int, error) {
	type deadliner interface{ SetReadDeadline(t time.Time) error }
	var buf bytes.Buffer
	deadline := time.Now().Add(timeout)
	b := make([]byte, 256)
	for time.Now().Before(deadline) {
		if d, ok := r.(deadliner); ok {
			_ = d.SetReadDeadline(time.Now().Add(500 * time.Millisecond))
		}
		n, err := r.Read(b)
		if n > 0 {
			buf.Write(b[:n])
			s := stripANSI(buf.String())
			for i, sub := range substrings {
				if strings.Contains(s, sub) {
					return buf.String(), i, nil
				}
			}
		}
		if err != nil && !strings.Contains(err.Error(), "deadline") {
			return buf.String(), -1, err
		}
	}
	return buf.String(), -1, nil
}
