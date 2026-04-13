package slashview

import "strings"

// ShouldFillOnly reports whether selecting chosen should only fill input.
func ShouldFillOnly(chosen Option, input string) bool {
	text := strings.TrimSpace(input)
	textLower := strings.ToLower(text)
	if chosen.FillValue == "" {
		cmd := strings.TrimSpace(chosen.Cmd)
		// Case-fold so "/access l" can complete "/access Local" (reserved tokens are title-cased).
		if strings.EqualFold(cmd, text) {
			return false
		}
		if strings.HasPrefix(strings.ToLower(cmd), textLower) {
			return true
		}
		return descMatchesInput(chosen.Desc, text)
	}
	cmd := strings.ToLower(strings.TrimSpace(chosen.Cmd))
	fill := strings.ToLower(strings.TrimSpace(chosen.FillValue))
	if cmd == textLower || fill == textLower {
		return false
	}
	if strings.HasPrefix(cmd, textLower) || strings.HasPrefix(fill, textLower) {
		return true
	}
	return descMatchesInput(chosen.Desc, text)
}

// ShouldResolveSelected reports whether selected slash option should be resolved.
func ShouldResolveSelected(chosen Option, input string) bool {
	text := strings.TrimSpace(input)
	if len(strings.TrimSpace(strings.TrimPrefix(text, "/"))) == 0 {
		return false
	}
	if chosen.FillValue == "" {
		if strings.HasPrefix(chosen.Cmd, text) {
			return true
		}
		return descMatchesInput(chosen.Desc, text)
	}
	if strings.HasPrefix(chosen.Cmd, text) || strings.HasPrefix(chosen.FillValue, text) {
		return true
	}
	return descMatchesInput(chosen.Desc, text)
}

func descMatchesInput(desc, input string) bool {
	descLower := strings.ToLower(strings.TrimSpace(desc))
	if descLower == "" {
		return false
	}
	inputLower := strings.ToLower(strings.TrimSpace(input))
	if strings.HasPrefix(descLower, inputLower) {
		return true
	}
	rest, ok := parseAccessRest(input)
	return ok && rest != "" && strings.HasPrefix(descLower, strings.ToLower(rest))
}
