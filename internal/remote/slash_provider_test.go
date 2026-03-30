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
	if opts[len(opts)-3].Cmd != "/access New" {
		t.Fatalf("expected /access New before Local/Offline, got %#v", opts[len(opts)-3])
	}
	if opts[len(opts)-2].Cmd != "/access Local" {
		t.Fatalf("expected /access Local second-to-last, got %#v", opts[len(opts)-2])
	}
	if opts[len(opts)-1].Cmd != "/access Offline" {
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
