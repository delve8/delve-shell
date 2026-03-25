package ui

import "delve-shell/internal/slashview"

// slashSuggestionContext returns options, visible indices, and slashview rows for the current input buffer
// using Model.getLang. Use slashSuggestionContextWithLang when the view passes an explicit lang.
func (m Model) slashSuggestionContext(inputVal string) (opts []SlashOption, vis []int, viewOpts []slashview.Option) {
	return m.slashSuggestionContextWithLang(inputVal, m.getLang())
}

// slashSuggestionContextWithLang is the same as slashSuggestionContext but uses an explicit UI language
// (e.g. slash dropdown in view_slash_dropdown.go).
func (m Model) slashSuggestionContextWithLang(inputVal, lang string) (opts []SlashOption, vis []int, viewOpts []slashview.Option) {
	opts = getSlashOptionsForInput(inputVal, lang, m.RunCompletion.RemoteRunCommands, m.Host.RemoteActive())
	vis = visibleSlashOptions(inputVal, opts)
	viewOpts = toSlashViewOptions(opts)
	return opts, vis, viewOpts
}
