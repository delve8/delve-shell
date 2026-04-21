package maininput

import (
	"testing"

	"delve-shell/internal/slash/view"
)

func TestCaptureSlashSelection_FillOnly(t *testing.T) {
	res := CaptureSlashSelection(CaptureInput{
		InputVal:     "/sk",
		Text:         "/sk",
		SuggestIndex: 0,
		Selected:     slashview.Option{Cmd: "/skill demo"},
		HasSelected:  true,
	})
	if !res.FillOnly || res.FillValue != "/skill demo " {
		t.Fatalf("unexpected result: %+v", res)
	}
}

func TestSyncSlashSuggestIndex_ResetOnNonSession(t *testing.T) {
	got := SyncSlashSuggestIndex(SyncInput{
		InputVal:            "/he",
		CurrentSuggestIndex: 3,
		VisibleCount:        4,
	})
	if got != 0 {
		t.Fatalf("unexpected index: %d", got)
	}
}
