package slashview

import "testing"

func TestVisibleIndices_MatchByPrefix(t *testing.T) {
	opts := []Option{
		{Cmd: "/help"},
		{Cmd: "/skill demo"},
		{Cmd: "/access"},
	}
	got := VisibleIndices("/a", opts)
	if len(got) != 1 || got[0] != 2 {
		t.Fatalf("unexpected indices: %#v", got)
	}
}

func TestVisibleIndices_NoPrefixMatchWithTypedInputShowsEmpty(t *testing.T) {
	opts := []Option{
		{Cmd: "/help"},
		{Cmd: "/skill demo"},
	}
	got := VisibleIndices("/zzz", opts)
	if len(got) != 0 {
		t.Fatalf("expected no rows when nothing matches typed garbage, got %#v", got)
	}
}

func TestVisibleIndices_EmptyInputAfterSlashShowsAll(t *testing.T) {
	opts := []Option{
		{Cmd: "/help"},
		{Cmd: "/skill demo"},
	}
	got := VisibleIndices("/", opts)
	if len(got) != 2 || got[0] != 0 || got[1] != 1 {
		t.Fatalf("expected all options for bare slash, got %#v", got)
	}
}

func TestVisibleIndices_SessionsByPrefix(t *testing.T) {
	opts := []Option{
		{Cmd: "/history demo"},
		{Cmd: "/history abc"},
	}
	got := VisibleIndices("/history d", opts)
	if len(got) != 1 || got[0] != 0 {
		t.Fatalf("unexpected session indices: %#v", got)
	}
}

func TestChosenToInputValue_StripsPlaceholder(t *testing.T) {
	got := ChosenToInputValue(Option{Cmd: "/skill {name} [text]"})
	if got != "/skill " {
		t.Fatalf("unexpected value: %q", got)
	}
}

func TestChosenToInputValue_StripsBracePlaceholder(t *testing.T) {
	got := ChosenToInputValue(Option{Cmd: "/skill {name} [...]"})
	if got != "/skill " {
		t.Fatalf("unexpected value: %q", got)
	}
}

func TestChosenToInputValue_trailingSpaceOnPlainCmd(t *testing.T) {
	got := ChosenToInputValue(Option{Cmd: "/config"})
	if got != "/config " {
		t.Fatalf("unexpected value: %q", got)
	}
}

func TestVisibleIndices_AccessHostPrefix(t *testing.T) {
	opts := []Option{
		{Cmd: "/access prod"},
		{Cmd: "/access db"},
		{Cmd: "/access Local"},
		{Cmd: "/access New"},
	}
	got := VisibleIndices("/access p", opts)
	if len(got) != 1 || got[0] != 0 {
		t.Fatalf("want prod only for /access p, got %#v", got)
	}
	got = VisibleIndices("/access pr", opts)
	if len(got) != 1 || got[0] != 0 {
		t.Fatalf("want prod for /access pr, got %#v", got)
	}
	if got2 := VisibleIndices("/access zzz", opts); len(got2) != 0 {
		t.Fatalf("want no host rows for /access zzz, got %#v", got2)
	}
}

func TestVisibleIndices_AccessFillValuePrefix(t *testing.T) {
	opts := []Option{
		{Cmd: "/access jump.example.com", FillValue: "/access bastion"},
		{Cmd: "/access db.example.com"},
	}
	got := VisibleIndices("/access bast", opts)
	if len(got) != 1 || got[0] != 0 {
		t.Fatalf("want ssh config alias row for /access bast, got %#v", got)
	}
	got = VisibleIndices("/access jump", opts)
	if len(got) != 1 || got[0] != 0 {
		t.Fatalf("want ssh config hostname row for /access jump, got %#v", got)
	}
}

func TestVisibleIndices_AccessExecuteValuePrefix(t *testing.T) {
	opts := []Option{
		{Cmd: "/access jump.example.com", FillValue: "/access jump.example.com", ExecuteValue: "/access bastion"},
		{Cmd: "/access db.example.com"},
	}
	got := VisibleIndices("/access bast", opts)
	if len(got) != 1 || got[0] != 0 {
		t.Fatalf("want ssh config alias row for /access bast, got %#v", got)
	}
	got = VisibleIndices("/access jump", opts)
	if len(got) != 1 || got[0] != 0 {
		t.Fatalf("want ssh config hostname row for /access jump, got %#v", got)
	}
}

func TestVisibleIndices_AccessDescPrefix(t *testing.T) {
	opts := []Option{
		{Cmd: "/access prod", Desc: "Production"},
		{Cmd: "/access db", Desc: "DB Bastion"},
	}
	got := VisibleIndices("/access pro", opts)
	if len(got) != 1 || got[0] != 0 {
		t.Fatalf("want saved remote name row for /access pro, got %#v", got)
	}
	got = VisibleIndices("/access db b", opts)
	if len(got) != 1 || got[0] != 1 {
		t.Fatalf("want saved remote spaced name row for /access db b, got %#v", got)
	}
}

func TestVisibleIndices_AccessLocalReservedVsHost(t *testing.T) {
	opts := []Option{
		{Cmd: "/access Local"},
		{Cmd: "/access local"},
	}
	got := VisibleIndices("/access Local", opts)
	if len(got) != 1 || got[0] != 0 {
		t.Fatalf("Title Local: want reserved row only, got %#v", got)
	}
	got = VisibleIndices("/access local", opts)
	if len(got) != 2 {
		t.Fatalf("lowercase local: want reserved + host row, got %#v", got)
	}
}

func TestVisibleIndices_AccessNewReservedVsHost(t *testing.T) {
	opts := []Option{
		{Cmd: "/access New"},
		{Cmd: "/access new"},
	}
	got := VisibleIndices("/access New", opts)
	if len(got) != 1 || got[0] != 0 {
		t.Fatalf("Title New: want reserved row only, got %#v", got)
	}
	got = VisibleIndices("/access new", opts)
	if len(got) != 2 {
		t.Fatalf("lowercase new: want reserved + host row, got %#v", got)
	}
}

func TestVisibleIndices_SkillNewReservedVsInstalledSkill(t *testing.T) {
	opts := []Option{
		{Cmd: "/skill New"},
		{Cmd: "/skill new", FillValue: "/skill new"},
	}
	got := VisibleIndices("/skill New", opts)
	if len(got) != 1 || got[0] != 0 {
		t.Fatalf("Title New: want reserved row only, got %#v", got)
	}
	got = VisibleIndices("/skill new", opts)
	if len(got) != 2 {
		t.Fatalf("lowercase new: want reserved + skill row, got %#v", got)
	}
}

func TestVisibleIndices_SkillReservedVsInstalledSkill(t *testing.T) {
	opts := []Option{
		{Cmd: "/skill Remove"},
		{Cmd: "/skill remove", FillValue: "/skill remove"},
		{Cmd: "/skill Update"},
		{Cmd: "/skill update", FillValue: "/skill update"},
	}
	got := VisibleIndices("/skill Remove", opts)
	if len(got) != 1 || got[0] != 0 {
		t.Fatalf("Title Remove: want reserved row only, got %#v", got)
	}
	got = VisibleIndices("/skill remove", opts)
	if len(got) != 2 {
		t.Fatalf("lowercase remove: want reserved + skill row, got %#v", got)
	}
	got = VisibleIndices("/skill Update", opts)
	if len(got) != 1 || got[0] != 2 {
		t.Fatalf("Title Update: want reserved row only, got %#v", got)
	}
	got = VisibleIndices("/skill update", opts)
	if len(got) != 2 {
		t.Fatalf("lowercase update: want reserved + skill row, got %#v", got)
	}
}

func TestVisibleIndices_ConfigRemoveRemoteHostPrefix(t *testing.T) {
	opts := []Option{
		{Cmd: "/config remove-remote prod"},
		{Cmd: "/config remove-remote db"},
	}
	got := VisibleIndices("/config remove-remote p", opts)
	if len(got) != 1 || got[0] != 0 {
		t.Fatalf("want prod only, got %#v", got)
	}
}
