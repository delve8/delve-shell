//go:build unix

package execenv

import (
	"context"
	"io"
	"os/exec"
	"syscall"
)

// localShRun runs "sh -c" with ctx cancellation. The shell runs in its own process group so Esc/cancel
// can signal the whole tree: [exec.CommandContext] only kills the direct child (sh), while pipelines
// or subshells often live in the same group as that sh when Setpgid is set on the shell.
func localShRun(ctx context.Context, command string, stdout, stderr io.Writer) (exitCode int, err error) {
	cmd := exec.Command("sh", "-c", command)
	cmd.Stdout = stdout
	cmd.Stderr = stderr
	cmd.SysProcAttr = &syscall.SysProcAttr{Setpgid: true}
	if err := cmd.Start(); err != nil {
		return -1, err
	}
	done := make(chan error, 1)
	go func() { done <- cmd.Wait() }()
	select {
	case <-ctx.Done():
		killShellProcessGroup(cmd)
		<-done
		return -1, ctx.Err()
	case err := <-done:
		if err == nil {
			return 0, nil
		}
		if exitErr, ok := err.(*exec.ExitError); ok {
			return exitErr.ExitCode(), err
		}
		return -1, err
	}
}

func killShellProcessGroup(cmd *exec.Cmd) {
	if cmd.Process == nil {
		return
	}
	pid := cmd.Process.Pid
	if pid <= 0 {
		return
	}
	// Negative PID: POSIX process group led by the shell (Setpgid above).
	_ = syscall.Kill(-pid, syscall.SIGTERM)
	_ = syscall.Kill(-pid, syscall.SIGKILL)
}
