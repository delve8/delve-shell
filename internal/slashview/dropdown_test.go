package slashview

import "testing"

func TestBuildDropdownRows_HighlightSessionCommand(t *testing.T) {
	opts := []Option{
		{Cmd: "/session demo"},
		{Cmd: "/session prod"},
	}
	vis := []int{0, 1}
	rows := BuildDropdownRows(opts, vis, 1, 100, 4)
	if len(rows) != 2 {
		t.Fatalf("unexpected rows length: %d", len(rows))
	}
	if rows[0].Text != "/session demo" {
		t.Fatalf("unexpected first row: %q", rows[0].Text)
	}
	if !rows[1].Highlight {
		t.Fatalf("expected second row to be highlighted")
	}
}

func TestBuildDropdownRows_WrapDescription(t *testing.T) {
	opts := []Option{
		{Cmd: "/exec <cmd>", Desc: "this is a very long description for wrap test"},
	}
	vis := []int{0}
	rows := BuildDropdownRows(opts, vis, 0, 28, 4)
	if len(rows) < 2 {
		t.Fatalf("expected wrapped rows, got %d", len(rows))
	}
	if !rows[0].Highlight {
		t.Fatalf("expected first row highlight")
	}
}
