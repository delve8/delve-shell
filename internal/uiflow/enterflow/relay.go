package enterflow

import (
	"strings"

	"delve-shell/internal/host/route"
)

// TryRelayMainEnter asks the host to relay structured slash intent for the main Enter path
// (payload without InputLine). Returns whether the relay was accepted.
func TryRelayMainEnter(text string, slashSelectedIndex int, relay func(route.SlashSubmitPayload) bool) bool {
	if !strings.HasPrefix(text, "/") {
		return false
	}
	return relay(route.SlashSubmitPayload{
		RawLine:            text,
		SlashSelectedIndex: slashSelectedIndex,
	})
}

// TryRelaySlashInputLine is used on slash-mode Enter when the raw input buffer must be preserved
// (InputLine) for local replay after SlashSubmitRelayMsg.
func TryRelaySlashInputLine(rawLine, inputLine string, slashSelectedIndex int, relay func(route.SlashSubmitPayload) bool) bool {
	if strings.TrimSpace(inputLine) == "" || !strings.HasPrefix(strings.TrimSpace(inputLine), "/") {
		return false
	}
	return relay(route.SlashSubmitPayload{
		RawLine:            strings.TrimSpace(rawLine),
		SlashSelectedIndex: slashSelectedIndex,
		InputLine:          inputLine,
	})
}
