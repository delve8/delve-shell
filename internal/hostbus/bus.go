package hostbus

import (
	"sync/atomic"

	tea "github.com/charmbracelet/bubbletea"

	"delve-shell/internal/agent/hiltypes"
	"delve-shell/internal/hostroute"
	"delve-shell/internal/remoteauth"
)

// Kind identifies one domain event category on host bus.
//
// Coverage: submit routing (new session / switch session / chat to LLM), config reload, cancel and direct
// exec, remote connect/off/auth, agent→UI HIL (approval / sensitive / exec result / unknown passthrough),
// LLM run completion, and slash dispatch trace (KindSlashEntered; execution remains in TUI).
//
// Architecture draft names (docs/ui-refactor-handoff.md §10.4) map to Kind via [Kind.SemanticLabel];
// wire values (string constants below) remain the stable contract for tests and persistence.
type Kind string

const (
	KindSessionNewRequested            Kind = "session_new_requested"
	KindSessionSwitchRequested         Kind = "session_switch_requested"
	KindUserChatSubmitted              Kind = "user_chat_submitted"
	KindConfigUpdated                  Kind = "config_updated"
	KindCancelRequested                Kind = "cancel_requested"
	KindExecDirectRequested            Kind = "exec_direct_requested"
	KindRemoteOnRequested              Kind = "remote_on_requested"
	KindRemoteOffRequested             Kind = "remote_off_requested"
	KindRemoteAuthResponseSubmitted    Kind = "remote_auth_response_submitted"
	KindApprovalRequested              Kind = "approval_requested"
	KindSensitiveConfirmationRequested Kind = "sensitive_confirmation_requested"
	KindAgentExecEvent                 Kind = "agent_exec_event"
	KindAgentUnknown                   Kind = "agent_unknown"
	KindLLMRunCompleted                Kind = "llm_run_completed"
	// KindSlashEntered is emitted after the TUI has successfully dispatched a slash command (exact or prefix).
	// Execution stays in the UI/registry; the controller observes the event for tracing and future routing.
	KindSlashEntered Kind = "slash_entered"
)

// Event carries one domain payload through the host bus.
type Event struct {
	Kind Kind

	UserText           string
	SessionID          string
	Command            string
	RemoteTarget       string
	RemoteAuthResponse remoteauth.Response

	Approval  *hiltypes.ApprovalRequest
	Sensitive *hiltypes.SensitiveConfirmationRequest
	AgentExec hiltypes.ExecEvent
	AgentUI   any // fallback when Kind == KindAgentUnknown

	Reply string
	Err   error
}

// Bus is a bounded event queue for host-side orchestration.
type Bus struct {
	events chan Event
	uiMsgs chan tea.Msg

	publishHook PublishHook // optional; see WithPublishHook
}

// New builds a bus. Options are applied in order (e.g. [WithPublishHook]).
func New(capacity int, opts ...BusOption) *Bus {
	if capacity <= 0 {
		capacity = 128
	}
	b := &Bus{
		events: make(chan Event, capacity),
		uiMsgs: make(chan tea.Msg, 256),
	}
	for _, o := range opts {
		if o != nil {
			o(b)
		}
	}
	return b
}

func (b *Bus) Events() <-chan Event { return b.events }

// Publish sends an event; returns false only when the queue is full.
func (b *Bus) Publish(e Event) bool {
	select {
	case b.events <- e:
		b.notifyPublish(e, true)
		return true
	default:
		b.notifyPublish(e, false)
		return false
	}
}

// PublishBlocking sends an event and waits for queue space.
func (b *Bus) PublishBlocking(e Event) {
	b.events <- e
	b.notifyPublish(e, true)
}

func (b *Bus) notifyPublish(e Event, accepted bool) {
	if b.publishHook != nil {
		b.publishHook(e, accepted)
	}
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
	RemoteAuthRespChan chan remoteauth.Response
	SlashTraceChan     chan string
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
		RemoteAuthRespChan: make(chan remoteauth.Response, 4),
		SlashTraceChan:     make(chan string, 8),
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
				route := hostroute.ClassifyUserSubmit(text)
				switch route.Kind {
				case hostroute.UserSubmitNewSession:
					b.PublishBlocking(Event{Kind: KindSessionNewRequested})
				case hostroute.UserSubmitSwitchSession:
					b.PublishBlocking(Event{Kind: KindSessionSwitchRequested, SessionID: route.SessionID})
				default:
					b.PublishBlocking(Event{Kind: KindUserChatSubmitted, UserText: text})
				}
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
			case line := <-in.SlashTraceChan:
				b.PublishBlocking(Event{Kind: KindSlashEntered, UserText: line})
			case x := <-in.AgentUIChan:
				if ev, ok := bridgeAgentUI(x); ok {
					b.PublishBlocking(ev)
				}
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

func bridgeAgentUI(x any) (Event, bool) {
	if x == nil {
		return Event{}, false
	}
	switch v := x.(type) {
	case *hiltypes.ApprovalRequest:
		return Event{Kind: KindApprovalRequested, Approval: v}, true
	case *hiltypes.SensitiveConfirmationRequest:
		return Event{Kind: KindSensitiveConfirmationRequested, Sensitive: v}, true
	case hiltypes.ExecEvent:
		return Event{Kind: KindAgentExecEvent, AgentExec: v}, true
	default:
		return Event{Kind: KindAgentUnknown, AgentUI: x}, true
	}
}
