package controller

import (
	"testing"

	"delve-shell/internal/host/bus"
)

func TestHostEventHandlersCoverAllKinds(t *testing.T) {
	t.Helper()
	all := bus.AllKinds()
	seen := make(map[bus.Kind]bool, len(all))
	for _, k := range all {
		seen[k] = true
		if _, ok := hostEventHandlers[k]; !ok {
			t.Errorf("missing handler for kind %q", k)
		}
	}
	for k := range hostEventHandlers {
		if !seen[k] {
			t.Errorf("unexpected handler for kind %q (not in bus.AllKinds)", k)
		}
	}
}
