package enterflow

import (
	"delve-shell/internal/maininput"
	"delve-shell/internal/slashview"
)

// PlanAfterSlashDispatches classifies the main Enter path after exact/prefix slash dispatch missed,
// using the same rules as [maininput.PlanMainEnter].
func PlanAfterSlashDispatches(
	text string,
	slashSelectedIndex int,
	viewOpts []slashview.Option,
	vis []int,
	sessionNoneMsg, delRemoteNoneMsg string,
) maininput.MainEnterPlan {
	return maininput.PlanMainEnter(maininput.MainEnterInput{
		Text:               text,
		SlashSelectedIndex: slashSelectedIndex,
		Options:            viewOpts,
		Visible:            vis,
		SessionNoneMsg:     sessionNoneMsg,
		DelRemoteNoneMsg:   delRemoteNoneMsg,
	})
}
