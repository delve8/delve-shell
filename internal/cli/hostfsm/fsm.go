// Package hostfsm is a small table-driven state machine for the CLI host (idle vs LLM turn, etc.).
// Transitions are registered from init() in register_*.go files.
package hostfsm

import "sync"

// State is an opaque host phase label.
type State string

// Event is something that moves the host FSM when a transition matches.
type Event string

// Context is passed to transition actions; extend when a transition needs shared deps.
type Context struct{}

// Transition is one row: in state From, on Event, go To and optionally run Action.
type Transition struct {
	From   State
	On     Event
	To     State
	Action func(*Context)
}

var (
	mu          sync.RWMutex
	transitions []Transition
)

// Register adds a transition row. Call from init() in feature packages.
func Register(t Transition) {
	mu.Lock()
	defer mu.Unlock()
	transitions = append(transitions, t)
}

// Machine holds current State and a compiled lookup table.
type Machine struct {
	mu    sync.Mutex
	state State
	table map[stateEvent]Transition
}

type stateEvent struct {
	from State
	on   Event
}

// NewMachine builds a Machine from all registered transitions. Call once at startup.
func NewMachine(initial State) *Machine {
	mu.RLock()
	defer mu.RUnlock()
	tab := make(map[stateEvent]Transition, len(transitions))
	for _, t := range transitions {
		key := stateEvent{from: t.From, on: t.On}
		tab[key] = t
	}
	return &Machine{state: initial, table: tab}
}

// State returns the current state (for tests / diagnostics).
func (m *Machine) State() State {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.state
}

// Apply looks up (current, e) and moves to To, running Action if set.
// Returns false if no transition is registered (caller may ignore or log).
func (m *Machine) Apply(ctx *Context, e Event) bool {
	m.mu.Lock()
	defer m.mu.Unlock()
	key := stateEvent{from: m.state, on: e}
	tr, ok := m.table[key]
	if !ok {
		return false
	}
	m.state = tr.To
	if tr.Action != nil {
		tr.Action(ctx)
	}
	return true
}
