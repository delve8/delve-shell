// Package uipresenter is the host→TUI boundary: enqueue Bubble Tea messages in domain terms.
// Call sites (e.g. host controller) should prefer these methods over scattering ui message constructors.
package uipresenter

import (
	tea "github.com/charmbracelet/bubbletea"

	"delve-shell/internal/agent"
	"delve-shell/internal/ui"
)

// Sender delivers a message to the active tea.Program (typically via bus UI queue).
type Sender interface {
	Send(msg tea.Msg)
}

// Presenter wraps Sender with named operations for header, dialogs, transcript, and agent payloads.
type Presenter struct {
	send Sender
}

// New returns a Presenter that uses send for every outbound UI message. send must be non-blocking or
// blocking per product choice; host uses blocking enqueue to avoid dropping approvals.
func New(send Sender) *Presenter {
	if send == nil {
		send = nopSender{}
	}
	return &Presenter{send: send}
}

type nopSender struct{}

func (nopSender) Send(tea.Msg) {}

// Raw forwards a message as-is (escape hatch).
func (p *Presenter) Raw(msg tea.Msg) {
	if p == nil || msg == nil {
		return
	}
	p.send.Send(msg)
}

// --- Config / session ---

func (p *Presenter) ConfigReloaded() {
	p.Raw(ui.NewConfigReloadedMsg())
}

func (p *Presenter) SessionSwitched() {
	p.Raw(ui.NewSessionSwitchedMsg())
}

// --- Agent reply (transcript) ---

func (p *Presenter) AgentReply(reply string, err error) {
	p.Raw(ui.NewAgentReplyMsg(reply, err))
}

// --- System line (non-AI) ---

func (p *Presenter) SystemNotify(text string) {
	p.Raw(ui.NewSystemNotifyMsg(text))
}

// --- Command execution (transcript) ---

func (p *Presenter) CommandExecutedDirect(cmd, result string) {
	p.Raw(ui.NewCommandExecutedDirectMsg(cmd, result))
}

func (p *Presenter) CommandExecutedFromTool(cmd string, allowed bool, result string, sensitive, suggested bool) {
	p.Raw(ui.NewCommandExecutedFromToolMsg(cmd, allowed, result, sensitive, suggested))
}

// --- HIL: approval & sensitive confirmation (Agent payloads as tea.Msg) ---

func (p *Presenter) ShowApproval(req *agent.ApprovalRequest) {
	if req == nil {
		return
	}
	p.Raw(req)
}

func (p *Presenter) ShowSensitiveConfirmation(req *agent.SensitiveConfirmationRequest) {
	if req == nil {
		return
	}
	p.Raw(req)
}

// DispatchAgentUI maps agent-side UIEvents payloads to TUI messages.
func (p *Presenter) DispatchAgentUI(x any) {
	switch v := x.(type) {
	case *agent.ApprovalRequest:
		p.ShowApproval(v)
	case *agent.SensitiveConfirmationRequest:
		p.ShowSensitiveConfirmation(v)
	case agent.ExecEvent:
		p.CommandExecutedFromTool(v.Command, v.Allowed, v.Result, v.Sensitive, v.Suggested)
	}
}

// --- Remote / header ---

func (p *Presenter) RemoteStatus(active bool, label string) {
	p.Raw(ui.NewRemoteStatusMsg(active, label))
}

func (p *Presenter) RemoteConnectDone(success bool, label, errText string) {
	p.Raw(ui.NewRemoteConnectDoneMsg(success, label, errText))
}

func (p *Presenter) RemoteAuthPrompt(m ui.RemoteAuthPromptMsg) {
	p.Raw(m)
}

func (p *Presenter) RemoteAuthPromptPtr(m *ui.RemoteAuthPromptMsg) {
	if m == nil {
		return
	}
	p.Raw(*m)
}

// --- Completion cache (/run) ---

func (p *Presenter) RunCompletionCache(remoteLabel string, commands []string) {
	p.Raw(ui.NewRunCompletionCacheMsg(remoteLabel, commands))
}

// --- Overlays & async config checks (used by feature packages via tea.Msg today) ---

func (p *Presenter) OverlayClose() {
	p.Raw(ui.NewOverlayCloseMsg())
}

func (p *Presenter) OverlayShow(title, content string) {
	p.Raw(ui.NewOverlayShowMsg(title, content))
}

func (p *Presenter) ConfigLLMCheckDone(err error, correctedBaseURL string) {
	p.Raw(ui.NewConfigLLMCheckDoneMsg(err, correctedBaseURL))
}

func (p *Presenter) AddSkillRefsLoaded(refs []string) {
	p.Raw(ui.NewAddSkillRefsLoadedMsg(refs))
}

func (p *Presenter) AddSkillPathsLoaded(paths []string) {
	p.Raw(ui.NewAddSkillPathsLoadedMsg(paths))
}
