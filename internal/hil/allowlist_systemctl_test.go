package hil

import (
	"testing"

	"delve-shell/internal/config"
)

func TestAllowlist_DefaultSystemctlReadOnly(t *testing.T) {
	w := NewAllowlist(config.DefaultLoadedAllowlist())

	t.Run("is-system-running with stderr discard", func(t *testing.T) {
		cmd := "systemctl is-system-running 2>/dev/null"
		if !w.CommandAllowsAutoApprove(cmd) {
			t.Fatalf("CommandAllowsAutoApprove(%q) want true", cmd)
		}
	})

	t.Run("status with no-pager", func(t *testing.T) {
		cmd := "systemctl status kubelet --no-pager"
		if !w.CommandAllowsAutoApprove(cmd) {
			t.Fatalf("CommandAllowsAutoApprove(%q) want true", cmd)
		}
	})

	t.Run("show property", func(t *testing.T) {
		cmd := "systemctl show kubelet -p ActiveState --value"
		if !w.CommandAllowsAutoApprove(cmd) {
			t.Fatalf("CommandAllowsAutoApprove(%q) want true", cmd)
		}
	})

	t.Run("list units filtered", func(t *testing.T) {
		cmd := "systemctl list-units --type=service --state=running --no-pager"
		if !w.CommandAllowsAutoApprove(cmd) {
			t.Fatalf("CommandAllowsAutoApprove(%q) want true", cmd)
		}
	})

	t.Run("restart still blocked", func(t *testing.T) {
		cmd := "systemctl restart kubelet"
		if w.CommandAllowsAutoApprove(cmd) {
			t.Fatalf("CommandAllowsAutoApprove(%q) want false", cmd)
		}
	})
}
