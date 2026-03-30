package history

import "strings"

// SwitchSessionIDFromSlashLine returns the first token after "/history" from a submitted line.
// Any extra text (e.g. a copied menu description) is ignored so "/history <id> …" still switches to <id>.
func SwitchSessionIDFromSlashLine(trimmed string) (id string, ok bool) {
	trimmed = strings.TrimSpace(trimmed)
	if len(trimmed) < len("/history") || !strings.HasPrefix(trimmed, "/history") {
		return "", false
	}
	tail := trimmed[len("/history"):]
	if tail != "" {
		c0 := tail[0]
		if c0 != ' ' && c0 != '\t' {
			// Not "/history …" as a command (e.g. "/historybook").
			return "", false
		}
	}
	rest := strings.TrimSpace(tail)
	if rest == "" {
		return "", false
	}
	fields := strings.Fields(rest)
	if len(fields) == 0 {
		return "", false
	}
	return fields[0], true
}
