package hostnotify

import "sync"

// One-shot: open Config LLM overlay on the first WindowSizeMsg of the current UI session (wired from cli/run).
var (
	openConfigLLMFirstMu sync.Mutex
	openConfigLLMFirst   bool
)

// SetOpenConfigLLMOnFirstLayout arms the next first-layout open (typically once per tea.Program from cli).
func SetOpenConfigLLMOnFirstLayout(v bool) {
	openConfigLLMFirstMu.Lock()
	defer openConfigLLMFirstMu.Unlock()
	openConfigLLMFirst = v
}

// TakeOpenConfigLLMOnFirstLayout returns whether to run startup overlay providers and clears the flag.
func TakeOpenConfigLLMOnFirstLayout() bool {
	openConfigLLMFirstMu.Lock()
	defer openConfigLLMFirstMu.Unlock()
	v := openConfigLLMFirst
	openConfigLLMFirst = false
	return v
}
