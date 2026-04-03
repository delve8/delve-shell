package slashview

import "strings"

// ShouldFillOnly reports whether selecting chosen should only fill input.
func ShouldFillOnly(chosen Option, input string) bool {
	text := strings.TrimSpace(input)
	if chosen.FillValue == "" {
		cmd := strings.TrimSpace(chosen.Cmd)
		// Case-fold so "/access l" can complete "/access Local" (reserved tokens are title-cased).
		if strings.EqualFold(cmd, text) {
			return false
		}
		return strings.HasPrefix(strings.ToLower(cmd), strings.ToLower(text))
	}
	return strings.HasPrefix(chosen.Cmd, text)
}

// ShouldResolveSelected reports whether selected slash option should be resolved.
func ShouldResolveSelected(chosen Option, input string) bool {
	text := strings.TrimSpace(input)
	return len(strings.TrimSpace(strings.TrimPrefix(text, "/"))) > 0 && strings.HasPrefix(chosen.Cmd, text)
}
