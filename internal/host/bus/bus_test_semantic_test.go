package bus

import (
	"strings"
	"testing"

	"delve-shell/internal/remoteauth"
)

func TestSemanticLabel_MapsDraftNames(t *testing.T) {
	if g, w := KindUserChatSubmitted.SemanticLabel(), "AIRequested"; g != w {
		t.Fatalf("KindUserChatSubmitted: got %q want %q", g, w)
	}
	if g, w := KindConfigUpdated.SemanticLabel(), "ConfigReloaded"; g != w {
		t.Fatalf("KindConfigUpdated: got %q want %q", g, w)
	}
}

func TestRedactedSummary_OmitsRemoteAuthSecret(t *testing.T) {
	e := Event{
		Kind: KindRemoteAuthResponseSubmitted,
		RemoteAuthResponse: remoteauth.Response{
			Target:   "u@h",
			Kind:     "password",
			Password: "supersecret",
		},
	}
	s := e.RedactedSummary()
	if strings.Contains(s, "supersecret") {
		t.Fatalf("summary leaked password: %q", s)
	}
	if !strings.Contains(s, "u@h") || !strings.Contains(s, "password") {
		t.Fatalf("expected target and kind in summary: %q", s)
	}
}
