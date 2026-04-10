package execenv

import "testing"

func TestParseUserHost_RequiresExplicitUsername(t *testing.T) {
	_, _, err := parseUserHost("example.com")
	if err == nil {
		t.Fatal("expected missing username to fail")
	}
	if got, want := err.Error(), "ssh target must include username (user@host or user@host:port)"; got != want {
		t.Fatalf("error=%q want %q", got, want)
	}
}

func TestParseUserHost_AddsDefaultPort(t *testing.T) {
	user, hostPort, err := parseUserHost("alice@example.com")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if user != "alice" {
		t.Fatalf("user=%q want %q", user, "alice")
	}
	if hostPort != "example.com:22" {
		t.Fatalf("hostPort=%q want %q", hostPort, "example.com:22")
	}
}
