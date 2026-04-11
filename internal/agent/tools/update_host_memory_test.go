package tools

import (
	"context"
	"strings"
	"testing"

	"delve-shell/internal/hil/types"
	"delve-shell/internal/hostmem"
)

func TestUpdateHostMemoryTool_NotifiesTranscript(t *testing.T) {
	t.Setenv("DELVE_SHELL_ROOT", t.TempDir())

	uiEvents := make(chan any, 1)
	tool := &UpdateHostMemoryTool{
		CurrentContext: func() hostmem.Context {
			return hostmem.Context{
				HostID:     "linux:machine-id:abc123",
				ProfileKey: "root",
			}
		},
		UIEvents: uiEvents,
	}

	out, err := tool.InvokableRun(context.Background(), `{
		"role":"k8s_control_plane",
		"role_confidence":0.88,
		"capabilities_add":["runs_kubernetes_control_plane"],
		"responsibilities_add":["cluster_administration"],
		"missing_commands_add":["jq","tar"],
		"evidence_add":["kubectl: command not found"]
	}`)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(out, "Role: k8s_control_plane") {
		t.Fatalf("summary = %q", out)
	}

	select {
	case raw := <-uiEvents:
		msg, ok := raw.(hiltypes.AgentNotify)
		if !ok {
			t.Fatalf("notify type = %T", raw)
		}
		if !strings.Contains(msg.Text, "Host memory updated:") {
			t.Fatalf("notify text = %q", msg.Text)
		}
		if !strings.Contains(msg.Text, "role=k8s_control_plane") {
			t.Fatalf("notify text = %q", msg.Text)
		}
		if !strings.Contains(msg.Text, "capabilities += runs_kubernetes_control_plane") {
			t.Fatalf("notify text = %q", msg.Text)
		}
		if !strings.Contains(msg.Text, "responsibilities += cluster_administration") {
			t.Fatalf("notify text = %q", msg.Text)
		}
		if !strings.Contains(msg.Text, "missing commands += jq, tar") {
			t.Fatalf("notify text = %q", msg.Text)
		}
		if !strings.Contains(msg.Text, "evidence += 1") {
			t.Fatalf("notify text = %q", msg.Text)
		}
	default:
		t.Fatal("expected transcript notification")
	}
}
