package controller

import (
	"context"
	"os"
	"testing"

	tea "github.com/charmbracelet/bubbletea"

	"delve-shell/internal/cli/hostfsm"
	"delve-shell/internal/host/bus"
	"delve-shell/internal/runtime/sessionmgr"
	"delve-shell/internal/uipresenter"
)

type recordSender struct {
	msgs []tea.Msg
}

func (r *recordSender) Send(msg tea.Msg) {
	if msg == nil {
		return
	}
	r.msgs = append(r.msgs, msg)
}

type fakeExec struct {
	stdout   string
	stderr   string
	exitCode int
	err      error
	lastCmd  string
}

func (f *fakeExec) Run(ctx context.Context, command string) (stdout, stderr string, exitCode int, err error) {
	_ = ctx
	f.lastCmd = command
	return f.stdout, f.stderr, f.exitCode, f.err
}

func newTestControllerWithPresenter(sender *recordSender) *Controller {
	stop := make(chan struct{})
	c := &Controller{
		stop:     stop,
		bus:      bus.New(64),
		ui:       uipresenter.New(sender),
		fsm:      hostfsm.NewMachine(hostfsm.StateIdle),
		sessions: &sessionmgr.Manager{},
	}
	return c
}

func TestMain(m *testing.M) {
	os.Exit(m.Run())
}
