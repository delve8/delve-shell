package slashview

import "testing"

func TestBuildDropdownRows_HighlightSessionCommand(t *testing.T) {
	opts := []Option{
		{Cmd: "/history demo"},
		{Cmd: "/history prod"},
	}
	vis := []int{0, 1}
	rows := BuildDropdownRows(opts, vis, 1, 100, 4)
	if len(rows) != 2 {
		t.Fatalf("unexpected rows length: %d", len(rows))
	}
	if rows[0].Text != "/history demo" {
		t.Fatalf("unexpected first row: %q", rows[0].Text)
	}
	if !rows[1].Highlight {
		t.Fatalf("expected second row to be highlighted")
	}
}

func TestBuildDropdownRows_TruncatesDescriptionToSingleLine(t *testing.T) {
	opts := []Option{
		{Cmd: "/exec <cmd>", Desc: "this is a very long description for wrap test"},
	}
	vis := []int{0}
	rows := BuildDropdownRows(opts, vis, 0, 28, 4)
	if len(rows) != 1 {
		t.Fatalf("expected one row, got %d", len(rows))
	}
	if !rows[0].Highlight {
		t.Fatalf("expected first row highlight")
	}
	if rows[0].Text == "" {
		t.Fatal("expected non-empty row text")
	}
	if rows[0].Text[len(rows[0].Text)-3:] != "..." {
		t.Fatalf("expected truncated row to end with ..., got %q", rows[0].Text)
	}
}

func TestBuildDropdownRows_OneRowPerVisibleOption(t *testing.T) {
	opts := []Option{
		{Cmd: "/skill alpha", Desc: "first long description that used to wrap"},
		{Cmd: "/skill beta", Desc: "second long description that used to wrap"},
		{Cmd: "/skill gamma", Desc: "third long description that used to wrap"},
	}
	vis := []int{0, 1, 2}
	rows := BuildDropdownRows(opts, vis, 1, 30, 3)
	if len(rows) != 3 {
		t.Fatalf("expected one row per option, got %d", len(rows))
	}
	if !rows[1].Highlight {
		t.Fatalf("expected highlighted selected row")
	}
}
