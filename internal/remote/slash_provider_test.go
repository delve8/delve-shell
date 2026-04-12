package remote

import (
	"testing"

	"delve-shell/internal/config"
	"delve-shell/internal/slash/access"
	"github.com/charmbracelet/bubbles/textinput"
)

func TestRemoteSlashOptions_RootOrdersHostsThenActions(t *testing.T) {
	t.Setenv("DELVE_SHELL_ROOT", t.TempDir())
	if err := config.EnsureRootDir(); err != nil {
		t.Fatal(err)
	}
	if err := config.AddRemote("ops@prod", "Production", ""); err != nil {
		t.Fatal(err)
	}
	if err := config.AddRemote("ops@db-bastion", "DB Bastion", ""); err != nil {
		t.Fatal(err)
	}

	opts, handled := remoteSlashOptionsProvider("/access", "en")
	if !handled {
		t.Fatal("expected /access to be handled")
	}
	if len(opts) < 5 {
		t.Fatalf("expected at least 5 options, got %d", len(opts))
	}
	if opts[0].Cmd != "/access prod" || opts[1].Cmd != "/access db-bastion" {
		t.Fatalf("expected host options first, got %#v", opts[:2])
	}
	if opts[len(opts)-3].Cmd != slashaccess.Command(slashaccess.ReservedNew) {
		t.Fatalf("expected /access New before Local/Offline, got %#v", opts[len(opts)-3])
	}
	if opts[len(opts)-2].Cmd != slashaccess.Command(slashaccess.ReservedLocal) {
		t.Fatalf("expected /access Local second-to-last, got %#v", opts[len(opts)-2])
	}
	if opts[len(opts)-1].Cmd != slashaccess.Command(slashaccess.ReservedOffline) {
		t.Fatalf("expected /access Offline last, got %#v", opts[len(opts)-1])
	}
}

func TestRemoteSlashOptions_ProviderListsAllRemotesThenActions(t *testing.T) {
	t.Setenv("DELVE_SHELL_ROOT", t.TempDir())
	if err := config.EnsureRootDir(); err != nil {
		t.Fatal(err)
	}

	opts, handled := remoteSlashOptionsProvider("/access zzz", "en")
	if !handled {
		t.Fatal("expected /access zzz to be handled")
	}
	if len(opts) != 3 {
		t.Fatalf("no remotes: want New+Local+Offline, got %d", len(opts))
	}

	if err := config.AddRemote("ops@prod", "Production", ""); err != nil {
		t.Fatal(err)
	}
	if err := config.AddRemote("ops@db-bastion", "DB", ""); err != nil {
		t.Fatal(err)
	}
	full, handled := remoteSlashOptionsProvider("/access p", "en")
	if !handled || len(full) != 5 {
		t.Fatalf("want 2 hosts + New + Local + Offline from provider, got %d %#v", len(full), full)
	}
}

func TestResolveConnectTarget_ConfigMatchPrefersSavedRemote(t *testing.T) {
	t.Setenv("DELVE_SHELL_ROOT", t.TempDir())
	if err := config.EnsureRootDir(); err != nil {
		t.Fatal(err)
	}
	if err := config.AddRemote("ops@prod", "Production", "~/.ssh/prod"); err != nil {
		t.Fatal(err)
	}
	if got := resolveConnectTarget("prod"); got != "ops@prod" {
		t.Fatalf("resolveConnectTarget(prod)=%q want %q", got, "ops@prod")
	}
}

func TestPrefillAddRemoteFromParams_ConfigMatchPrefillsSavedRemote(t *testing.T) {
	t.Setenv("DELVE_SHELL_ROOT", t.TempDir())
	if err := config.EnsureRootDir(); err != nil {
		t.Fatal(err)
	}
	if err := config.AddRemote("ops@prod", "Production", "~/.ssh/prod"); err != nil {
		t.Fatal(err)
	}
	state := AddRemoteOverlayState{
		HostInput: textinput.New(),
		UserInput: textinput.New(),
		KeyInput:  textinput.New(),
		NameInput: textinput.New(),
	}
	prefillAddRemoteFromParams(&state, map[string]string{"target": "prod"})
	if state.UserInput.Value() != "ops" || state.HostInput.Value() != "prod" {
		t.Fatalf("prefill user/host = %q/%q", state.UserInput.Value(), state.HostInput.Value())
	}
	if state.KeyInput.Value() != "~/.ssh/prod" {
		t.Fatalf("key=%q want ~/.ssh/prod", state.KeyInput.Value())
	}
}

func TestPrefillAddRemoteFromParams_HostOnlyRequiresUsername(t *testing.T) {
	state := AddRemoteOverlayState{
		HostInput: textinput.New(),
		UserInput: textinput.New(),
		KeyInput:  textinput.New(),
		NameInput: textinput.New(),
	}
	prefillAddRemoteFromParams(&state, map[string]string{"target": "prod"})
	if state.HostInput.Value() != "prod" {
		t.Fatalf("host=%q want prod", state.HostInput.Value())
	}
	if state.FieldIndex != 1 {
		t.Fatalf("fieldIndex=%d want 1", state.FieldIndex)
	}
}
