package slashreg

// ExactEntry is a generic exact-command entry.
type ExactEntry[M any, C any] struct {
	Handle     func(M) (M, C)
	ClearInput bool
}

// ExactRegistry stores exact command dispatch entries.
type ExactRegistry[M any, C any] struct {
	entries map[string]ExactEntry[M, C]
}

// NewExactRegistry creates an empty exact registry.
func NewExactRegistry[M any, C any]() *ExactRegistry[M, C] {
	return &ExactRegistry[M, C]{entries: make(map[string]ExactEntry[M, C])}
}

// Set registers or overwrites an exact command entry.
func (r *ExactRegistry[M, C]) Set(cmd string, entry ExactEntry[M, C]) {
	if r == nil || cmd == "" {
		return
	}
	r.entries[cmd] = entry
}

// Get returns a registered exact command entry.
func (r *ExactRegistry[M, C]) Get(cmd string) (ExactEntry[M, C], bool) {
	if r == nil || cmd == "" {
		var zero ExactEntry[M, C]
		return zero, false
	}
	entry, ok := r.entries[cmd]
	return entry, ok
}
