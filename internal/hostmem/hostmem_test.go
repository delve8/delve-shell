package hostmem

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"strings"
	"testing"
	"time"
)

type fakeExecutor struct{}

func (fakeExecutor) Run(ctx context.Context, command string) (string, string, int, error) {
	_ = ctx
	if strings.Contains(command, "machine_id") {
		return strings.Join([]string{
			"os=Linux",
			"user=ops",
			"shell=/bin/bash",
			"machine_id=abc123",
			"pkg=apt",
			"have=kubectl",
		}, "\n"), "", 0, nil
	}
	return "kubectl\njq\njq\nls\nbasename\n/bin/ignored\nbad cmd\n", "", 0, nil
}

func TestProbeAndApplyStrongMachineID(t *testing.T) {
	t.Setenv("DELVE_SHELL_ROOT", t.TempDir())

	pr, err := Probe(context.Background(), fakeExecutor{}, "ops@node1")
	if err != nil {
		t.Fatal(err)
	}
	if pr.Context.HostID != "linux:machine-id:abc123" {
		t.Fatalf("HostID = %q", pr.Context.HostID)
	}
	if pr.Context.WeakIdentity {
		t.Fatal("expected strong machine identity")
	}
	if pr.Context.ProfileKey != "ops" {
		t.Fatalf("ProfileKey = %q", pr.Context.ProfileKey)
	}
	ctx, err := ApplyProbe(pr)
	if err != nil {
		t.Fatal(err)
	}
	got, err := Load(ctx.HostID)
	if err != nil {
		t.Fatal(err)
	}
	if got == nil {
		t.Fatal("expected saved memory")
	}
	profile := got.Profiles["ops"]
	if profile == nil {
		t.Fatal("expected ops profile")
	}
	if !contains(profile.Available, "jq") || !contains(profile.Available, "kubectl") {
		t.Fatalf("available commands = %#v", profile.Available)
	}
	if contains(profile.Available, "basename") {
		t.Fatalf("persisted available commands should be curated, got %#v", profile.Available)
	}
	if !contains(profile.Missing, "tar") {
		t.Fatalf("expected important missing command snapshot, got %#v", profile.Missing)
	}
	if got.Machine.OSFamily != "Linux" {
		t.Fatalf("OSFamily = %q", got.Machine.OSFamily)
	}
}

func TestFileNameUsesOSPrefixAndShortHash(t *testing.T) {
	sum := sha256.Sum256([]byte("linux:machine-id:abc123"))
	got := fileNameFor("linux:machine-id:abc123", sum[:16])
	want := "linux-" + hex.EncodeToString(sum[:16]) + ".json"
	if got != want {
		t.Fatalf("fileNameFor() = %q, want %q", got, want)
	}
}

func TestUpdateAndProbeRefresh(t *testing.T) {
	t.Setenv("DELVE_SHELL_ROOT", t.TempDir())
	ctx := Context{HostID: "linux:machine-id:abc123", ProfileKey: "root", Alias: "root@node1"}
	initial := ProbeResult{
		Context:         ctx,
		IdentitySource:  "/etc/machine-id",
		OSFamily:        "Linux",
		User:            "root",
		Shell:           "/bin/bash",
		Available:       []string{"bash", "kubectl", "jq"},
		PackageManagers: []string{"apt"},
		ObservedAt:      time.Now().UTC(),
	}
	if _, err := ApplyProbe(initial); err != nil {
		t.Fatal(err)
	}
	if _, err := Update(ctx, UpdatePatch{
		Role:           "k8s_control_plane",
		RoleConfidence: 0.88,
		TagsAdd:        []string{"k8s", "control-plane"},
		EvidenceAdd:    []string{"/etc/kubernetes/manifests/kube-apiserver.yaml exists"},
		MissingAdd:     []string{"yq"},
	}); err != nil {
		t.Fatal(err)
	}
	mem, err := Load(ctx.HostID)
	if err != nil {
		t.Fatal(err)
	}
	if mem.Machine.Role != "k8s_control_plane" {
		t.Fatalf("role = %q", mem.Machine.Role)
	}
	if !contains(mem.Profiles["root"].Missing, "yq") {
		t.Fatalf("missing = %#v", mem.Profiles["root"].Missing)
	}

	refreshed := initial
	refreshed.Available = []string{"bash", "yq", "tar"}
	if _, err := ApplyProbe(refreshed); err != nil {
		t.Fatal(err)
	}
	mem, err = Load(ctx.HostID)
	if err != nil {
		t.Fatal(err)
	}
	profile := mem.Profiles["root"]
	if contains(profile.Available, "kubectl") {
		t.Fatalf("probe should refresh available snapshot, got %#v", profile.Available)
	}
	if contains(profile.Missing, "yq") {
		t.Fatalf("available command should be removed from missing, got %#v", profile.Missing)
	}
	if contains(profile.Missing, "tar") {
		t.Fatalf("newly available command should be removed from missing, got %#v", profile.Missing)
	}
	summary := Summary(mem, "root")
	if !strings.Contains(summary, "Role: k8s_control_plane") || !strings.Contains(summary, "Available commands: bash, tar, yq") {
		t.Fatalf("summary = %q", summary)
	}
}

func contains(items []string, want string) bool {
	for _, item := range items {
		if item == want {
			return true
		}
	}
	return false
}
