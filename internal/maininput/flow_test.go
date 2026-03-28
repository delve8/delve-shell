package maininput

import (
	"testing"

	"delve-shell/internal/slashview"
)

func TestCaptureSlashSelection_FillOnly(t *testing.T) {
	res := CaptureSlashSelection(CaptureInput{
		InputVal:     "/e",
		Text:         "/e",
		SuggestIndex: 0,
		Selected:     slashview.Option{Cmd: "/exec <cmd>", FillValue: "/exec "},
		HasSelected:  true,
	})
	if !res.FillOnly || res.FillValue != "/exec " {
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
