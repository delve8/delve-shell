package slashview

import (
	"strings"

	"delve-shell/internal/slash/access"
	slashskill "delve-shell/internal/slash/skill"
)

// parseAccessRest returns text after "/access " (input without leading slash). ok is false if input is not an /access command.
func parseAccessRest(input string) (rest string, ok bool) {
	input = strings.TrimSpace(strings.TrimPrefix(strings.TrimSpace(input), "/"))
	if input == "" {
		return "", false
	}
	parts := strings.SplitN(input, " ", 2)
	if len(parts) == 0 || !strings.EqualFold(parts[0], "access") {
		return "", false
	}
	if len(parts) == 1 {
		return "", true
	}
	return strings.TrimSpace(parts[1]), true
}

// parseSkillRest returns text after "/skill " (input without leading slash). ok is false if input is not a /skill command.
func parseSkillRest(input string) (rest string, ok bool) {
	input = strings.TrimSpace(strings.TrimPrefix(strings.TrimSpace(input), "/"))
	if input == "" {
		return "", false
	}
	parts := strings.SplitN(input, " ", 2)
	if len(parts) == 0 || !strings.EqualFold(parts[0], slashskill.Subcommand) {
		return "", false
	}
	if len(parts) == 1 {
		return "", true
	}
	return strings.TrimSpace(parts[1]), true
}

func reservedRowMatch(rest, displayToken, lowerToken string) bool {
	if rest == "" {
		return true
	}
	if rest == displayToken {
		return true
	}
	restLower := strings.ToLower(rest)
	restAllLower := rest == restLower
	if restAllLower {
		if restLower == lowerToken || strings.HasPrefix(lowerToken, restLower) {
			return true
		}
	}
	if rest != "" && strings.HasPrefix(lowerToken, restLower) {
		return true
	}
	return false
}

func accessHostRowMatch(rest, hostSuffix string) bool {
	if rest == "" {
		return true
	}
	if strings.ContainsAny(rest, " \t") {
		return false
	}
	// Exact Title-case reserved tokens only match reserved rows, not a host named new/local/offline.
	if rest == slashaccess.ReservedLocal || rest == slashaccess.ReservedNew || rest == slashaccess.ReservedOffline {
		return false
	}
	restLower := strings.ToLower(rest)
	hostLower := strings.ToLower(hostSuffix)
	return strings.HasPrefix(hostLower, restLower)
}

// accessTargetMatch matches /access rows: reserved "New"/"Local" use case rules; host rows are lowercase-only.
func accessTargetMatch(input, inputLower, cmd string) bool {
	rest, isAccess := parseAccessRest(input)
	if !isAccess {
		return false
	}
	if !strings.HasPrefix(cmd, "/access ") {
		return false
	}
	suffix := strings.TrimPrefix(cmd, "/access ")
	if suffix == "" {
		return false
	}
	switch suffix {
	case slashaccess.ReservedLocal:
		return reservedRowMatch(rest, slashaccess.ReservedLocal, slashaccess.FilterLocal)
	case slashaccess.ReservedNew:
		return reservedRowMatch(rest, slashaccess.ReservedNew, slashaccess.FilterNew)
	case slashaccess.ReservedOffline:
		return reservedRowMatch(rest, slashaccess.ReservedOffline, slashaccess.FilterOffline)
	default:
		return accessHostRowMatch(rest, suffix)
	}
}

func skillInstalledRowMatch(rest, skillSuffix string) bool {
	if rest == "" {
		return true
	}
	token := rest
	if fields := strings.Fields(rest); len(fields) > 0 {
		token = fields[0]
	}
	if strings.ContainsAny(token, " \t") {
		return false
	}
	// Exact Title-case reserved tokens only match reserved rows, not installed skills with the same lowercase name.
	switch token {
	case slashskill.ReservedNew, slashskill.ReservedRemove, slashskill.ReservedUpdate:
		return false
	}
	return strings.HasPrefix(strings.ToLower(skillSuffix), strings.ToLower(token))
}

// skillTargetMatch matches /skill rows: reserved "New" uses case rules; installed skills are matched by first token prefix.
func skillTargetMatch(input, cmd string) bool {
	rest, isSkill := parseSkillRest(input)
	if !isSkill {
		return false
	}
	if !strings.HasPrefix(cmd, slashskill.Prefix) {
		return false
	}
	suffix := strings.TrimPrefix(cmd, slashskill.Prefix)
	if suffix == "" {
		return false
	}
	switch suffix {
	case slashskill.ReservedNew:
		return reservedRowMatch(rest, slashskill.ReservedNew, slashskill.FilterNew)
	case slashskill.ReservedRemove:
		return reservedRowMatch(rest, slashskill.ReservedRemove, slashskill.FilterRemove)
	case slashskill.ReservedUpdate:
		return reservedRowMatch(rest, slashskill.ReservedUpdate, slashskill.FilterUpdate)
	default:
		return skillInstalledRowMatch(rest, suffix)
	}
}

// configRemoveRemoteHostMatch matches /config remove-remote <host> rows against "config …" input.
func configDelRemoteHostMatch(inputLower, cmd string) bool {
	host, ok := strings.CutPrefix(cmd, "/config remove-remote ")
	if !ok || host == "" {
		return false
	}
	rest := strings.TrimSpace(strings.TrimPrefix(inputLower, "config"))
	rest = strings.TrimSpace(strings.TrimPrefix(rest, "remove-remote"))
	if rest == "" {
		return true
	}
	return strings.HasPrefix(strings.ToLower(host), strings.ToLower(rest))
}

type Option struct {
	Cmd       string
	Desc      string
	FillValue string
}

// VisibleIndices filters options by input prefix and returns matching indices.
func VisibleIndices(input string, opts []Option) []int {
	input = strings.TrimPrefix(input, "/")
	input = strings.TrimSpace(input)
	inputLower := strings.ToLower(input)
	if len(opts) == 1 {
		return []int{0}
	}
	var out []int
	for i, opt := range opts {
		base := strings.Split(opt.Cmd, " ")[0]
		base = strings.TrimPrefix(base, "/")
		if strings.HasPrefix(opt.Cmd, "/access ") {
			switch {
			case inputLower == "":
				out = append(out, i)
			case strings.HasPrefix(inputLower, "access"):
				if accessTargetMatch(input, inputLower, opt.Cmd) {
					out = append(out, i)
				}
			default:
				// e.g. /a narrows to /access without spelling "access"
				if strings.HasPrefix(base, inputLower) || strings.HasPrefix(strings.ToLower(opt.Cmd), "/"+inputLower) {
					out = append(out, i)
				}
			}
			continue
		}
		if strings.HasPrefix(opt.Cmd, slashskill.Prefix) {
			switch {
			case inputLower == "":
				out = append(out, i)
			case strings.HasPrefix(inputLower, slashskill.Subcommand):
				if skillTargetMatch(input, opt.Cmd) {
					out = append(out, i)
				}
			default:
				if strings.HasPrefix(base, inputLower) || strings.HasPrefix(strings.ToLower(opt.Cmd), "/"+inputLower) {
					out = append(out, i)
				}
			}
			continue
		}
		if inputLower == "" || strings.HasPrefix(base, inputLower) || strings.HasPrefix(opt.Cmd, "/"+inputLower) ||
			configDelRemoteHostMatch(inputLower, opt.Cmd) {
			out = append(out, i)
		}
	}
	// When the user typed something that matches no command prefix, show nothing — not the full list
	// (a full list reads like random noise after typos like "/zzz").
	if len(out) == 0 && inputLower != "" {
		return nil
	}
	if len(out) == 0 {
		for i := range opts {
			out = append(out, i)
		}
	}
	return out
}

// ChosenToInputValue converts chosen slash command to input value.
// The result always ends with a single trailing ASCII space so the user can continue typing;
// execution paths should use strings.TrimSpace on the full line before interpreting the command.
func ChosenToInputValue(chosen Option) string {
	var s string
	if chosen.FillValue != "" {
		s = chosen.FillValue
	} else if i := strings.Index(chosen.Cmd, " <"); i > 0 {
		s = chosen.Cmd[:i] + " "
	} else if i := strings.Index(chosen.Cmd, " {"); i > 0 {
		// e.g. "/skill {name} [...]" — fill only the command prefix, not display placeholders
		s = chosen.Cmd[:i] + " "
	}
	if s == "" {
		s = chosen.Cmd
	}
	return strings.TrimRight(s, " \t") + " "
}
