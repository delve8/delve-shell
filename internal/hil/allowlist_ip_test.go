package hil

import (
	"testing"

	"delve-shell/internal/config"
)

func TestAllowlist_DefaultIPReadOnly(t *testing.T) {
	w := NewAllowlist(config.DefaultLoadedAllowlist())

	for _, cmd := range []string{
		"ip -br addr 2>/dev/null",
		"ip addr",
		"ip a",
		"ip a s",
		"ip a s dev eth0",
		"ip l s dev eth0",
		"ip r s table main",
		"ip address show dev eth0",
		"ip link show dev eth0",
		"ip route show table main",
		"ip route get 1.1.1.1",
		"ip -4 route list table main 2>/dev/null",
	} {
		if !w.CommandAllowsAutoApprove(cmd) {
			t.Fatalf("CommandAllowsAutoApprove(%q) want true", cmd)
		}
	}

	for _, cmd := range []string{
		"ip addr add 10.0.0.2/24 dev eth0",
		"ip link set eth0 up",
		"ip route add default via 10.0.0.1",
		"ip route del default",
	} {
		if w.CommandAllowsAutoApprove(cmd) {
			t.Fatalf("CommandAllowsAutoApprove(%q) want false", cmd)
		}
	}
}
