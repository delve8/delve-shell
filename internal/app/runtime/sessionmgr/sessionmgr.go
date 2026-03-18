package sessionmgr

import (
	"fmt"
	"sync"
	"time"

	"delve-shell/internal/history"
)

// Manager owns the current session and serializes session switching/closing.
// It is safe for concurrent use.
type Manager struct {
	mu      sync.Mutex
	session *history.Session
}

func New(initial *history.Session) *Manager {
	return &Manager{session: initial}
}

// Current returns the current session pointer (may be nil).
// Callers must not close it; use CloseCurrent/CloseAll.
func (m *Manager) Current() *history.Session {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.session
}

// SwitchTo opens an existing session file and makes it current.
// The previous session (if any) is closed.
func (m *Manager) SwitchTo(path string) (*history.Session, error) {
	s, err := history.OpenSession(path)
	if err != nil {
		return nil, fmt.Errorf("open session: %w", err)
	}

	m.mu.Lock()
	old := m.session
	m.session = s
	m.mu.Unlock()

	if old != nil {
		_ = old.Close()
	}
	return s, nil
}

// NewSession creates a new session id using the provided suffix generator, makes it current,
// and closes the previous session (if any). The new session is created lazily on first write
// by the history package, but it is returned immediately.
func (m *Manager) NewSession(idSuffix func() string) (*history.Session, error) {
	id := time.Now().Format("060102-150405")
	if idSuffix != nil {
		id += "-" + idSuffix()
	}
	s, err := history.NewSession(id)
	if err != nil {
		return nil, fmt.Errorf("new session: %w", err)
	}

	m.mu.Lock()
	old := m.session
	m.session = s
	m.mu.Unlock()

	if old != nil {
		_ = old.Close()
	}
	return s, nil
}

// CloseAll closes the current session (if any) and clears it.
// It is safe to call multiple times.
func (m *Manager) CloseAll() {
	m.mu.Lock()
	old := m.session
	m.session = nil
	m.mu.Unlock()

	if old != nil {
		_ = old.Close()
	}
}

