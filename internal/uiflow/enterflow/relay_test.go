package enterflow

import (
	"testing"

	"delve-shell/internal/uitypes"
)

func TestTryRelayMainEnter_nonSlashNoop(t *testing.T) {
	t.Helper()
	if TryRelayMainEnter("hello", 0, func(uitypes.SlashSubmitPayload) bool { t.Fatal("relay called"); return true }) {
		t.Fatal("unexpected relay")
	}
}

func TestTryRelayMainEnter_callsRelay(t *testing.T) {
	t.Helper()
	var saw bool
	ok := TryRelayMainEnter("/x", 3, func(p uitypes.SlashSubmitPayload) bool {
		saw = true
		return p.RawLine == "/x" && p.SlashSelectedIndex == 3
	})
	if !ok || !saw {
		t.Fatalf("relay not applied ok=%v saw=%v", ok, saw)
	}
}
