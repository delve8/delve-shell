package slashview

import "testing"

func TestVisibleIndices_MatchByPrefix(t *testing.T) {
	opts := []Option{
		{Cmd: "/help"},
		{Cmd: "/run <cmd>"},
		{Cmd: "/remote on"},
	}
	got := VisibleIndices("/r", opts)
	if len(got) != 2 || got[0] != 1 || got[1] != 2 {
		t.Fatalf("unexpected indices: %#v", got)
	}
}

func TestVisibleIndices_FallbackToAllWhenNoMatch(t *testing.T) {
	opts := []Option{
		{Cmd: "/help"},
		{Cmd: "/run <cmd>"},
	}
	got := VisibleIndices("/zzz", opts)
	if len(got) != 2 || got[0] != 0 || got[1] != 1 {
		t.Fatalf("unexpected fallback indices: %#v", got)
	}
}

func TestVisibleIndices_SessionOptionsReturnAll(t *testing.T) {
	opts := []Option{
		{Cmd: "demo", Path: "/tmp/demo.jsonl"},
		{Cmd: "abc", Path: "/tmp/abc.jsonl"},
	}
	got := VisibleIndices("/sessions d", opts)
	if len(got) != 2 || got[0] != 0 || got[1] != 1 {
		t.Fatalf("unexpected session indices: %#v", got)
	}
}

func TestChosenToInputValue_StripsPlaceholder(t *testing.T) {
	got := ChosenToInputValue("/run <cmd>")
	if got != "/run " {
		t.Fatalf("unexpected value: %q", got)
	}
}
