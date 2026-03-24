package ui

import tea "github.com/charmbracelet/bubbletea"

func (m Model) handleWindowSizeMsg(msg tea.WindowSizeMsg) (Model, tea.Cmd) {
	m.Layout.Width = msg.Width
	m.Layout.Height = msg.Height
	// Use full terminal width for input so long lines don't scroll until they exceed the line.
	if m.Layout.Width > 4 {
		m.Input.Width = m.Layout.Width - 4 // leave margin for prompt "> " and right edge
	}
	if m.Layout.Height > 4 {
		vh := m.Layout.Height - 10 // header + sep + viewport; bottom 2 lines for input + slash/choice dropdown
		if vh < 1 {
			vh = 1
		}
		m.Viewport.Width = m.Layout.Width
		m.Viewport.Height = vh
	}
	m.Viewport.SetContent(m.buildContent())
	m.Viewport.GotoBottom()
	if m.Startup.InitialShowConfigLLM {
		m.Startup.InitialShowConfigLLM = false
		if m2, cmd, handled := m.dispatchSlashExact("/config llm"); handled {
			return m2, cmd
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
