package interactive

import (
	"sync/atomic"

	tea "github.com/charmbracelet/bubbletea"

	"delve-shell/internal/config"
	"delve-shell/internal/history"
	"delve-shell/internal/host/app"
	"delve-shell/internal/host/bus"
	"delve-shell/internal/host/cmd"
	"delve-shell/internal/host/controller"
	"delve-shell/internal/remote/execenv"
	"delve-shell/internal/runtime/execcancel"
	"delve-shell/internal/runtime/executormgr"
	"delve-shell/internal/runtime/runnermgr"
	"delve-shell/internal/runtime/sessionmgr"
)

// hostStack wires bus, controller, runner, executor, and app.Runtime for one interactive session.
type hostStack struct {
	controller *controller.Controller
	rt         *app.Runtime
	currentP   *atomic.Pointer[tea.Program]
	shellSnap  <-chan hostcmd.ShellSnapshot
	commands   chan hostcmd.Command
	getExec    func() execenv.CommandExecutor
}

// wireHostStack builds runners, host bus ports, controller, and *app.Runtime. Caller must Start() the controller.
func wireHostStack(
	stop <-chan struct{},
	pf *PreflightResult,
	sessions *sessionmgr.Manager,
	syncSessionPath func(string),
) *hostStack {
	hostBus := bus.New(512)
	ports := bus.NewInputPorts()

	executors := executormgr.New()
	getExecutor := func() execenv.CommandExecutor { return executors.Get() }

	execCancelHub := execcancel.New()
	rt := app.NewRuntime()

	runners := runnermgr.New(runnermgr.Options{
		RulesText: pf.RulesText,
		LoadConfig: func() (*config.Config, error) {
			return config.LoadEnsured()
		},
		LoadAllowlist: func() ([]config.AllowlistEntry, error) {
			return config.LoadAllowlist()
		},
		LoadSensitivePatterns: func() ([]string, error) {
			return config.LoadSensitivePatterns()
		},
		SessionProvider:  func() *history.Session { return sessions.Current() },
		ExecutorProvider: getExecutor,
		OfflineMode:      func() bool { return rt.Offline() },
		UIEvents:         ports.AgentUIChan,
		ExecCancelHub:    execCancelHub,
	})

	shellRequestedChan := make(chan hostcmd.ShellSnapshot, 1)
	rt.WireSend(&app.Send{
		Submission:     ports.SubmissionChan,
		ConfigUpdated:  ports.ConfigUpdatedChan,
		CancelRequest:  ports.CancelRequestChan,
		ExecDirect:     ports.ExecDirectChan,
		RemoteOn:       ports.RemoteOnChan,
		RemoteOff:      ports.RemoteOffChan,
		RemoteAuthResp: ports.RemoteAuthRespChan,
		ShellSnapshot:  shellRequestedChan,
	})

	var currentP atomic.Pointer[tea.Program]
	commands := make(chan hostcmd.Command, 128)
	controller := controller.New(controller.Options{
		Stop:            stop,
		Bus:             hostBus,
		Inputs:          ports,
		CurrentP:        &currentP,
		Commands:        commands,
		ShellSnapshot:   shellRequestedChan,
		Sessions:        sessions,
		Runners:         runners,
		Executors:       executors,
		GetExec:         getExecutor,
		SyncSessionPath: syncSessionPath,
		Runtime:         rt,
		ExecCancelHub:   execCancelHub,
	})
	controller.Start()

	return &hostStack{
		controller: controller,
		rt:         rt,
		currentP:   &currentP,
		shellSnap:  shellRequestedChan,
		commands:   commands,
		getExec:    getExecutor,
	}
}
