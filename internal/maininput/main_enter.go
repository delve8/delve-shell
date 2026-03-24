package maininput

import (
	"delve-shell/internal/slashflow"
	"delve-shell/internal/slashview"
)

type MainEnterPlanKind int

const (
	MainEnterPassToSubmit MainEnterPlanKind = iota
	MainEnterSwitchSession
	MainEnterShowSessionNone
	MainEnterResolveSelected
	MainEnterUnknownSlash
)

type MainEnterPlan struct {
	Kind   MainEnterPlanKind
	Chosen string
}

type MainEnterInput struct {
	Text               string
	SlashSelectedPath  string
	SlashSelectedIndex int
	Options            []slashview.Option
	Visible            []int
	SessionNoneMsg     string
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
		HasSlashPrefix:      true,
		SelectedPath:        in.SlashSelectedPath,
		SelectedCmd:         selected.Cmd,
		VisibleOptionCount:  len(in.Visible),
		IsSessionNoneOption: selected.Path == "" && selected.Cmd == in.SessionNoneMsg,
	})
	switch outcome {
	case slashflow.OutcomeSwitchSession:
		return MainEnterPlan{Kind: MainEnterSwitchSession}
	case slashflow.OutcomeShowSessionNone:
		return MainEnterPlan{Kind: MainEnterShowSessionNone}
	case slashflow.OutcomeResolveSelected:
		return MainEnterPlan{Kind: MainEnterResolveSelected, Chosen: selected.Cmd}
	case slashflow.OutcomeUnknownSlash:
		return MainEnterPlan{Kind: MainEnterUnknownSlash}
	default:
		return MainEnterPlan{Kind: MainEnterPassToSubmit}
	}
}
