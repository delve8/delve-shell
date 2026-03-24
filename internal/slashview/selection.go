package slashview

import "strings"

// ShouldFillOnly reports whether selecting chosen should only fill input.
func ShouldFillOnly(chosen string, input string) bool {
	text := strings.TrimSpace(input)
	return strings.HasPrefix(chosen, text) && chosen != text
}

// ShouldResolveSelected reports whether selected slash option should be resolved.
func ShouldResolveSelected(chosen string, input string) bool {
	text := strings.TrimSpace(input)
	return len(strings.TrimSpace(strings.TrimPrefix(text, "/"))) > 0 && strings.HasPrefix(chosen, text)
}
