package remote

import (
	"strings"
	"testing"

	"delve-shell/internal/ui"
)

func TestBuildRemoteOverlayContent_ShowsOverwriteChoices(t *testing.T) {
	m := ui.NewModel(nil, nil)
	state := getRemoteOverlayState()
	state.AddRemote.Active = true
	state.AddRemote.Error = "remote target already exists: root@example.com"
	state.AddRemote.OfferOverwrite = true
	state.AddRemote.ChoiceIndex = 1
	setRemoteOverlayState(state)
	t.Cleanup(resetRemoteOverlayState)

	content, handled := buildRemoteOverlayContent(m)
	if !handled {
		t.Fatal("expected add-remote overlay content to be handled")
	}
	if !strings.Contains(content, "Overwrite saved remote") {
		t.Fatalf("expected overwrite choice in content, got %q", content)
	}
	if !strings.Contains(content, "Keep editing") {
		t.Fatalf("expected keep-editing choice in content, got %q", content)
	}
}
