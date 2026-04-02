package execenv

import (
	"bytes"
	"context"
	"strings"
	"testing"
)

func TestLineEmitWriter(t *testing.T) {
	var lines []string
	w := NewLineEmitWriter(func(l string) { lines = append(lines, l) })
	_, _ = w.Write([]byte("a\nb"))
	_, _ = w.Write([]byte("c"))
	w.Flush()
	if len(lines) != 2 || lines[0] != "a" || lines[1] != "bc" {
		t.Fatalf("got %q", lines)
	}
}

func TestLocalExecutor_RunStreaming(t *testing.T) {
	ctx := context.Background()
	var outb, errb bytes.Buffer
	var x LocalExecutor
	code, err := x.RunStreaming(ctx, `printf 'out' && echo err >&2`, &outb, &errb)
	if err != nil || code != 0 {
		t.Fatalf("code=%d err=%v", code, err)
	}
	if got := outb.String(); got != "out" {
		t.Fatalf("stdout: %q", got)
	}
	if got := strings.TrimSpace(errb.String()); got != "err" {
		t.Fatalf("stderr: %q", errb.String())
	}
}
