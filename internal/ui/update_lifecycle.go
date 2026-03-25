package ui

import (
	tea "github.com/charmbracelet/bubbletea"
)

func (m Model) handleWindowSizeMsg(msg tea.WindowSizeMsg) (Model, tea.Cmd) {
	m.layout.Width = msg.Width
	m.layout.Height = msg.Height
	// Use full terminal width for input so long lines don't scroll until they exceed the line.
	if m.layout.Width > minInputLayoutWidth {
		m.Input.Width = m.layout.Width - minInputLayoutWidth // leave margin for prompt "> " and right edge
	}
	if m.layout.Height > minInputLayoutWidth {
		vh := m.mainViewportHeight() // header + sep + viewport; bottom 2 lines for input + slash/choice dropdown
		m.Viewport.Width = m.layout.Width
		m.Viewport.Height = vh
	}
	m = m.RefreshViewport()
	if m.Host.TakeOpenConfigLLMOnFirstLayout() {
		for _, p := range startupOverlayProviderChain.List() {
			if m2, cmd, handled := p(m); handled {
				return m2, cmd
			}
		}
	}
	return m, nil
}

func (m Model) handleBlurMsg() (Model, tea.Cmd) {
	// Window lost focus: blur main input so its cursor stops blinking.
	m.Input.Blur()
	return m, nil
}

func (m Model) handleFocusMsg() (Model, tea.Cmd) {
	// Window gained focus: restore main input focus only when not in an overlay.
	if !m.Overlay.Active {
		return m, m.Input.Focus()
	}
	return m, nil
}

func (m Model) handleMouseMsg(msg tea.MouseMsg) (Model, tea.Cmd) {
	var cmd tea.Cmd
	m.Viewport, cmd = m.Viewport.Update(msg)
	return m, cmd
}
