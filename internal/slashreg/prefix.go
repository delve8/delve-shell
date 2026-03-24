package slashreg

// PrefixEntry is a generic prefix-command entry.
type PrefixEntry[M any, C any] struct {
	Prefix string
	Handle func(M, string) (M, C, bool)
}

// PrefixRegistry stores ordered prefix dispatch entries.
type PrefixRegistry[M any, C any] struct {
	entries []PrefixEntry[M, C]
}

// NewPrefixRegistry creates an empty prefix registry.
func NewPrefixRegistry[M any, C any]() *PrefixRegistry[M, C] {
	return &PrefixRegistry[M, C]{entries: make([]PrefixEntry[M, C], 0)}
}

// Set registers or overwrites a prefix command entry.
func (r *PrefixRegistry[M, C]) Set(prefix string, entry PrefixEntry[M, C]) {
	if r == nil || prefix == "" {
		return
	}
	if entry.Prefix == "" {
		entry.Prefix = prefix
	}
	for i := range r.entries {
		if r.entries[i].Prefix == entry.Prefix {
			r.entries[i] = entry
			return
		}
	}
	r.entries = append(r.entries, entry)
}

// Entries returns all prefix entries in registration order.
func (r *PrefixRegistry[M, C]) Entries() []PrefixEntry[M, C] {
	if r == nil {
		return nil
	}
	return r.entries
}
