package hil

import (
	"sort"
	"strings"

	hiltypes "delve-shell/internal/hil/types"
)

// CommandAutoApproveHighlight returns non-overlapping half-open byte spans covering command for approval UI.
// Safe segments match the same per-segment checks as [Allowlist.CommandAllowsAutoApprove] (without deduplication);
// Risk marks segments that fail those checks; Neutral fills gaps (separators). On parse failure, full-command
// rejection, or top-level write redirection, the whole command is one Risk span.
// When the tree is rejected and there is at least one disallowed expansion under a non-permissive argv0, those
// expansion spans plus related read/for variable names are Risk and the rest is Neutral; otherwise full-line Risk.
func (w *Allowlist) CommandAutoApproveHighlight(command string) []hiltypes.AutoApproveHighlightSpan {
	command = strings.TrimSpace(command)
	if command == "" || w == nil {
		return nil
	}
	n := len(command)
	riskAll := func() []hiltypes.AutoApproveHighlightSpan {
		return []hiltypes.AutoApproveHighlightSpan{{Start: 0, End: n, Kind: hiltypes.AutoApproveHighlightRisk}}
	}
	if ContainsWriteRedirection(command) {
		return riskAll()
	}
	f, err := parseShell(command)
	if err != nil {
		return riskAll()
	}
	varArg := func(name string) bool { return w.argv0PermitsVarArgs(name) }
	locals := localFunctionNames(f)
	_, ranges, reject := collectAllowlistSegments(f, command, locals, shellUnwrapMax, varArg)
	if reject {
		if pin := expansionPolicyRiskSpans(command, f, locals, varArg); len(pin) > 0 {
			raw := make([]hiltypes.AutoApproveHighlightSpan, 0, len(pin))
			for _, rg := range pin {
				raw = append(raw, hiltypes.AutoApproveHighlightSpan{Start: rg.start, End: rg.end, Kind: hiltypes.AutoApproveHighlightRisk})
			}
			return flattenAutoApproveHighlight(n, raw)
		}
		return riskAll()
	}
	if len(ranges) == 0 {
		return []hiltypes.AutoApproveHighlightSpan{{Start: 0, End: n, Kind: hiltypes.AutoApproveHighlightNeutral}}
	}
	var raw []hiltypes.AutoApproveHighlightSpan
	for _, rg := range ranges {
		if rg.start < 0 || rg.end > n || rg.start > rg.end {
			continue
		}
		t := strings.TrimSpace(command[rg.start:rg.end])
		if t == "" {
			continue
		}
		kind := hiltypes.AutoApproveHighlightSafe
		if ContainsWriteRedirection(t) || !w.segmentAllowed(t) {
			kind = hiltypes.AutoApproveHighlightRisk
		}
		raw = append(raw, hiltypes.AutoApproveHighlightSpan{Start: rg.start, End: rg.end, Kind: kind})
	}
	return flattenAutoApproveHighlight(n, raw)
}

// flattenAutoApproveHighlight splits [0,n) using span boundaries; when intervals overlap, the narrowest containing span wins (so inner $(cmd) can differ from outer).
func flattenAutoApproveHighlight(n int, classified []hiltypes.AutoApproveHighlightSpan) []hiltypes.AutoApproveHighlightSpan {
	pts := map[int]struct{}{0: {}, n: {}}
	for _, s := range classified {
		if s.Start >= s.End {
			continue
		}
		pts[s.Start] = struct{}{}
		pts[s.End] = struct{}{}
	}
	keys := make([]int, 0, len(pts))
	for k := range pts {
		keys = append(keys, k)
	}
	sort.Ints(keys)
	var out []hiltypes.AutoApproveHighlightSpan
	for i := 0; i < len(keys)-1; i++ {
		a, b := keys[i], keys[i+1]
		if a >= b || a < 0 || b > n {
			continue
		}
		bestW := n + 1
		var bestKind hiltypes.AutoApproveHighlightKind
		var found bool
		for _, s := range classified {
			if s.Start > a || s.End < b {
				continue
			}
			w := s.End - s.Start
			if w < bestW {
				bestW = w
				bestKind = s.Kind
				found = true
			}
		}
		kind := hiltypes.AutoApproveHighlightNeutral
		if found {
			kind = bestKind
		}
		if len(out) > 0 && out[len(out)-1].Kind == kind && out[len(out)-1].End == a {
			out[len(out)-1].End = b
		} else {
			out = append(out, hiltypes.AutoApproveHighlightSpan{Start: a, End: b, Kind: kind})
		}
	}
	return out
}
