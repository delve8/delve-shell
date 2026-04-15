package slashview

import "strings"

// ShouldFillOnly reports whether selecting chosen should only fill input.
func ShouldFillOnly(chosen Option, input string) bool {
	text := strings.TrimSpace(input)
	textLower := strings.ToLower(text)
	fillValue := strings.TrimSpace(chosen.FillValue)
	executeValue := strings.TrimSpace(chosen.ExecuteValue)
	if fillValue == "" && executeValue == "" {
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
	values := []string{
		strings.ToLower(strings.TrimSpace(chosen.Cmd)),
		strings.ToLower(fillValue),
		strings.ToLower(executeValue),
	}
	for _, value := range values {
		if value != "" && value == textLower {
			return false
		}
	}
	for _, value := range values {
		if value != "" && strings.HasPrefix(value, textLower) {
			return true
		}
	}
	return descMatchesInput(chosen.Desc, text)
}

// ShouldResolveSelected reports whether selected slash option should be resolved.
func ShouldResolveSelected(chosen Option, input string) bool {
	text := strings.TrimSpace(input)
	if len(strings.TrimSpace(strings.TrimPrefix(text, "/"))) == 0 {
		return false
	}
	if strings.TrimSpace(chosen.FillValue) == "" && strings.TrimSpace(chosen.ExecuteValue) == "" {
		if strings.HasPrefix(chosen.Cmd, text) {
			return true
		}
		return descMatchesInput(chosen.Desc, text)
	}
	for _, value := range []string{chosen.Cmd, chosen.FillValue, chosen.ExecuteValue} {
		if value != "" && strings.HasPrefix(value, text) {
			return true
		}
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
