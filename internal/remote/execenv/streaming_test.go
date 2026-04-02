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

func TestLineEmitWriter_carriageReturnProgress(t *testing.T) {
	var lines []string
	w := NewLineEmitWriter(func(l string) { lines = append(lines, l) })
	_, _ = w.Write([]byte("uploading\rwriting manifest: done\n"))
	if len(lines) != 1 || lines[0] != "writing manifest: done" {
		t.Fatalf("got %q", lines)
	}
}

func TestLineEmitWriter_carriageReturnSplitWrites(t *testing.T) {
	var lines []string
	w := NewLineEmitWriter(func(l string) { lines = append(lines, l) })
	_, _ = w.Write([]byte("phase1\r"))
	_, _ = w.Write([]byte("phase2\n"))
	if len(lines) != 1 || lines[0] != "phase2" {
		t.Fatalf("got %q", lines)
	}
}

func TestNormalizeLineForEmit(t *testing.T) {
	tests := []struct {
		in, want string
	}{
		{"hello", "hello"},
		{"hello\r", "hello"},
		{"hello\rworld", "world"},
		{"a\rb\rc", "c"},
		{"", ""},
		{"\r", ""},
	}
	for _, tt := range tests {
		if got := normalizeLineForEmit(tt.in); got != tt.want {
			t.Errorf("normalizeLineForEmit(%q) = %q, want %q", tt.in, got, tt.want)
		}
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
