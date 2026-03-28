package interactive

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"sync/atomic"

	tea "github.com/charmbracelet/bubbletea"

	"delve-shell/internal/execenv"
	"delve-shell/internal/host/app"
	"delve-shell/internal/host/controller"
	"delve-shell/internal/hostcmd"
	"delve-shell/internal/ui"
)

// defaultTUIProgramOptions are passed to every tea.NewProgram for the main interactive session.
var defaultTUIProgramOptions = []tea.ProgramOption{
	tea.WithReportFocus(),
}

// tuiRestartLoop runs one or more Bubble Tea programs in sequence: when the user exits the TUI
// and the shell bridge has delivered saved transcript lines (e.g. /sh), either a local bash
// is started on stdio or (when Remote is active) an interactive shell runs over the existing
// SSH connection; when that returns, the TUI starts again with those messages restored.
type tuiRestartLoop struct {
	controller *controller.Controller
	host       app.Host
	// programPtr is shared with the host controller and UI pump so outbound tea.Msg reach the active program.
	programPtr *atomic.Pointer[tea.Program]
	// shellAfterExit receives at most one buffered snapshot per TUI exit when subshell was requested.
	shellAfterExit <-chan hostcmd.ShellSnapshot
	commands       chan<- hostcmd.Command
	getExec        func() execenv.CommandExecutor
	// openConfigLLMOnFirstLayout is applied only for the first TUI session in this process (startup overlay).
	openConfigLLMOnFirstLayout bool
}

type hostReadModel struct {
	host app.Host
}

func (r hostReadModel) AllowlistAutoRunEnabled() bool {
	if r.host == nil {
		return true
	}
	return r.host.AllowlistAutoRunEnabled()
}

func (r hostReadModel) TakeOpenConfigLLMOnFirstLayout() bool {
	if r.host == nil {
		return false
	}
	return r.host.TakeOpenConfigLLMOnFirstLayout()
}

func newTuiRestartLoop(
	controller *controller.Controller,
	programPtr *atomic.Pointer[tea.Program],
	shellAfterExit <-chan hostcmd.ShellSnapshot,
	commands chan<- hostcmd.Command,
	openConfigLLMOnFirstLayout bool,
	host app.Host,
	getExec func() execenv.CommandExecutor,
) *tuiRestartLoop {
	if host == nil {
		host = app.Nop()
	}
	if getExec == nil {
		getExec = func() execenv.CommandExecutor { return execenv.LocalExecutor{} }
	}
	return &tuiRestartLoop{
		controller:                 controller,
		host:                       host,
		programPtr:                 programPtr,
		shellAfterExit:             shellAfterExit,
		commands:                   commands,
		openConfigLLMOnFirstLayout: openConfigLLMOnFirstLayout,
		getExec:                    getExec,
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
		case snap := <-l.shellAfterExit:
			saved = snap.Messages
			if snap.Mode == hostcmd.SubshellModeRemoteSSH {
				if err := execenv.RunInteractiveSSHShell(context.Background(), l.getExec()); err != nil {
					_, _ = fmt.Fprintf(os.Stderr, "delve-shell: remote shell: %v\n", err)
				}
			} else {
				runEmbeddedSubshellIgnoringExitCode()
			}
		default:
			return nil
		}
	}
}

func (l *tuiRestartLoop) runOneSession(saved *[]string, openConfigLLM bool) error {
	l.controller.SyncCurrentSessionPath()
	l.host.SetOpenConfigLLMOnFirstLayout(openConfigLLM)
	model := ui.NewModel(*saved, hostReadModel{host: l.host})
	model.CommandSender = ui.NewCommandChannelSender(l.commands)
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
