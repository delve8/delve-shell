package slashview

import "testing"

func TestVisibleIndices_MatchByPrefix(t *testing.T) {
	opts := []Option{
		{Cmd: "/help"},
		{Cmd: "/exec <cmd>"},
		{Cmd: "/remote on"},
	}
	got := VisibleIndices("/r", opts)
	// /exec no longer shares the "r" prefix with /remote.
	if len(got) != 1 || got[0] != 2 {
		t.Fatalf("unexpected indices: %#v", got)
	}
}

func TestVisibleIndices_NoPrefixMatchWithTypedInputShowsEmpty(t *testing.T) {
	opts := []Option{
		{Cmd: "/help"},
		{Cmd: "/exec <cmd>"},
	}
	got := VisibleIndices("/zzz", opts)
	if len(got) != 0 {
		t.Fatalf("expected no rows when nothing matches typed garbage, got %#v", got)
	}
}

func TestVisibleIndices_EmptyInputAfterSlashShowsAll(t *testing.T) {
	opts := []Option{
		{Cmd: "/help"},
		{Cmd: "/exec <cmd>"},
	}
	got := VisibleIndices("/", opts)
	if len(got) != 2 || got[0] != 0 || got[1] != 1 {
		t.Fatalf("expected all options for bare slash, got %#v", got)
	}
}

func TestVisibleIndices_SessionsByPrefix(t *testing.T) {
	opts := []Option{
		{Cmd: "/session demo"},
		{Cmd: "/session abc"},
	}
	got := VisibleIndices("/session d", opts)
	if len(got) != 1 || got[0] != 0 {
		t.Fatalf("unexpected session indices: %#v", got)
	}
}

func TestChosenToInputValue_StripsPlaceholder(t *testing.T) {
	got := ChosenToInputValue(Option{Cmd: "/exec <cmd>"})
	if got != "/exec " {
		t.Fatalf("unexpected value: %q", got)
	}
}

func TestVisibleIndices_RemoteOnHostPrefix(t *testing.T) {
	opts := []Option{
		{Cmd: "/remote on prod"},
		{Cmd: "/remote on db"},
		{Cmd: "/remote on"},
		{Cmd: "/remote off"},
	}
	got := VisibleIndices("/remote p", opts)
	if len(got) != 1 || got[0] != 0 {
		t.Fatalf("want prod only for /remote p, got %#v", got)
	}
	got = VisibleIndices("/remote on pr", opts)
	if len(got) != 1 || got[0] != 0 {
		t.Fatalf("want prod for /remote on pr, got %#v", got)
	}
	if got2 := VisibleIndices("/remote zzz", opts); len(got2) != 0 {
		t.Fatalf("want no host rows for /remote zzz, got %#v", got2)
	}
}

func TestVisibleIndices_ConfigDelRemoteHostPrefix(t *testing.T) {
	opts := []Option{
		{Cmd: "/config del-remote prod"},
		{Cmd: "/config del-remote db"},
	}
	got := VisibleIndices("/config del-remote p", opts)
	if len(got) != 1 || got[0] != 0 {
		t.Fatalf("want prod only, got %#v", got)
	}
}
