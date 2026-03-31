package skill

// OverlayFeatureKey is the stable overlay registration ID for skill config UI.
const OverlayFeatureKey = "skill"

// SlashSubcommand is the slash command name for skill invocation ("/skill …").
const SlashSubcommand = "skill"

// Overlay open request keys ([inputlifecycletype.OverlayPayload.Key]) for skill flows.
const (
	OverlayOpenKeyAdd    = "skill_add"
	OverlayOpenKeyUpdate = "skill_update"
)
