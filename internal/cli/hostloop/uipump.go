package hostloop

import (
	"sync/atomic"

	tea "github.com/charmbracelet/bubbletea"
)

// RunUIPump delivers messages to the active tea.Program from one goroutine.
func RunUIPump(stop <-chan struct{}, uiMsgChan <-chan tea.Msg, currentP *atomic.Pointer[tea.Program]) {
	for {
		select {
		case <-stop:
			return
		case m := <-uiMsgChan:
			if m == nil {
				continue
			}
			if p := currentP.Load(); p != nil {
				p.Send(m)
			}
		}
	}
}
