package interactive

import (
	"os"
	"os/exec"
	"sync/atomic"

	tea "github.com/charmbracelet/bubbletea"

	"delve-shell/internal/hostcontroller"
	"delve-shell/internal/hostapp"
	"delve-shell/internal/ui"
)

// defaultTUIProgramOptions are passed to every tea.NewProgram for the main interactive session.
var defaultTUIProgramOptions = []tea.ProgramOption{
	tea.WithAltScreen(),
	tea.WithReportFocus(),
}

// tuiRestartLoop runs one or more Bubble Tea programs in sequence: when the user exits the TUI
// and the shell bridge has delivered saved transcript lines (e.g. /sh), an interactive bash
// is started on stdio; when that returns, the TUI starts again with those messages restored.
type tuiRestartLoop struct {
	controller *hostcontroller.Controller
	// programPtr is shared with the host controller and UI pump so outbound tea.Msg reach the active program.
	programPtr *atomic.Pointer[tea.Program]
	// shellAfterExit receives at most one buffered snapshot per TUI exit when subshell was requested.
	shellAfterExit <-chan []string
	// openConfigLLMOnFirstLayout is applied only for the first TUI session in this process (startup overlay).
	openConfigLLMOnFirstLayout bool
}

func newTuiRestartLoop(
	controller *hostcontroller.Controller,
	programPtr *atomic.Pointer[tea.Program],
	shellAfterExit <-chan []string,
	openConfigLLMOnFirstLayout bool,
) *tuiRestartLoop {
	return &tuiRestartLoop{
		controller:                 controller,
		programPtr:                 programPtr,
		shellAfterExit:             shellAfterExit,
		openConfigLLMOnFirstLayout: openConfigLLMOnFirstLayout,
	}
}

// run blocks until the user leaves the TUI without requesting the embedded subshell, or until
// tea.Program.Run returns an error.
func (l *tuiRestartLoop) run() error {
	var saved []string
	openLLM := l.openConfigLLMOnFirstLayout
	for {
		if err := l.runOneSession(&saved, openLLM); err != nil {
			return err
		}
		openLLM = false
		select {
		case saved = <-l.shellAfterExit:
			runEmbeddedSubshellIgnoringExitCode()
		default:
			return nil
		}
	}
}

func (l *tuiRestartLoop) runOneSession(saved *[]string, openConfigLLM bool) error {
	l.controller.SyncCurrentSessionPath()
	hostapp.SetOpenConfigLLMOnFirstLayout(openConfigLLM)
	model := ui.NewModel(*saved)
	p := tea.NewProgram(model, defaultTUIProgramOptions...)
	l.programPtr.Store(p)
	_, err := p.Run()
	l.programPtr.Store(nil)
	return err
}

// runEmbeddedSubshellIgnoringExitCode runs bash -i on the process stdio; bash exit status is ignored
// to match historical behavior (subshell is a convenience escape, not a scripted step).
func runEmbeddedSubshellIgnoringExitCode() {
	sh := exec.Command("bash", "-i")
	sh.Stdin = os.Stdin
	sh.Stdout = os.Stdout
	sh.Stderr = os.Stderr
	_ = sh.Run()
}
