package hil

import (
	"strings"
	"testing"

	hiltypes "delve-shell/internal/hil/types"
)

func TestCommandAutoApproveHighlight_expansionRejectPinsVarSites(t *testing.T) {
	w := NewAllowlist(nil)
	// kubectl -n "$(...)" still rejects collection; pin expansion spans instead of full-line risk.
	cmd := `while read ns; do kubectl -n "$(echo x)" get pods; done`
	spans := w.CommandAutoApproveHighlight(cmd)
	if len(spans) == 1 && spans[0].Start == 0 && spans[0].End == len(cmd) && spans[0].Kind == hiltypes.AutoApproveHighlightRisk {
		t.Fatal("expected narrowed risk spans, got full-line risk")
	}
	var riskText []string
	for _, s := range spans {
		if s.Kind != hiltypes.AutoApproveHighlightRisk {
			continue
		}
		if s.Start < 0 || s.End > len(cmd) || s.Start >= s.End {
			t.Fatalf("bad span %+v", s)
		}
		riskText = append(riskText, cmd[s.Start:s.End])
	}
	joined := strings.Join(riskText, " ")
	if !strings.Contains(joined, "$(") {
		t.Fatalf("expected risk spans to cover command substitution; risk fragments: %q", joined)
	}
}

func TestCommandAutoApproveHighlight_kubectlQuotedOpaqueNamespaceNotFullLineRisk(t *testing.T) {
	w := NewAllowlist(nil)
	cmd := `kubectl -n "$ns" get pods --no-headers`
	spans := w.CommandAutoApproveHighlight(cmd)
	for _, s := range spans {
		if s.Kind == hiltypes.AutoApproveHighlightRisk && s.Start == 0 && s.End == len(cmd) {
			t.Fatalf("did not want full-line risk for quoted opaque -n; spans=%+v", spans)
		}
	}
}

func TestCommandAutoApproveHighlight_dynamicArgv0StillFullRisk(t *testing.T) {
	w := NewAllowlist(nil)
	cmd := `x=foo; $x get pods`
	spans := w.CommandAutoApproveHighlight(cmd)
	if len(spans) != 1 || spans[0].Start != 0 || spans[0].End != len(cmd) || spans[0].Kind != hiltypes.AutoApproveHighlightRisk {
		t.Fatalf("want single full-line risk for dynamic argv0, got %+v", spans)
	}
}
