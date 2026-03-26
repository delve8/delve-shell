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

func TestVisibleIndices_SessionsByPrefix(t *testing.T) {
	opts := []Option{
		{Cmd: "/sessions demo"},
		{Cmd: "/sessions abc"},
	}
	got := VisibleIndices("/sessions d", opts)
	if len(got) != 1 || got[0] != 0 {
		t.Fatalf("unexpected session indices: %#v", got)
	}
}

func TestChosenToInputValue_StripsPlaceholder(t *testing.T) {
	got := ChosenToInputValue(Option{Cmd: "/run <cmd>"})
	if got != "/run " {
		t.Fatalf("unexpected value: %q", got)
	}
}
