package config

import (
	"os"
	"path/filepath"
	"testing"
)

func writeTestSSHConfig(t *testing.T, home string, content string) {
	t.Helper()
	sshDir := filepath.Join(home, ".ssh")
	if err := os.MkdirAll(sshDir, 0700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(sshDir, "config"), []byte(content), 0600); err != nil {
		t.Fatal(err)
	}
}

func TestLoadSSHConfigHosts_ExplicitAliasesOnly(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	t.Setenv("USER", "localuser")
	writeTestSSHConfig(t, home, `
Host prod
  HostName prod.example.com
  User ops
  Port 2222
  IdentityFile ~/.ssh/prod_key

Host *.internal
  User ignored

Host db staging
  HostName %h.example.com
  User deploy
`)

	hosts, err := LoadSSHConfigHosts()
	if err != nil {
		t.Fatal(err)
	}
	if len(hosts) != 3 {
		t.Fatalf("len(hosts)=%d want 3: %#v", len(hosts), hosts)
	}
	if hosts[0].Alias != "prod" || hosts[0].Target != "ops@prod.example.com:2222" {
		t.Fatalf("prod entry=%#v", hosts[0])
	}
	if hosts[0].IdentityFile != filepath.Join(home, ".ssh", "prod_key") {
		t.Fatalf("identity=%q", hosts[0].IdentityFile)
	}
	if hosts[1].Alias != "db" || hosts[1].Target != "deploy@db.example.com" {
		t.Fatalf("db entry=%#v", hosts[1])
	}
	if hosts[2].Alias != "staging" || hosts[2].Target != "deploy@staging.example.com" {
		t.Fatalf("staging entry=%#v", hosts[2])
	}
}

func TestResolveSSHConfigHost_DefaultsUserHostAndPort(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	t.Setenv("USER", "localuser")
	writeTestSSHConfig(t, home, `
Host dev
  IdentityFile ~/.ssh/%r-%h
`)

	host, ok, err := ResolveSSHConfigHost("DEV")
	if err != nil {
		t.Fatal(err)
	}
	if !ok {
		t.Fatal("expected dev to resolve")
	}
	if host.Target != "localuser@dev" {
		t.Fatalf("target=%q want localuser@dev", host.Target)
	}
	if host.IdentityFile != filepath.Join(home, ".ssh", "localuser-dev") {
		t.Fatalf("identity=%q", host.IdentityFile)
	}
}

func TestResolveSSHConfigHost_MatchesHostName(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	t.Setenv("USER", "localuser")
	writeTestSSHConfig(t, home, `
Host jump
  HostName jump.example.com
  User ops
  Port 2201
`)

	host, ok, err := ResolveSSHConfigHost("jump.example.com")
	if err != nil {
		t.Fatal(err)
	}
	if !ok {
		t.Fatal("expected jump.example.com to resolve")
	}
	if host.Alias != "jump" || host.Target != "ops@jump.example.com:2201" {
		t.Fatalf("host=%#v", host)
	}
}

func TestLoadSSHConfigHosts_MissingFileIsEmpty(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)

	hosts, err := LoadSSHConfigHosts()
	if err != nil {
		t.Fatal(err)
	}
	if len(hosts) != 0 {
		t.Fatalf("hosts=%#v want empty", hosts)
	}
}
