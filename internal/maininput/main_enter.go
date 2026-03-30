package maininput

import (
	"strings"

	"delve-shell/internal/slashflow"
	"delve-shell/internal/slashview"
)

type MainEnterPlanKind int

const (
	MainEnterPassToSubmit MainEnterPlanKind = iota
	MainEnterShowSessionNone
	MainEnterShowDelRemoteNone
	MainEnterResolveSelected
	MainEnterUnknownSlash
)

type MainEnterPlan struct {
	Kind     MainEnterPlanKind
	Chosen   string
	Selected slashview.Option
}

type MainEnterInput struct {
	Text               string
	SlashSelectedIndex int
	Options            []slashview.Option
	Visible            []int
	SessionNoneMsg     string
	DelRemoteNoneMsg   string
}

func isHistorySlashInput(s string) bool {
	s = strings.TrimSpace(s)
	return s == "/history" || strings.HasPrefix(s, "/history ")
}

func PlanMainEnter(in MainEnterInput) MainEnterPlan {
	if len(in.Text) == 0 || in.Text[0] != '/' {
		return MainEnterPlan{Kind: MainEnterPassToSubmit}
	}
	selected, ok := slashview.SelectedByVisibleIndex(in.Options, in.Visible, in.SlashSelectedIndex)
	if !ok {
		selected = slashview.Option{}
	}
	outcome := slashflow.EvaluateMainEnter(in.Text, slashflow.EnterInput{
		HasSlashPrefix:     true,
		Selected:           selected,
		VisibleOptionCount: len(in.Visible),
		// Do not use strings.HasPrefix(..., "/history"): "/historyx" would match incorrectly.
		IsSessionNoneOption:   isHistorySlashInput(in.Text) && selected.Cmd == in.SessionNoneMsg,
		IsDelRemoteNoneOption: selected.Cmd == in.DelRemoteNoneMsg,
	})
	switch outcome {
	case slashflow.OutcomeShowSessionNone:
		return MainEnterPlan{Kind: MainEnterShowSessionNone}
	case slashflow.OutcomeShowDelRemoteNone:
		return MainEnterPlan{Kind: MainEnterShowDelRemoteNone}
	case slashflow.OutcomeResolveSelected:
		return MainEnterPlan{Kind: MainEnterResolveSelected, Chosen: selected.Cmd, Selected: selected}
	case slashflow.OutcomeUnknownSlash:
		return MainEnterPlan{Kind: MainEnterUnknownSlash}
	default:
		return MainEnterPlan{Kind: MainEnterPassToSubmit}
	}
}
