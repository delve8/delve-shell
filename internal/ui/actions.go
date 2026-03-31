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
	select {
	case s.ch <- command:
		return true
	default:
		return false
	}
}
