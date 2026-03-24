package slashreg

// ProviderChain stores providers in registration order and allows iteration.
type ProviderChain[P any] struct {
	providers []P
}

// NewProviderChain creates an empty provider chain.
func NewProviderChain[P any]() *ProviderChain[P] {
	return &ProviderChain[P]{providers: make([]P, 0)}
}

// Add appends a provider when non-nil.
func (c *ProviderChain[P]) Add(provider P, isNil func(P) bool) {
	if c == nil {
		return
	}
	if isNil != nil && isNil(provider) {
		return
	}
	c.providers = append(c.providers, provider)
}

// List returns providers in registration order.
func (c *ProviderChain[P]) List() []P {
	if c == nil {
		return nil
	}
	return c.providers
}
