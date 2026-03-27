package remote

import (
	"testing"

	"delve-shell/internal/config"
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

	opts, handled := remoteSlashOptionsProvider("/remote", "en")
	if !handled {
		t.Fatal("expected /remote to be handled")
	}
	if len(opts) < 4 {
		t.Fatalf("expected at least 4 options, got %d", len(opts))
	}
	if opts[0].Cmd != "/remote on prod" || opts[1].Cmd != "/remote on db-bastion" {
		t.Fatalf("expected host options first, got %#v", opts[:2])
	}
	if opts[len(opts)-2].Cmd != "/remote on" || opts[len(opts)-1].Cmd != "/remote off" {
		t.Fatalf("expected action options last, got %#v", opts[len(opts)-2:])
	}
}

func TestRemoteSlashOptions_ProviderListsAllRemotesThenActions(t *testing.T) {
	t.Setenv("DELVE_SHELL_ROOT", t.TempDir())
	if err := config.EnsureRootDir(); err != nil {
		t.Fatal(err)
	}

	opts, handled := remoteSlashOptionsProvider("/remote zzz", "en")
	if !handled {
		t.Fatal("expected /remote zzz to be handled")
	}
	if len(opts) != 2 {
		t.Fatalf("no remotes: want on+off only, got %d", len(opts))
	}

	if err := config.AddRemote("ops@prod", "Production", ""); err != nil {
		t.Fatal(err)
	}
	if err := config.AddRemote("ops@db-bastion", "DB", ""); err != nil {
		t.Fatal(err)
	}
	full, handled := remoteSlashOptionsProvider("/remote p", "en")
	if !handled || len(full) != 4 {
		t.Fatalf("want 2 hosts + on + off from provider, got %d %#v", len(full), full)
	}
}
