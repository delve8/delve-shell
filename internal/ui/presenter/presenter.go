// Package uipresenter is the host→TUI boundary: enqueue Bubble Tea messages in domain terms.
// Call sites (e.g. host controller) should prefer these methods over scattering ui message constructors.
package uipresenter

import (
	"context"
	"errors"

	tea "github.com/charmbracelet/bubbletea"

	"delve-shell/internal/hil/types"
	"delve-shell/internal/remote"
	"delve-shell/internal/ui"
	"delve-shell/internal/ui/uivm"
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

// TranscriptAppend appends semantic transcript lines.
func (p *Presenter) TranscriptAppend(lines []uivm.Line) {
	if len(lines) == 0 {
		return
	}
	p.Raw(ui.TranscriptAppendMsg{Lines: lines})
}

// TranscriptReplace replaces the whole transcript.
func (p *Presenter) TranscriptReplace(lines []uivm.Line) {
	p.Raw(ui.TranscriptReplaceMsg{Lines: lines})
}

// --- Config / session ---

func (p *Presenter) ConfigReloaded() {
	p.Raw(ui.TranscriptAppendMsg{Lines: []uivm.Line{
		{Kind: uivm.LineSystemSuggest, Text: "Config reloaded."},
		{Kind: uivm.LineBlank},
	}})
}

// --- Agent reply (transcript) ---

func (p *Presenter) AgentReply(reply string, err error) {
	if err != nil {
		// Keep UI pure of agent/config; presenter provides a stable, human-readable error.
		if errors.Is(err, context.Canceled) {
			p.Raw(ui.TranscriptAppendMsg{
				ClearWaitingForAI: true,
				Lines: []uivm.Line{
					{Kind: uivm.LineSystemSuggest, Text: "Cancelled."},
					{Kind: uivm.LineBlank},
				},
			})
			return
		}
		p.Raw(ui.TranscriptAppendMsg{
			ClearWaitingForAI: true,
			Lines: []uivm.Line{
				{Kind: uivm.LineSystemError, Text: err.Error()},
				{Kind: uivm.LineBlank},
			},
		})
		return
	}
	if reply == "" {
		p.Raw(ui.TranscriptAppendMsg{ClearWaitingForAI: true})
		return
	}
	p.Raw(ui.TranscriptAppendMsg{
		ClearWaitingForAI: true,
		Lines: []uivm.Line{
			{Kind: uivm.LineAI, Text: reply},
		},
	})
}

// --- System line (non-AI) ---

func (p *Presenter) SystemNotify(text string) {
	if text == "" {
		return
	}
	p.Raw(ui.TranscriptAppendMsg{Lines: []uivm.Line{
		{Kind: uivm.LineSystemSuggest, Text: text},
		{Kind: uivm.LineBlank},
	}})
}

// --- Command execution (transcript) ---

func (p *Presenter) CommandExecutedDirect(cmd, result string) {
	p.CommandExecutedFromTool(cmd, false, result, false, false, true)
}

func execRunTag(allowed, suggested, direct bool) string {
	tag := "approved"
	if direct {
		tag = "direct"
	} else if suggested {
		tag = "suggested"
	} else if allowed {
		tag = "allowlist"
	}
	return tag
}

func (p *Presenter) CommandExecutedFromTool(cmd string, allowed bool, result string, sensitive, suggested, direct bool) {
	// Presenter builds transcript semantics; UI owns styling and wrapping.
	runLine := "Run: " + cmd + " (" + execRunTag(allowed, suggested, direct) + ")"
	lines := []uivm.Line{{Kind: uivm.LineExec, Text: runLine}}
	if sensitive {
		lines = append(lines, uivm.Line{Kind: uivm.LineSystemSuggest, Text: "Result contains sensitive data."})
	}
	if result != "" {
		lines = append(lines, uivm.Line{Kind: uivm.LineResult, Text: result})
	}
	lines = append(lines, uivm.Line{Kind: uivm.LineBlank})
	p.Raw(ui.TranscriptAppendMsg{Lines: lines})
}

// ExecStreamBegin appends the Run: line before streamed stdout/stderr lines.
func (p *Presenter) ExecStreamBegin(cmd string, allowed, suggested, direct bool) {
	runLine := "Run: " + cmd + " (" + execRunTag(allowed, suggested, direct) + ")"
	p.Raw(ui.TranscriptAppendMsg{Lines: []uivm.Line{{Kind: uivm.LineExec, Text: runLine}}})
}

// ExecStreamLineOut appends one streamed command output line (stdout or stderr).
func (p *Presenter) ExecStreamLineOut(line string, stderr bool) {
	if line == "" {
		return
	}
	text := line
	if stderr {
		text = "stderr: " + line
	}
	p.Raw(ui.TranscriptAppendMsg{Lines: []uivm.Line{{Kind: uivm.LineResult, Text: text}}})
}

// CommandExecutedStreamEnd finishes a streamed run: sensitive note, exit/footer tail, blank line (no second Run: line).
func (p *Presenter) CommandExecutedStreamEnd(sensitive bool, tail string) {
	var lines []uivm.Line
	if sensitive {
		lines = append(lines, uivm.Line{Kind: uivm.LineSystemSuggest, Text: "Result contains sensitive data."})
	}
	if tail != "" {
		lines = append(lines, uivm.Line{Kind: uivm.LineResult, Text: tail})
	}
	lines = append(lines, uivm.Line{Kind: uivm.LineBlank})
	p.Raw(ui.TranscriptAppendMsg{Lines: lines})
}

// --- HIL: approval & sensitive confirmation (Agent payloads as tea.Msg) ---

func (p *Presenter) ShowApproval(req *hiltypes.ApprovalRequest) {
	if req == nil {
		return
	}
	// Map domain request to UI view-model; respond writes back into domain channel.
	p.Raw(ui.ChoiceCardShowMsg{PendingApproval: &uivm.PendingApproval{
		Command:   req.Command,
		Summary:   req.Summary,
		Reason:    req.Reason,
		RiskLevel: req.RiskLevel,
		SkillName: req.SkillName,
		Respond: func(r uivm.ApprovalResponse) {
			req.ResponseCh <- hiltypes.ApprovalResponse{Approved: r.Approved, CopyRequested: r.CopyRequested}
		},
	}})
}

func (p *Presenter) ShowSensitiveConfirmation(req *hiltypes.SensitiveConfirmationRequest) {
	if req == nil {
		return
	}
	p.Raw(ui.ChoiceCardShowMsg{PendingSensitive: &uivm.PendingSensitive{
		Command: req.Command,
		Respond: func(c uivm.SensitiveChoice) {
			switch c {
			case uivm.SensitiveRunAndStore:
				req.ResponseCh <- hiltypes.SensitiveRunAndStore
			case uivm.SensitiveRunNoStore:
				req.ResponseCh <- hiltypes.SensitiveRunNoStore
			default:
				req.ResponseCh <- hiltypes.SensitiveRefuse
			}
		},
	}})
}

// ShowOfflinePaste opens the offline manual-relay paste dialog; response is sent on req.ResponseCh.
func (p *Presenter) ShowOfflinePaste(req *hiltypes.OfflinePasteRequest) {
	if req == nil || req.ResponseCh == nil {
		return
	}
	p.Raw(ui.OfflinePasteShowMsg{Pending: &uivm.PendingOfflinePaste{
		Command:   req.Command,
		Reason:    req.Reason,
		RiskLevel: req.RiskLevel,
		Respond: func(text string, cancelled bool) {
			req.ResponseCh <- hiltypes.OfflinePasteResponse{Text: text, Cancelled: cancelled}
		},
	}})
}

// DispatchAgentUI maps agent-side UIEvents payloads to TUI messages.
func (p *Presenter) DispatchAgentUI(x any) {
	switch v := x.(type) {
	case *hiltypes.ApprovalRequest:
		p.ShowApproval(v)
	case *hiltypes.SensitiveConfirmationRequest:
		p.ShowSensitiveConfirmation(v)
	case *hiltypes.OfflinePasteRequest:
		p.ShowOfflinePaste(v)
	case hiltypes.ExecStreamStart:
		p.ExecStreamBegin(v.Command, v.Allowed, v.Suggested, v.Direct)
	case hiltypes.ExecStreamLine:
		p.ExecStreamLineOut(v.Line, v.Stderr)
	case hiltypes.CommandExecutionState:
		p.CommandExecutionActive(v.Active)
	case hiltypes.AgentNotify:
		p.SystemNotify(v.Text)
	case hiltypes.ExecEvent:
		if v.Streamed {
			p.CommandExecutedStreamEnd(v.Sensitive, v.Result)
		} else {
			p.CommandExecutedFromTool(v.Command, v.Allowed, v.Result, v.Sensitive, v.Suggested, false)
		}
	}
}

// CommandExecutionActive toggles [EXECUTING] footer state and input lock while a shell command runs.
func (p *Presenter) CommandExecutionActive(active bool) {
	p.Raw(ui.CommandExecutionStateMsg{Active: active})
}

// --- Remote / header ---

func (p *Presenter) RemoteStatus(active bool, label string, offline bool) {
	p.Raw(remote.ExecutionChangedMsg{Active: active, Label: label, Offline: offline})
}

func (p *Presenter) RemoteConnectDone(success bool, label, errText string) {
	p.Raw(remote.ConnectDoneMsg{Success: success, Label: label, Err: errText})
}

func (p *Presenter) RemoteAuthPrompt(target, errText string, useConfiguredIdentity bool) {
	p.Raw(remote.AuthPromptMsg{Target: target, Err: errText, UseConfiguredIdentity: useConfiguredIdentity})
}

// --- Completion cache (/exec) ---

func (p *Presenter) RunCompletionCache(remoteLabel string, commands []string) {
	p.Raw(remote.RunCompletionCacheMsg{RemoteLabel: remoteLabel, Commands: commands})
}
