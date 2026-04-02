//go:build !unix && !windows

package execenv

import (
	"context"
	"io"
	"os/exec"
)

func localShRun(ctx context.Context, command string, stdout, stderr io.Writer) (exitCode int, err error) {
	cmd := exec.CommandContext(ctx, "sh", "-c", command)
	cmd.Stdout = stdout
	cmd.Stderr = stderr
	runErr := cmd.Run()
	if exitErr, ok := runErr.(*exec.ExitError); ok {
		exitCode = exitErr.ExitCode()
	}
	if runErr != nil && exitCode == 0 {
		exitCode = -1
	}
	return exitCode, runErr
}
