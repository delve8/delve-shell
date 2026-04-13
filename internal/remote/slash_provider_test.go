package remote

import (
	"os"
	"path/filepath"
	"testing"

	"delve-shell/internal/config"
	"delve-shell/internal/slash/access"
	"github.com/charmbracelet/bubbles/textinput"
)

func writeRemoteTestSSHConfig(t *testing.T, home string, content string) {
	t.Helper()
	sshDir := filepath.Join(home, ".ssh")
	if err := os.MkdirAll(sshDir, 0700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(sshDir, "config"), []byte(content), 0600); err != nil {
		t.Fatal(err)
	}
}

func TestRemoteSlashOptions_RootOrdersHostsThenActions(t *testing.T) {
	t.Setenv("DELVE_SHELL_ROOT", t.TempDir())
	t.Setenv("HOME", t.TempDir())
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

func TestRemoteSlashOptions_ListsSSHConfigAliasesAfterSavedRemotes(t *testing.T) {
	t.Setenv("DELVE_SHELL_ROOT", t.TempDir())
	home := t.TempDir()
	t.Setenv("HOME", home)
	t.Setenv("USER", "localuser")
	if err := config.EnsureRootDir(); err != nil {
		t.Fatal(err)
	}
	if err := config.AddRemote("ops@prod", "Production", ""); err != nil {
		t.Fatal(err)
	}
	writeRemoteTestSSHConfig(t, home, `
Host jump
  HostName jump.example.com
  User ops
`)

	opts, handled := remoteSlashOptionsProvider("/access", "en")
	if !handled {
		t.Fatal("expected /access to be handled")
	}
	if len(opts) < 5 {
		t.Fatalf("expected host rows plus actions, got %#v", opts)
	}
	if opts[0].Cmd != "/access prod" {
		t.Fatalf("first row=%#v want saved remote first", opts[0])
	}
	if opts[1].Cmd != "/access jump.example.com" {
		t.Fatalf("second row=%#v want ssh config host second", opts[1])
	}
	if opts[1].Desc != "jump (from ~/.ssh/config)" {
		t.Fatalf("ssh config desc=%q want jump (from ~/.ssh/config)", opts[1].Desc)
	}
	if opts[1].FillValue != "/access jump.example.com" {
		t.Fatalf("fill=%q want /access jump.example.com", opts[1].FillValue)
	}
}

func TestRemoteSlashOptions_ProviderListsAllRemotesThenActions(t *testing.T) {
	t.Setenv("DELVE_SHELL_ROOT", t.TempDir())
	t.Setenv("HOME", t.TempDir())
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

func TestResolveConnectTarget_SSHConfigAlias(t *testing.T) {
	t.Setenv("DELVE_SHELL_ROOT", t.TempDir())
	home := t.TempDir()
	t.Setenv("HOME", home)
	t.Setenv("USER", "localuser")
	writeRemoteTestSSHConfig(t, home, `
Host jump
  HostName jump.example.com
  User ops
  Port 2201
`)
	if got := resolveConnectTarget("jump"); got != "ops@jump.example.com:2201" {
		t.Fatalf("resolveConnectTarget(jump)=%q", got)
	}
	if got := resolveConnectTarget("jump.example.com"); got != "ops@jump.example.com:2201" {
		t.Fatalf("resolveConnectTarget(jump.example.com)=%q", got)
	}
}

func TestPrefillAddRemoteFromParams_ConfigMatchPrefillsSavedRemote(t *testing.T) {
	t.Setenv("DELVE_SHELL_ROOT", t.TempDir())
	t.Setenv("HOME", t.TempDir())
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

func TestPrefillAddRemoteFromParams_SSHConfigMatchPrefillsAlias(t *testing.T) {
	t.Setenv("DELVE_SHELL_ROOT", t.TempDir())
	home := t.TempDir()
	t.Setenv("HOME", home)
	t.Setenv("USER", "localuser")
	writeRemoteTestSSHConfig(t, home, `
Host jump
  HostName jump.example.com
  User ops
  IdentityFile ~/.ssh/jump_key
`)
	state := AddRemoteOverlayState{
		HostInput: textinput.New(),
		UserInput: textinput.New(),
		KeyInput:  textinput.New(),
		NameInput: textinput.New(),
	}
	prefillAddRemoteFromParams(&state, map[string]string{"target": "jump"})
	if state.UserInput.Value() != "ops" || state.HostInput.Value() != "jump.example.com" {
		t.Fatalf("prefill user/host = %q/%q", state.UserInput.Value(), state.HostInput.Value())
	}
	if state.KeyInput.Value() != filepath.Join(home, ".ssh", "jump_key") {
		t.Fatalf("key=%q", state.KeyInput.Value())
	}
	if state.NameInput.Value() != "jump" {
		t.Fatalf("name=%q want jump", state.NameInput.Value())
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
