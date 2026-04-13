package hostmem

import (
	"context"
	"path/filepath"
	"testing"
	"time"

	"delve-shell/internal/config"
	"delve-shell/internal/history"
)

type fakeBackgroundAnalyzer struct {
	patch UpdatePatch
	calls chan BackgroundAnalyzeInput
}

func (f fakeBackgroundAnalyzer) Analyze(_ context.Context, input BackgroundAnalyzeInput) (UpdatePatch, error) {
	if f.calls != nil {
		f.calls <- input
	}
	return f.patch, nil
}

func TestBackgroundUpdaterAppliesPatchFromRecentSession(t *testing.T) {
	t.Setenv("DELVE_SHELL_ROOT", t.TempDir())
	if err := config.EnsureRootDir(); err != nil {
		t.Fatalf("EnsureRootDir error: %v", err)
	}

	s := &history.Session{}
	sessionPath := filepath.Join(config.HistoryDir(), "bg.jsonl")
	s = mustOpenSessionForTest(t, sessionPath)
	defer s.Close()
	if err := s.AppendUserInput("检查这台机器"); err != nil {
		t.Fatalf("AppendUserInput error: %v", err)
	}
	if err := s.AppendCommandResult("kubectl version --client", "Client Version: v1.30.0", "", 0); err != nil {
		t.Fatalf("AppendCommandResult error: %v", err)
	}

	calls := make(chan BackgroundAnalyzeInput, 1)
	stop := make(chan struct{})
	updater := NewBackgroundUpdater(BackgroundUpdaterOptions{
		Analyzer: fakeBackgroundAnalyzer{
			patch: UpdatePatch{
				Role:                "k8s_control_plane",
				RoleConfidence:      0.91,
				AvailableAdd:        []string{"kubectl"},
				CapabilitiesAdd:     []string{"hosts_kubectl"},
				ResponsibilitiesAdd: []string{"cluster_administration"},
			},
			calls: calls,
		},
		Stop:      stop,
		Debounce:  10 * time.Millisecond,
		MaxEvents: 8,
	})
	defer close(stop)

	memCtx := Context{HostID: "linux:machine-id:test", ProfileKey: "root", Alias: "local"}
	updater.Enqueue(sessionPath, memCtx, history.Event{Type: history.EventTypeCommandResult})

	select {
	case input := <-calls:
		if len(input.Events) == 0 {
			t.Fatal("expected recent events")
		}
	case <-time.After(2 * time.Second):
		t.Fatal("timed out waiting for analyzer")
	}

	mem := waitForMemory(t, memCtx.HostID)
	if mem.Machine.Role != "k8s_control_plane" {
		t.Fatalf("role=%q", mem.Machine.Role)
	}
	profile := mem.Profiles[memCtx.ProfileKey]
	if profile == nil {
		t.Fatal("expected profile")
	}
	if len(profile.Available) != 1 || profile.Available[0] != "kubectl" {
		t.Fatalf("available=%v", profile.Available)
	}
}

func TestBackgroundUpdaterIgnoresIrrelevantEvents(t *testing.T) {
	t.Setenv("DELVE_SHELL_ROOT", t.TempDir())

	calls := make(chan BackgroundAnalyzeInput, 1)
	stop := make(chan struct{})
	updater := NewBackgroundUpdater(BackgroundUpdaterOptions{
		Analyzer: fakeBackgroundAnalyzer{
			patch: UpdatePatch{Role: "bastion"},
			calls: calls,
		},
		Stop:     stop,
		Debounce: 10 * time.Millisecond,
	})
	defer close(stop)

	updater.Enqueue("/tmp/nope.jsonl", Context{HostID: "linux:machine-id:test", ProfileKey: "root"}, history.Event{Type: history.EventTypeUserInput})

	select {
	case <-calls:
		t.Fatal("unexpected analyzer call for irrelevant event")
	case <-time.After(80 * time.Millisecond):
	}
}

func mustOpenSessionForTest(t *testing.T, path string) *history.Session {
	t.Helper()
	s, err := history.OpenSession(path)
	if err != nil {
		t.Fatalf("OpenSession error: %v", err)
	}
	return s
}

func waitForMemory(t *testing.T, hostID string) *Memory {
	t.Helper()
	deadline := time.Now().Add(2 * time.Second)
	for time.Now().Before(deadline) {
		mem, err := Load(hostID)
		if err != nil {
			t.Fatalf("Load error: %v", err)
		}
		if mem != nil {
			return mem
		}
		time.Sleep(10 * time.Millisecond)
	}
	t.Fatal("expected saved memory")
	return nil
}
