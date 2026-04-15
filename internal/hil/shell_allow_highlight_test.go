package hil

import (
	"strings"
	"testing"

	hiltypes "delve-shell/internal/hil/types"
	"delve-shell/internal/i18n"
)

func TestCommandAutoApproveHighlight_expansionRejectPinsVarSites(t *testing.T) {
	w := NewAllowlist(nil)
	// Unquoted command substitution in a non-permissive argv slot still rejects collection; pin the
	// expansion span instead of falling back to full-line risk.
	cmd := `while read ns; do kubectl -n $(echo x) get pods; done`
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

func TestCommandAutoApproveHighlight_kubectlQuotedCmdSubstFlagValueNotFullLineRisk(t *testing.T) {
	w := NewAllowlist(nil)
	cmd := `kubectl -n "$(printf '%s' "$ns")" get pods`
	spans := w.CommandAutoApproveHighlight(cmd)
	for _, s := range spans {
		if s.Kind == hiltypes.AutoApproveHighlightRisk && s.Start == 0 && s.End == len(cmd) {
			t.Fatalf("did not want full-line risk for quoted cmdSubst flag value; spans=%+v", spans)
		}
	}
}

func TestCommandAutoApproveHighlight_openAnyFlagQuotedCmdSubstStillShowsRisk(t *testing.T) {
	w := NewAllowlist(nil)
	cmd := `crictl ps -a --name "$(printf '%s' "$p")" -q`
	spans := w.CommandAutoApproveHighlight(cmd)
	var hasRisk, hasSafe bool
	for _, s := range spans {
		switch s.Kind {
		case hiltypes.AutoApproveHighlightRisk:
			hasRisk = true
		case hiltypes.AutoApproveHighlightSafe:
			hasSafe = true
		}
	}
	if !hasRisk {
		t.Fatalf("expected risk for open-any flag with quoted cmdSubst, got %+v", spans)
	}
	if !hasSafe {
		t.Fatalf("expected inner read-only command to remain highlighted as safe, got %+v", spans)
	}
}

func TestCommandAutoApproveHighlight_dynamicArgv0StillFullRisk(t *testing.T) {
	w := NewAllowlist(nil)
	cmd := `x=foo; $x get pods`
	spans := w.CommandAutoApproveHighlight(cmd)
	if len(spans) != 1 || spans[0].Start != 0 || spans[0].End != len(cmd) || spans[0].Kind != hiltypes.AutoApproveHighlightRisk {
		t.Fatalf("want single full-line risk for dynamic argv0, got %+v", spans)
	}
	if spans[0].Reason == "" {
		t.Fatal("expected non-empty Risk Reason for full-line rejection")
	}
}

func TestCommandAutoApproveHighlight_xargsReasons(t *testing.T) {
	i18n.SetLang("en")
	w := NewAllowlist(nil)
	tests := []struct {
		name string
		cmd  string
		want string
	}{
		{"missing sentinel", `printf '%s\n' pod-a | xargs -r -n1 kubectl get pod`, i18n.T(i18n.KeyAutoApproveHLXargsMissingSentinel)},
		{"unsafe flag", `printf '%s\n' pod-a | xargs -P 4 kubectl get pod --`, i18n.T(i18n.KeyAutoApproveHLXargsUnsafeFlag)},
		{"unsafe target", `printf '%s\n' pod-a | xargs -r sh -c 'echo "$1"' --`, i18n.T(i18n.KeyAutoApproveHLXargsUnsafeTarget)},
		{"target mismatch", `printf '%s\n' delete pod-a | xargs -r -n1 kubectl --`, i18n.T(i18n.KeyAutoApproveHLXargsTargetMismatch)},
	}
	for _, tt := range tests {
		spans := w.CommandAutoApproveHighlight(tt.cmd)
		var found bool
		for _, s := range spans {
			if s.Kind == hiltypes.AutoApproveHighlightRisk && s.Reason == tt.want {
				found = true
				break
			}
		}
		if !found {
			t.Fatalf("%s: expected risk reason %q in spans=%+v", tt.name, tt.want, spans)
		}
	}
}

func TestCommandAutoApproveHighlight_xargsEchoSinkNotRisk(t *testing.T) {
	w := NewAllowlist(nil)
	cmd := `kubectl get pods -A --no-headers 2>/dev/null | wc -l | xargs echo "Total pods:"`
	spans := w.CommandAutoApproveHighlight(cmd)
	for _, s := range spans {
		if s.Kind != hiltypes.AutoApproveHighlightRisk {
			continue
		}
		got := cmd[s.Start:s.End]
		if strings.Contains(got, `xargs echo "Total pods:"`) {
			t.Fatalf("expected xargs echo sink to avoid risk highlight, spans=%+v", spans)
		}
	}
}
