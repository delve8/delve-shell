package runnermgr

import (
	"context"
	"fmt"
	"sync"

	"delve-shell/internal/agent"
	"delve-shell/internal/config"
	"delve-shell/internal/hil"
	"delve-shell/internal/history"
	"delve-shell/internal/remote/execenv"
	"delve-shell/internal/runtime/execcancel"
)

// Manager owns the Runner cache and rebuild logic.
// It is safe for concurrent use.
type Manager struct {
	mu sync.Mutex
	r  *agent.Runner

	loadConfig             func() (*config.Config, error)
	loadAllowlist          func() ([]config.AllowlistEntry, error)
	loadSensitivePatterns  func() ([]string, error)
	sessionProvider        func() *history.Session
	executorProvider       func() execenv.CommandExecutor
	execContextDescription func() string
	offlineMode            func() bool

	rulesText string

	uiEvents      chan<- any
	execCancelHub *execcancel.Hub
}

type Options struct {
	RulesText string

	LoadConfig            func() (*config.Config, error)
	LoadAllowlist         func() ([]config.AllowlistEntry, error)
	LoadSensitivePatterns func() ([]string, error)

	SessionProvider  func() *history.Session
	ExecutorProvider func() execenv.CommandExecutor
	// ExecContextDescription optional; see agent.RunnerUILoopInput.ExecContextDescription.
	ExecContextDescription func() string
	// OfflineMode when true builds a runner without skill tools and with offline execute_command behavior.
	OfflineMode func() bool

	UIEvents chan<- any // *ApprovalRequest | *SensitiveConfirmationRequest | ExecEvent | ExecStreamStart | ExecStreamLine
	// ExecCancelHub optional; host uses it when the user presses Esc during [EXECUTING].
	ExecCancelHub *execcancel.Hub
}

func New(opts Options) *Manager {
	m := &Manager{
		loadConfig:             opts.LoadConfig,
		loadAllowlist:          opts.LoadAllowlist,
		loadSensitivePatterns:  opts.LoadSensitivePatterns,
		sessionProvider:        opts.SessionProvider,
		executorProvider:       opts.ExecutorProvider,
		execContextDescription: opts.ExecContextDescription,
		offlineMode:            opts.OfflineMode,
		rulesText:              opts.RulesText,
		uiEvents:               opts.UIEvents,
		execCancelHub:          opts.ExecCancelHub,
	}
	return m
}

// Invalidate drops the cached runner so the next Get will rebuild it.
func (m *Manager) Invalidate() {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.r = nil
}

// Get returns a cached Runner or builds a new one.
func (m *Manager) Get(ctx context.Context) (*agent.Runner, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.r != nil {
		return m.r, nil
	}
	if m.loadConfig == nil {
		return nil, fmt.Errorf("runnermgr: LoadConfig is nil")
	}
	cfg, err := m.loadConfig()
	if err != nil {
		return nil, err
	}
	allowlistEntries, err := m.loadAllowlist()
	if err != nil {
		return nil, fmt.Errorf("load allowlist: %w", err)
	}
	allowlist := hil.NewAllowlist(allowlistEntries)
	offline := m.offlineMode != nil && m.offlineMode()
	sensitivePatterns, err := m.loadSensitivePatterns()
	if err != nil {
		return nil, fmt.Errorf("load sensitive patterns: %w", err)
	}
	sensitiveMatcher := hil.NewSensitiveMatcher(sensitivePatterns)

	r, err := agent.NewRunner(ctx, agent.RunnerOptions{
		Config: cfg,
		HIL: agent.RunnerHILInput{
			Allowlist:        allowlist,
			SensitiveMatcher: sensitiveMatcher,
		},
		Session: agent.RunnerSessionInput{
			Session:   m.sessionProvider(),
			RulesText: m.rulesText,
		},
		UILoop: agent.RunnerUILoopInput{
			UIEvents:               m.uiEvents,
			ExecutorProvider:       m.executorProvider,
			ExecCancelHub:          m.execCancelHub,
			ExecContextDescription: m.execContextDescription,
		},
		Offline: offline,
	})
	if err != nil {
		return nil, err
	}
	m.r = r
	return r, nil
}
