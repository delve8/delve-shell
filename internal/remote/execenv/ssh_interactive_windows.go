//go:build windows

package execenv

import (
	"context"
	"errors"
)

// RunInteractiveSSHShell is not supported on Windows in this build.
func RunInteractiveSSHShell(ctx context.Context, exec CommandExecutor) error {
	_, ok := exec.(*SSHExecutor)
	if !ok {
		return errors.New("remote subshell requires an active SSH connection")
	}
	return errors.New("remote interactive shell is not supported on Windows")
}
