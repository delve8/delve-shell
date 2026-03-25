package uipresenter

import (
	tea "github.com/charmbracelet/bubbletea"

	"delve-shell/internal/host/bus"
)

// BusSender adapts bus.Bus to Sender (blocking UI enqueue).
type BusSender struct {
	Bus *bus.Bus
}

func (s BusSender) Send(msg tea.Msg) {
	if s.Bus == nil || msg == nil {
		return
	}
	s.Bus.EnqueueUIBlocking(msg)
}
