package ui

// ReadModel provides host-derived read-only state needed by UI rendering and local decisions.
type ReadModel interface {
	AllowlistAutoRunEnabled() bool
	TakeOpenConfigLLMOnFirstLayout() bool
}

type nopReadModel struct{}

func (nopReadModel) AllowlistAutoRunEnabled() bool        { return true }
func (nopReadModel) TakeOpenConfigLLMOnFirstLayout() bool { return false }

func defaultReadModel(r ReadModel) ReadModel {
	if r == nil {
		return nopReadModel{}
	}
	return r
}

func (m Model) allowlistAutoRunEnabled() bool {
	return defaultReadModel(m.ReadModel).AllowlistAutoRunEnabled()
}

func (m Model) takeOpenConfigLLMOnFirstLayout() bool {
	return defaultReadModel(m.ReadModel).TakeOpenConfigLLMOnFirstLayout()
}
