package tools

import (
	"bytes"
	"context"
	"io"

	hiltypes "delve-shell/internal/hil/types"
	"delve-shell/internal/remote/execenv"
)

// runExecutorWithStream runs command through executor. When onStream is non-nil and executor implements
// [execenv.StreamingRunner], emits streamStart (with Command set to command) then [hiltypes.ExecStreamLine]
// events and uses RunStreaming; otherwise uses Run without calling onStream.
func runExecutorWithStream(ctx context.Context, executor execenv.CommandExecutor, command string, onStream func(any), streamStart hiltypes.ExecStreamStart) (outStr, errStr string, exitCode int, runErr error, streamed bool) {
	streamStart.Command = command
	if onStream != nil {
		if sr, ok := executor.(execenv.StreamingRunner); ok {
			onStream(streamStart)
			var outBuf, errBuf bytes.Buffer
			lineOut := execenv.NewLineEmitWriter(func(line string) {
				onStream(hiltypes.ExecStreamLine{Line: line, Stderr: false})
			})
			lineErr := execenv.NewLineEmitWriter(func(line string) {
				onStream(hiltypes.ExecStreamLine{Line: line, Stderr: true})
			})
			mwOut := io.MultiWriter(&outBuf, lineOut)
			mwErr := io.MultiWriter(&errBuf, lineErr)
			exitCode, runErr = sr.RunStreaming(ctx, command, mwOut, mwErr)
			lineOut.Flush()
			lineErr.Flush()
			return outBuf.String(), errBuf.String(), exitCode, runErr, true
		}
	}
	outStr, errStr, exitCode, runErr = executor.Run(ctx, command)
	return outStr, errStr, exitCode, runErr, false
}
