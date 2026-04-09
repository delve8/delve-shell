package run

import (
	"testing"

	"delve-shell/internal/i18n"
)

func TestRootSlashOptions_UsesStableDescriptions(t *testing.T) {
	i18n.SetLang("en")
	opts := rootSlashOptions("en")
	got := map[string]string{}
	for _, opt := range opts {
		got[opt.Cmd] = opt.Desc
	}
	want := map[string]string{
		"/access":  "Switch execution target",
		"/skill":   "Use a skill",
		"/config":  "Manage models, hosts and skills",
		"/new":     "Start a new session",
		"/history": "Browse and switch sessions",
		"/help":    "Show help",
		"/quit":    "Quit delve-shell",
	}
	for cmd, desc := range want {
		if got[cmd] != desc {
			t.Fatalf("%s desc=%q want %q", cmd, got[cmd], desc)
		}
	}
}

func TestConfigSlashOptions_UsesStableDescriptions(t *testing.T) {
	i18n.SetLang("en")
	opts := configSlashOptions()
	got := map[string]string{}
	for _, opt := range opts {
		got[opt.Cmd] = opt.Desc
	}
	want := map[string]string{
		"/config del-remote":   "Remove a remote host",
		"/config del-skill":    "Remove an installed skill",
		"/config update-skill": "Update an installed skill",
		"/config model":        "Configure model settings",
	}
	for cmd, desc := range want {
		if got[cmd] != desc {
			t.Fatalf("%s desc=%q want %q", cmd, got[cmd], desc)
		}
	}
}
