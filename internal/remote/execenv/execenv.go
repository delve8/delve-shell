package execenv

import (
	"bytes"
	"context"
)

// CommandExecutor executes shell commands and returns separated stdout/stderr and exit code.
// Implementations may run locally or remotely (e.g. via SSH).
type CommandExecutor interface {
	Run(ctx context.Context, command string) (stdout, stderr string, exitCode int, err error)
}

// LocalExecutor runs commands on the local machine via "sh -c".
type LocalExecutor struct{}

func (LocalExecutor) Run(ctx context.Context, command string) (stdout, stderr string, exitCode int, err error) {
	var outBuf, errBuf bytes.Buffer
	exitCode, runErr := localShRun(ctx, command, &outBuf, &errBuf)
	if runErr != nil && exitCode == 0 {
		// Non-zero error without explicit exit code (rare); treat as generic failure with -1.
		exitCode = -1
	}
	return outBuf.String(), errBuf.String(), exitCode, runErr
}
