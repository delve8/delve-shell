package hostbus

// PublishHook is invoked after a publish attempt. accepted is false when the event queue was full (Publish only).
type PublishHook func(e Event, accepted bool)

// BusOption configures a Bus at construction.
type BusOption func(*Bus)

// WithPublishHook registers a hook for Publish and PublishBlocking outcomes.
func WithPublishHook(h PublishHook) BusOption {
	return func(b *Bus) {
		b.publishHook = h
	}
}
