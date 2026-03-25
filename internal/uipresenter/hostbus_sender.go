package uipresenter

import (
	tea "github.com/charmbracelet/bubbletea"

	"delve-shell/internal/hostbus"
)

// BusSender adapts hostbus.Bus to Sender (blocking UI enqueue).
type BusSender struct {
	Bus *hostbus.Bus
}

func (s BusSender) Send(msg tea.Msg) {
	if s.Bus == nil || msg == nil {
		return
	}
	s.Bus.EnqueueUIBlocking(msg)
}
