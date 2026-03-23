package ui

// SuggestStyleRender renders text using suggestion style.
func SuggestStyleRender(s string) string {
	return suggestStyle.Render(s)
}

// SuggestHiRender renders text using highlighted suggestion style.
func SuggestHiRender(s string) string {
	return suggestHi.Render(s)
}

// ErrStyleRender renders text using error style.
func ErrStyleRender(s string) string {
	return errStyle.Render(s)
}
