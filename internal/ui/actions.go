package ui

import (
	"delve-shell/internal/remoteauth"
	"delve-shell/internal/uitypes"
	"delve-shell/internal/uivm"
)

type ActionSender interface {
	Send(action uivm.UIAction) bool
}

type actionChannelSender struct {
	ch chan<- uivm.UIAction
}

func NewActionChannelSender(ch chan<- uivm.UIAction) ActionSender {
	if ch == nil {
		return nil
	}
	return actionChannelSender{ch: ch}
}

func (s actionChannelSender) Send(action uivm.UIAction) bool {
	select {
	case s.ch <- action:
		return true
	default:
		return false
	}
}

func (m Model) submitAction(text string) bool {
	if m.ActionSender == nil {
		return false
	}
	return m.ActionSender.Send(uivm.UIAction{
		Kind: uivm.UIActionSubmit,
		Text: text,
	})
}

func (m Model) relaySlashSubmitAction(p uitypes.SlashSubmitPayload) bool {
	if m.ActionSender == nil {
		return false
	}
	return m.ActionSender.Send(uivm.UIAction{
		Kind:        uivm.UIActionRelaySlashSubmit,
		SlashSubmit: p,
	})
}

func (m Model) requestSlashDispatchAction(line string) {
	if m.ActionSender == nil {
		return
	}
	_ = m.ActionSender.Send(uivm.UIAction{
		Kind: uivm.UIActionRequestSlashTrace,
		Text: line,
	})
}

func (m Model) traceSlashEnteredAction(line string) {
	if m.ActionSender == nil {
		return
	}
	_ = m.ActionSender.Send(uivm.UIAction{
		Kind: uivm.UIActionEnterSlashTrace,
		Text: line,
	})
}

func (m Model) EmitSubmitIntent(text string) bool {
	return m.submitAction(text)
}

func (m Model) EmitConfigUpdatedIntent() {
	if m.ActionSender == nil {
		return
	}
	_ = m.ActionSender.Send(uivm.UIAction{Kind: uivm.UIActionConfigUpdated})
}

func (m Model) EmitExecDirectIntent(cmd string) {
	if m.ActionSender == nil {
		return
	}
	_ = m.ActionSender.Send(uivm.UIAction{Kind: uivm.UIActionExecDirect, Text: cmd})
}

func (m Model) EmitCancelRequestIntent() bool {
	if m.ActionSender == nil {
		return false
	}
	return m.ActionSender.Send(uivm.UIAction{Kind: uivm.UIActionCancelRequested})
}

func (m Model) EmitShellSnapshotIntent(msgs []string) bool {
	if m.ActionSender == nil {
		return false
	}
	out := make([]string, len(msgs))
	copy(out, msgs)
	return m.ActionSender.Send(uivm.UIAction{Kind: uivm.UIActionShellSnapshot, Messages: out})
}

func (m Model) EmitRemoteOnTargetIntent(target string) bool {
	if m.ActionSender == nil {
		return false
	}
	return m.ActionSender.Send(uivm.UIAction{Kind: uivm.UIActionRemoteOnTarget, Text: target})
}

func (m Model) EmitRemoteOffIntent() bool {
	if m.ActionSender == nil {
		return false
	}
	return m.ActionSender.Send(uivm.UIAction{Kind: uivm.UIActionRemoteOff})
}

func (m Model) EmitRemoteAuthResponseIntent(resp remoteauth.Response) bool {
	if m.ActionSender == nil {
		return false
	}
	return m.ActionSender.Send(uivm.UIAction{Kind: uivm.UIActionRemoteAuthReply, RemoteAuthReply: resp})
}

func (m Model) EmitAllowlistAutoRunSyncIntent(v bool) {
	if m.ActionSender == nil {
		return
	}
	_ = m.ActionSender.Send(uivm.UIAction{Kind: uivm.UIActionAllowlistAutoRun, BoolValue: v})
}
