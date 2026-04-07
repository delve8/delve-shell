package hil

import (
	"testing"

	"delve-shell/internal/config"
)

func TestAllowlist_DefaultKubectlGlobalOptsBeforeSubcommand(t *testing.T) {
	w := NewAllowlist(config.DefaultLoadedAllowlist())

	t.Run("namespace before subcommand", func(t *testing.T) {
		cmd := "kubectl -n cpaas-system get pod foo -o wide"
		if !w.AllowStrict(cmd) {
			t.Fatalf("AllowStrict(%q) want true", cmd)
		}
	})
	t.Run("namespace equals form", func(t *testing.T) {
		cmd := "kubectl -n=cpaas-system get pod foo"
		if !w.AllowStrict(cmd) {
			t.Fatalf("AllowStrict(%q) want true", cmd)
		}
	})
	t.Run("logs with namespace before subcommand", func(t *testing.T) {
		cmd := "kubectl -n kserve logs deploy/x --previous --tail=80"
		if !w.AllowStrict(cmd) {
			t.Fatalf("AllowStrict(%q) want true", cmd)
		}
	})
	t.Run("long flags", func(t *testing.T) {
		cmd := "kubectl --namespace foo --context bar get pods"
		if !w.AllowStrict(cmd) {
			t.Fatalf("AllowStrict(%q) want true", cmd)
		}
	})
	t.Run("all namespaces", func(t *testing.T) {
		cmd := "kubectl -A get pods"
		if !w.AllowStrict(cmd) {
			t.Fatalf("AllowStrict(%q) want true", cmd)
		}
	})
	t.Run("still allows kubectl get without leading globals", func(t *testing.T) {
		cmd := "kubectl get pods -n default"
		if !w.AllowStrict(cmd) {
			t.Fatalf("AllowStrict(%q) want true", cmd)
		}
	})
	t.Run("reject arbitrary token between kubectl and get", func(t *testing.T) {
		cmd := "kubectl foo get pods"
		if w.AllowStrict(cmd) {
			t.Fatalf("AllowStrict(%q) want false", cmd)
		}
	})
	t.Run("reject unknown global before subcommand", func(t *testing.T) {
		cmd := "kubectl --request-timeout=1s get pods"
		if w.AllowStrict(cmd) {
			t.Fatalf("AllowStrict(%q) want false", cmd)
		}
	})
	t.Run("cluster-info dump not allowlisted", func(t *testing.T) {
		cmd := "kubectl cluster-info dump"
		if w.AllowStrict(cmd) {
			t.Fatalf("AllowStrict(%q) want false", cmd)
		}
	})
	t.Run("cluster-info with dump not first tail token", func(t *testing.T) {
		cmd := "kubectl cluster-info foo dump"
		if w.AllowStrict(cmd) {
			t.Fatalf("AllowStrict(%q) want false", cmd)
		}
	})
	t.Run("cluster-info alone allowlisted", func(t *testing.T) {
		cmd := "kubectl cluster-info"
		if !w.AllowStrict(cmd) {
			t.Fatalf("AllowStrict(%q) want true", cmd)
		}
	})
	t.Run("compound cluster-info get nodes echo field-selector", func(t *testing.T) {
		cmd := "kubectl cluster-info && echo '---' && kubectl get nodes -o wide && echo '---' && kubectl get pods -A --field-selector=status.phase!=Running,status.phase!=Succeeded -o wide"
		if ContainsWriteRedirection(cmd) {
			t.Fatal("unexpected write-redirection heuristic on field-selector !=")
		}
		if !w.AllowStrict(cmd) {
			t.Fatalf("AllowStrict(%q) want true", cmd)
		}
	})
}
