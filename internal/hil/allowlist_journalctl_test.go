package hil

import (
	"testing"

	"delve-shell/internal/config"
)

func TestAllowlist_DefaultJournalctlReadOnly(t *testing.T) {
	w := NewAllowlist(config.DefaultLoadedAllowlist())

	allowed := []string{
		"journalctl -u kubelet -n 100 --no-pager",
		"journalctl --since '1 hour ago' -p warning --no-pager",
		"journalctl -k -b --no-pager | tail -n 20",
		"journalctl --disk-usage",
	}
	for _, cmd := range allowed {
		if !w.CommandAllowsAutoApprove(cmd) {
			t.Fatalf("CommandAllowsAutoApprove(%q) want true", cmd)
		}
	}

	blocked := []string{
		"journalctl --rotate",
		"journalctl --flush",
		"journalctl --vacuum-time=1d",
		"journalctl --vacuum-size 1G",
		"journalctl --update-catalog",
	}
	for _, cmd := range blocked {
		if w.CommandAllowsAutoApprove(cmd) {
			t.Fatalf("CommandAllowsAutoApprove(%q) want false", cmd)
		}
	}
}
