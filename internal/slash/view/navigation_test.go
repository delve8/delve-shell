package slashview

import (
	"testing"

	"delve-shell/internal/teakey"
)

func TestNextSuggestIndex_Down(t *testing.T) {
	got, changed := NextSuggestIndex(0, 3, teakey.Down)
	if !changed || got != 1 {
		t.Fatalf("unexpected result: got=%d changed=%v", got, changed)
	}
}

func TestNextSuggestIndex_UpWrap(t *testing.T) {
	got, changed := NextSuggestIndex(0, 3, teakey.Up)
	if !changed || got != 2 {
		t.Fatalf("unexpected result: got=%d changed=%v", got, changed)
	}
}

func TestNextSuggestIndex_ResetOutOfRange(t *testing.T) {
	got, changed := NextSuggestIndex(99, 2, teakey.Down)
	if !changed || got != 1 {
		t.Fatalf("unexpected result: got=%d changed=%v", got, changed)
	}
}
