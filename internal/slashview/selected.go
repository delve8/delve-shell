package slashview

// SelectedByVisibleIndex returns selected option by visible index.
func SelectedByVisibleIndex(opts []Option, vis []int, selectedIndex int) (Option, bool) {
	if selectedIndex < 0 || selectedIndex >= len(vis) {
		return Option{}, false
	}
	optIndex := vis[selectedIndex]
	if optIndex < 0 || optIndex >= len(opts) {
		return Option{}, false
	}
	return opts[optIndex], true
}
