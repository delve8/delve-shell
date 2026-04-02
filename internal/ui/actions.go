package ui

import (
	"delve-shell/internal/host/cmd"
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
	// Cancel must not be dropped when the channel is momentarily full; otherwise Esc clears [EXECUTING]
	// (or the user expects abort) while the host never runs handleCancelRequest / execCancelHub.Cancel().
	switch command.(type) {
	case hostcmd.CancelRequested:
		s.ch <- command
		return true
	default:
		select {
		case s.ch <- command:
			return true
		default:
			return false
		}
	}
}
