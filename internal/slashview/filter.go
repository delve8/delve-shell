package slashview

import "strings"

// remoteOnHostMatch matches /remote on <host> rows when input is like "remote", "remote p", "remote on pr"
// (prefix on the host segment, same idea as /run <cmd> prefix matching).
func remoteOnHostMatch(inputLower, cmd string) bool {
	host, ok := strings.CutPrefix(cmd, "/remote on ")
	if !ok || host == "" {
		return false
	}
	rest := strings.TrimSpace(strings.TrimPrefix(inputLower, "remote"))
	rest = strings.TrimSpace(strings.TrimPrefix(rest, "on"))
	if rest == "" {
		return true
	}
	return strings.HasPrefix(strings.ToLower(host), strings.ToLower(rest))
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
		if inputLower == "" || strings.HasPrefix(base, inputLower) || strings.HasPrefix(opt.Cmd, "/"+inputLower) ||
			remoteOnHostMatch(inputLower, opt.Cmd) || configDelRemoteHostMatch(inputLower, opt.Cmd) {
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
func ChosenToInputValue(chosen Option) string {
	if chosen.FillValue != "" {
		return chosen.FillValue
	}
	if strings.Contains(chosen.Cmd, " <") {
		if i := strings.Index(chosen.Cmd, " <"); i > 0 {
			return chosen.Cmd[:i] + " "
		}
	}
	return chosen.Cmd
}
