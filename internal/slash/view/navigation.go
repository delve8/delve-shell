package slashview

import "delve-shell/internal/teakey"

// NextSuggestIndex computes next slash suggestion index for up/down navigation.
func NextSuggestIndex(current int, count int, key string) (int, bool) {
	if count <= 0 {
		return current, false
	}
	if key != teakey.Up && key != teakey.Down {
		return current, false
	}
	if current >= count || current < 0 {
		current = 0
	}
	if key == teakey.Down {
		return (current + 1) % count, true
	}
	return (current - 1 + count) % count, true
}
