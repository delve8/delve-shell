package remote

import (
	"testing"

	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"

	hostcmd "delve-shell/internal/host/cmd"
	"delve-shell/internal/pathcomplete"
	remoteauth "delve-shell/internal/remote/auth"
	"delve-shell/internal/ui"
)

func TestHandleRemoteAuthOverlayKey_HostKeyDownEnterRejects(t *testing.T) {
	ch := make(chan hostcmd.Command, 1)
	m := ui.NewModel(nil, nil)
	m.CommandSender = ui.NewCommandChannelSender(ch)
	state := getRemoteOverlayState()
	state.RemoteAuth.Step = AuthStepHostKey
	state.RemoteAuth.Target = "root@example.com"
	state.RemoteAuth.ChoiceIndex = 0
	setRemoteOverlayState(state)
	t.Cleanup(resetRemoteOverlayState)

	_, _, handled := handleRemoteAuthOverlayKey(m, tea.KeyDown.String(), tea.KeyMsg{Type: tea.KeyDown})
	if !handled {
		t.Fatal("expected down to be handled")
	}
	got := getRemoteOverlayState()
	if got.RemoteAuth.ChoiceIndex != 1 {
		t.Fatalf("choice index=%d want 1", got.RemoteAuth.ChoiceIndex)
	}

	_, _, handled = handleRemoteAuthOverlayKey(m, tea.KeyEnter.String(), tea.KeyMsg{Type: tea.KeyEnter})
	if !handled {
		t.Fatal("expected enter to be handled")
	}
	cmd := <-ch
	reply, ok := cmd.(hostcmd.RemoteAuthReply)
	if !ok {
		t.Fatalf("command type=%T want hostcmd.RemoteAuthReply", cmd)
	}
	if reply.Response.Kind != remoteauth.ResponseKindHostKeyReject {
		t.Fatalf("response kind=%q want %q", reply.Response.Kind, remoteauth.ResponseKindHostKeyReject)
	}
}

func TestHandleRemoteAuthOverlayKey_HostKeyEnterAcceptsSelectedChoice(t *testing.T) {
	ch := make(chan hostcmd.Command, 1)
	m := ui.NewModel(nil, nil)
	m.CommandSender = ui.NewCommandChannelSender(ch)
	state := getRemoteOverlayState()
	state.RemoteAuth.Step = AuthStepHostKey
	state.RemoteAuth.Target = "root@example.com"
	setRemoteOverlayState(state)
	t.Cleanup(resetRemoteOverlayState)

	_, _, handled := handleRemoteAuthOverlayKey(m, tea.KeyEnter.String(), tea.KeyMsg{Type: tea.KeyEnter})
	if !handled {
		t.Fatal("expected enter to be handled")
	}
	got := getRemoteOverlayState()
	if !got.RemoteAuth.Connecting {
		t.Fatal("expected connecting state after accept")
	}
	cmd := <-ch
	reply, ok := cmd.(hostcmd.RemoteAuthReply)
	if !ok {
		t.Fatalf("command type=%T want hostcmd.RemoteAuthReply", cmd)
	}
	if reply.Response.Kind != remoteauth.ResponseKindHostKeyAccept {
		t.Fatalf("response kind=%q want %q", reply.Response.Kind, remoteauth.ResponseKindHostKeyAccept)
	}
}

func TestHandleRemoteAuthOverlayKey_ChooseDownEnterOpensIdentity(t *testing.T) {
	m := ui.NewModel(nil, nil)
	state := getRemoteOverlayState()
	state.RemoteAuth.Step = AuthStepChoose
	state.RemoteAuth.ChoiceIndex = 0
	setRemoteOverlayState(state)
	pathcomplete.ResetState()
	t.Cleanup(func() {
		resetRemoteOverlayState()
		pathcomplete.ResetState()
	})

	_, _, handled := handleRemoteAuthOverlayKey(m, tea.KeyDown.String(), tea.KeyMsg{Type: tea.KeyDown})
	if !handled {
		t.Fatal("expected down to be handled")
	}
	got := getRemoteOverlayState()
	if got.RemoteAuth.ChoiceIndex != 1 {
		t.Fatalf("choice index=%d want 1", got.RemoteAuth.ChoiceIndex)
	}

	_, _, handled = handleRemoteAuthOverlayKey(m, tea.KeyEnter.String(), tea.KeyMsg{Type: tea.KeyEnter})
	if !handled {
		t.Fatal("expected enter to be handled")
	}
	got = getRemoteOverlayState()
	if got.RemoteAuth.Step != AuthStepIdentity {
		t.Fatalf("step=%q want %q", got.RemoteAuth.Step, AuthStepIdentity)
	}
	if got.RemoteAuth.Input.Placeholder == "" {
		t.Fatal("expected identity input to be initialized")
	}
}

func TestHandleRemoteAuthOverlayKey_ChooseNumericShortcutStillWorks(t *testing.T) {
	m := ui.NewModel(nil, nil)
	state := getRemoteOverlayState()
	state.RemoteAuth.Step = AuthStepChoose
	setRemoteOverlayState(state)
	pathcomplete.ResetState()
	t.Cleanup(func() {
		resetRemoteOverlayState()
		pathcomplete.ResetState()
	})

	_, _, handled := handleRemoteAuthOverlayKey(m, "1", tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'1'}})
	if !handled {
		t.Fatal("expected numeric shortcut to be handled")
	}
	got := getRemoteOverlayState()
	if got.RemoteAuth.Step != AuthStepPassword {
		t.Fatalf("step=%q want %q", got.RemoteAuth.Step, AuthStepPassword)
	}
}

func TestHandleRemoteAuthOverlayKey_UsernameEnterRequiresExplicitUsername(t *testing.T) {
	m := ui.NewModel(nil, nil)
	state := getRemoteOverlayState()
	state.RemoteAuth.Step = AuthStepUsername
	state.RemoteAuth.UsernameInput = textinput.New()
	setRemoteOverlayState(state)
	t.Cleanup(resetRemoteOverlayState)

	_, _, handled := handleRemoteAuthOverlayKey(m, tea.KeyEnter.String(), tea.KeyMsg{Type: tea.KeyEnter})
	if !handled {
		t.Fatal("expected enter to be handled")
	}
	got := getRemoteOverlayState()
	if got.RemoteAuth.Step != AuthStepUsername {
		t.Fatalf("step=%q want %q", got.RemoteAuth.Step, AuthStepUsername)
	}
	if got.RemoteAuth.Error != "username is required" {
		t.Fatalf("error=%q want %q", got.RemoteAuth.Error, "username is required")
	}
}
