package remote

import (
	"strings"

	"delve-shell/internal/i18n"
	"delve-shell/internal/service/remotesvc"
	"delve-shell/internal/ui"
)

func applyConfigAddRemote(m ui.Model, args string) ui.Model {
	lang := "en"
	parts := strings.Fields(args)
	if len(parts) < 1 {
		m.Messages = append(m.Messages, ui.ErrStyleRender(i18n.T(lang, i18n.KeyConfigPrefix)+"Usage: /config add-remote <user@host> [name] [identity_file]"))
		return m.RefreshViewport()
	}
	target := parts[0]
	name := ""
	identityFile := ""
	if len(parts) >= 2 {
		name = parts[1]
	}
	if len(parts) >= 3 {
		identityFile = parts[2]
	}
	if err := remotesvc.Add(target, name, identityFile); err != nil {
		m.Messages = append(m.Messages, ui.ErrStyleRender(i18n.T(lang, i18n.KeyConfigPrefix)+err.Error()))
		return m.RefreshViewport()
	}
	display := target
	if name != "" {
		display = name + " (" + target + ")"
	}
	prefix := i18n.T(lang, i18n.KeyDelveLabel) + " "
	m.Messages = append(m.Messages, ui.SuggestStyleRender(prefix+i18n.Tf(lang, i18n.KeyConfigRemoteAdded, display)))
	m.Messages = append(m.Messages, "")
	m = m.RefreshViewport()
	if m.Ports.ConfigUpdatedChan != nil {
		select {
		case m.Ports.ConfigUpdatedChan <- struct{}{}:
		default:
		}
	}
	return m
}

func applyConfigRemoveRemote(m ui.Model, nameOrTarget string) ui.Model {
	lang := "en"
	nameOrTarget = strings.TrimSpace(nameOrTarget)
	if nameOrTarget == "" {
		m.Messages = append(m.Messages, ui.ErrStyleRender(i18n.T(lang, i18n.KeyConfigPrefix)+"Usage: select a remote from /config del-remote list"))
		return m.RefreshViewport()
	}
	if err := remotesvc.Remove(nameOrTarget); err != nil {
		m.Messages = append(m.Messages, ui.ErrStyleRender(i18n.T(lang, i18n.KeyConfigPrefix)+err.Error()))
		return m.RefreshViewport()
	}
	prefix := i18n.T(lang, i18n.KeyDelveLabel) + " "
	m.Messages = append(m.Messages, ui.SuggestStyleRender(prefix+i18n.Tf(lang, i18n.KeyConfigRemoteRemoved, nameOrTarget)))
	m.Messages = append(m.Messages, "")
	m = m.RefreshViewport()
	if m.Ports.ConfigUpdatedChan != nil {
		select {
		case m.Ports.ConfigUpdatedChan <- struct{}{}:
		default:
		}
	}
	return m
}
