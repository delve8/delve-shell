package enterflow

import (
	"testing"

	"delve-shell/internal/maininput"
	"delve-shell/internal/slashview"
)

func TestPlanAfterSlashDispatches_nonSlashPassesThrough(t *testing.T) {
	t.Helper()
	p := PlanAfterSlashDispatches("hello", -1, nil, nil, "none", "noremote")
	if p.Kind != maininput.MainEnterPassToSubmit {
		t.Fatalf("got kind %v want PassToSubmit", p.Kind)
	}
}

func TestPlanAfterSlashDispatches_unknownSlash(t *testing.T) {
	t.Helper()
	p := PlanAfterSlashDispatches("/zzzzunknown", 0, nil, []int{}, "sn", "dr")
	if p.Kind != maininput.MainEnterUnknownSlash {
		t.Fatalf("got kind %v want UnknownSlash", p.Kind)
	}
}

func TestPlanAfterSlashDispatches_resolveSelected(t *testing.T) {
	t.Helper()
	opts := []slashview.Option{{Cmd: "/skill x", Desc: "d"}}
	vis := []int{0}
	p := PlanAfterSlashDispatches("/skill", 0, opts, vis, "sn", "dr")
	if p.Kind != maininput.MainEnterResolveSelected || p.Chosen != "/skill x" {
		t.Fatalf("unexpected plan: %+v", p)
	}
}
