package ui

import (
	"delve-shell/internal/hostcmd"
	"delve-shell/internal/inputlifecycletype"
	"delve-shell/internal/remoteauth"
)

type CommandSender interface {
	Send(command hostcmd.Command) bool
}

type commandChannelSender struct {
	ch chan<- hostcmd.Command
}

func NewCommandChannelSender(ch chan<- hostcmd.Command) CommandSender {
	if ch == nil {
		return nil
	}
	return commandChannelSender{ch: ch}
}

func (s commandChannelSender) Send(command hostcmd.Command) bool {
	select {
	case s.ch <- command:
		return true
	default:
		return false
	}
}

func (m Model) EmitSubmissionIntent(sub inputlifecycletype.InputSubmission) bool {
	if m.CommandSender == nil {
		return false
	}
	return m.CommandSender.Send(hostcmd.Submission{Submission: sub})
}

func (m Model) EmitChatSubmitIntent(text string, source inputlifecycletype.SubmissionSource) bool {
	return m.EmitSubmissionIntent(inputlifecycletype.InputSubmission{
		Kind:    inputlifecycletype.SubmissionChat,
		Source:  source,
		RawText: text,
	})
}

func (m Model) EmitSessionNewIntent() bool {
	if m.CommandSender == nil {
		return false
	}
	return m.CommandSender.Send(hostcmd.SessionNew{})
}

func (m Model) EmitSessionSwitchIntent(sessionID string) bool {
	if m.CommandSender == nil {
		return false
	}
	return m.CommandSender.Send(hostcmd.SessionSwitch{SessionID: sessionID})
}

func (m Model) EmitConfigUpdatedIntent() {
	if m.CommandSender == nil {
		return
	}
	_ = m.CommandSender.Send(hostcmd.ConfigUpdated{})
}

func (m Model) EmitExecDirectIntent(cmd string) {
	if m.CommandSender == nil {
		return
	}
	_ = m.CommandSender.Send(hostcmd.ExecDirect{Command: cmd})
}

func (m Model) EmitShellSnapshotIntent(msgs []string) bool {
	return m.EmitShellSnapshotIntentWithMode(msgs, hostcmd.SubshellModeLocalBash)
}

// EmitShellSnapshotIntentWithMode sends transcript lines for /bash; mode selects local bash vs remote SSH shell.
func (m Model) EmitShellSnapshotIntentWithMode(msgs []string, mode hostcmd.SubshellMode) bool {
	if m.CommandSender == nil {
		return false
	}
	out := make([]string, len(msgs))
	copy(out, msgs)
	return m.CommandSender.Send(hostcmd.ShellSnapshot{Messages: out, Mode: mode})
}

func (m Model) EmitRemoteOnTargetIntent(target string) bool {
	if m.CommandSender == nil {
		return false
	}
	return m.CommandSender.Send(hostcmd.RemoteOnTarget{Target: target})
}

func (m Model) EmitRemoteOffIntent() bool {
	if m.CommandSender == nil {
		return false
	}
	return m.CommandSender.Send(hostcmd.RemoteOff{})
}

func (m Model) EmitRemoteAuthResponseIntent(resp remoteauth.Response) bool {
	if m.CommandSender == nil {
		return false
	}
	return m.CommandSender.Send(hostcmd.RemoteAuthReply{Response: resp})
}

func (m Model) EmitAllowlistAutoRunSyncIntent(v bool) {
	if m.CommandSender == nil {
		return
	}
	_ = m.CommandSender.Send(hostcmd.AllowlistAutoRun{Enabled: v})
}
