package runnermgr

import (
	"context"
	"fmt"
	"sync"
	"sync/atomic"

	"delve-shell/internal/agent"
	"delve-shell/internal/config"
	"delve-shell/internal/execenv"
	"delve-shell/internal/hil"
	"delve-shell/internal/history"
)

// Manager owns the Runner cache and rebuild logic.
// It is safe for concurrent use.
type Manager struct {
	mu sync.Mutex
	r  *agent.Runner

	loadConfig            func() (*config.Config, error)
	loadAllowlist         func() ([]config.AllowlistEntry, error)
	loadSensitivePatterns func() ([]string, error)
	sessionProvider       func() *history.Session
	executorProvider      func() execenv.CommandExecutor

	rulesText string

	allowlistAutoRun atomic.Bool

	uiEvents chan<- any
}

type Options struct {
	RulesText string

	LoadConfig            func() (*config.Config, error)
	LoadAllowlist         func() ([]config.AllowlistEntry, error)
	LoadSensitivePatterns func() ([]string, error)

	SessionProvider  func() *history.Session
	ExecutorProvider func() execenv.CommandExecutor

	AllowlistAutoRun bool

	UIEvents chan<- any // *ApprovalRequest | *SensitiveConfirmationRequest | ExecEvent
}

func New(opts Options) *Manager {
	m := &Manager{
		loadConfig:            opts.LoadConfig,
		loadAllowlist:         opts.LoadAllowlist,
		loadSensitivePatterns: opts.LoadSensitivePatterns,
		sessionProvider:       opts.SessionProvider,
		executorProvider:      opts.ExecutorProvider,
		rulesText:             opts.RulesText,
		uiEvents:              opts.UIEvents,
	}
	m.allowlistAutoRun.Store(opts.AllowlistAutoRun)
	return m
}

// Invalidate drops the cached runner so the next Get will rebuild it.
func (m *Manager) Invalidate() {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.r = nil
}

// SetAllowlistAutoRun updates the runtime value (used for UI header + approval options) and invalidates the runner.
func (m *Manager) SetAllowlistAutoRun(v bool) {
	m.allowlistAutoRun.Store(v)
	m.Invalidate()
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
	sensitivePatterns, err := m.loadSensitivePatterns()
	if err != nil {
		return nil, fmt.Errorf("load sensitive patterns: %w", err)
	}
	sensitiveMatcher := hil.NewSensitiveMatcher(sensitivePatterns)

	autoRun := m.allowlistAutoRun.Load()
	r, err := agent.NewRunner(ctx, agent.RunnerOptions{
		Config:           cfg,
		AllowlistAutoRun: &autoRun,
		Allowlist:        allowlist,
		SensitiveMatcher: sensitiveMatcher,
		Session:          m.sessionProvider(),
		RulesText:        m.rulesText,
		UIEvents:         m.uiEvents,
		ExecutorProvider: m.executorProvider,
	})
	if err != nil {
		return nil, err
	}
	m.r = r
	return r, nil
}
