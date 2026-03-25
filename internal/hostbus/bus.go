package hostbus

import (
	"sync/atomic"

	tea "github.com/charmbracelet/bubbletea"

	"delve-shell/internal/ui"
)

// Kind identifies one domain event category on host bus.
type Kind string

const (
	KindUserSubmitted                 Kind = "user_submitted"
	KindConfigUpdated                 Kind = "config_updated"
	KindCancelRequested               Kind = "cancel_requested"
	KindExecDirectRequested           Kind = "exec_direct_requested"
	KindRemoteOnRequested             Kind = "remote_on_requested"
	KindRemoteOffRequested            Kind = "remote_off_requested"
	KindRemoteAuthResponseSubmitted   Kind = "remote_auth_response_submitted"
	KindAgentUIEmitted                Kind = "agent_ui_emitted"
	KindLLMRunCompleted               Kind = "llm_run_completed"
)

// Event carries one domain payload through the host bus.
type Event struct {
	Kind Kind

	UserText           string
	Command            string
	RemoteTarget       string
	RemoteAuthResponse ui.RemoteAuthResponse

	AgentUI any

	Reply string
	Err   error
}

// Bus is a bounded event queue for host-side orchestration.
type Bus struct {
	events chan Event
	uiMsgs chan tea.Msg
}

func New(capacity int) *Bus {
	if capacity <= 0 {
		capacity = 128
	}
	return &Bus{
		events: make(chan Event, capacity),
		uiMsgs: make(chan tea.Msg, 256),
	}
}

func (b *Bus) Events() <-chan Event { return b.events }

// Publish sends an event; returns false only when the queue is full.
func (b *Bus) Publish(e Event) bool {
	select {
	case b.events <- e:
		return true
	default:
		return false
	}
}

// PublishBlocking sends an event and waits for queue space.
func (b *Bus) PublishBlocking(e Event) {
	b.events <- e
}

func (b *Bus) EnqueueUI(msg tea.Msg) bool {
	if msg == nil {
		return false
	}
	select {
	case b.uiMsgs <- msg:
		return true
	default:
		return false
	}
}

func (b *Bus) EnqueueUIBlocking(msg tea.Msg) {
	if msg == nil {
		return
	}
	b.uiMsgs <- msg
}

// InputPorts are the external send-only channels wired to feature packages.
// They are bridged into Bus events by BridgeInputs.
type InputPorts struct {
	SubmitChan         chan string
	ConfigUpdatedChan  chan struct{}
	CancelRequestChan  chan struct{}
	ExecDirectChan     chan string
	RemoteOnChan       chan string
	RemoteOffChan      chan struct{}
	RemoteAuthRespChan chan ui.RemoteAuthResponse
	AgentUIChan        chan any
}

func NewInputPorts() InputPorts {
	return InputPorts{
		SubmitChan:         make(chan string, 8),
		ConfigUpdatedChan:  make(chan struct{}, 8),
		CancelRequestChan:  make(chan struct{}, 8),
		ExecDirectChan:     make(chan string, 8),
		RemoteOnChan:       make(chan string, 4),
		RemoteOffChan:      make(chan struct{}, 4),
		RemoteAuthRespChan: make(chan ui.RemoteAuthResponse, 4),
		AgentUIChan:        make(chan any, 64),
	}
}

// BridgeInputs forwards all external input channels into Bus events.
func BridgeInputs(stop <-chan struct{}, b *Bus, in InputPorts) {
	go func() {
		for {
			select {
			case <-stop:
				return
			case text := <-in.SubmitChan:
				b.PublishBlocking(Event{Kind: KindUserSubmitted, UserText: text})
			case <-in.ConfigUpdatedChan:
				b.PublishBlocking(Event{Kind: KindConfigUpdated})
			case <-in.CancelRequestChan:
				b.PublishBlocking(Event{Kind: KindCancelRequested})
			case cmd := <-in.ExecDirectChan:
				b.PublishBlocking(Event{Kind: KindExecDirectRequested, Command: cmd})
			case target := <-in.RemoteOnChan:
				b.PublishBlocking(Event{Kind: KindRemoteOnRequested, RemoteTarget: target})
			case <-in.RemoteOffChan:
				b.PublishBlocking(Event{Kind: KindRemoteOffRequested})
			case resp := <-in.RemoteAuthRespChan:
				b.PublishBlocking(Event{Kind: KindRemoteAuthResponseSubmitted, RemoteAuthResponse: resp})
			case x := <-in.AgentUIChan:
				b.PublishBlocking(Event{Kind: KindAgentUIEmitted, AgentUI: x})
			}
		}
	}()
}

// StartUIPump delivers UI messages from bus events to active tea program.
func StartUIPump(stop <-chan struct{}, b *Bus, currentP *atomic.Pointer[tea.Program]) {
	go func() {
		for {
			select {
			case <-stop:
				return
			case msg := <-b.uiMsgs:
				if p := currentP.Load(); p != nil {
					p.Send(msg)
				}
			}
		}
	}()
}
