package slashview

import "strings"

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

func accessReservedRowMatch(rest, displayToken, lowerToken string) bool {
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
	if rest == "Local" || rest == "New" || rest == "Offline" {
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
	case "Local":
		return accessReservedRowMatch(rest, "Local", "local")
	case "New":
		return accessReservedRowMatch(rest, "New", "new")
	case "Offline":
		return accessReservedRowMatch(rest, "Offline", "offline")
	default:
		return accessHostRowMatch(rest, suffix)
	}
}

// configDelRemoteHostMatch matches /config del-remote <host> rows against "config …" input.
func configDelRemoteHostMatch(inputLower, cmd string) bool {
	host, ok := strings.CutPrefix(cmd, "/config del-remote ")
	if !ok || host == "" {
		return false
	}
	rest := strings.TrimSpace(strings.TrimPrefix(inputLower, "config"))
	rest = strings.TrimSpace(strings.TrimPrefix(rest, "del-remote"))
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
	} else if strings.Contains(chosen.Cmd, " <") {
		if i := strings.Index(chosen.Cmd, " <"); i > 0 {
			s = chosen.Cmd[:i] + " "
		}
	}
	if s == "" {
		s = chosen.Cmd
	}
	return strings.TrimRight(s, " \t") + " "
}
