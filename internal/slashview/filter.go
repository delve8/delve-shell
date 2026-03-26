package slashview

import "strings"

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
		if inputLower == "" || strings.HasPrefix(base, inputLower) || strings.HasPrefix(opt.Cmd, "/"+inputLower) {
			out = append(out, i)
		}
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
