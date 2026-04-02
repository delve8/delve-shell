package execenv

import (
	"bytes"
	"context"
	"io"
	"strings"
)

// StreamingRunner runs a command while copying stdout/stderr to writers (for live TUI). Implementations may omit this and fall back to [CommandExecutor.Run].
type StreamingRunner interface {
	RunStreaming(ctx context.Context, command string, stdout, stderr io.Writer) (exitCode int, err error)
}

// LineEmitWriter buffers output and calls emit once per newline-terminated line. Call [LineEmitWriter.Flush] after the process exits to emit a final partial line.
type LineEmitWriter struct {
	buf  bytes.Buffer
	emit func(line string)
}

// NewLineEmitWriter returns a writer that splits on '\n' and invokes emit for each line (without the newline). Carriage returns before '\n' are trimmed from line text.
func NewLineEmitWriter(emit func(line string)) *LineEmitWriter {
	if emit == nil {
		emit = func(string) {}
	}
	return &LineEmitWriter{emit: emit}
}

func (w *LineEmitWriter) Write(p []byte) (int, error) {
	w.buf.Write(p)
	for {
		b := w.buf.Bytes()
		i := bytes.IndexByte(b, '\n')
		if i < 0 {
			break
		}
		line := strings.TrimSuffix(string(b[:i]), "\r")
		w.buf.Next(i + 1)
		w.emit(line)
	}
	return len(p), nil
}

// Flush emits any bytes after the last newline as one line.
func (w *LineEmitWriter) Flush() {
	if w.buf.Len() == 0 {
		return
	}
	line := strings.TrimSuffix(w.buf.String(), "\r")
	w.buf.Reset()
	w.emit(line)
}

// RunStreaming runs "sh -c" with stdout/stderr wired to the given writers (same semantics as [LocalExecutor.Run] for exit codes).
func (LocalExecutor) RunStreaming(ctx context.Context, command string, stdout, stderr io.Writer) (exitCode int, err error) {
	exitCode, runErr := localShRun(ctx, command, stdout, stderr)
	if runErr != nil && exitCode == 0 {
		exitCode = -1
	}
	return exitCode, runErr
}
