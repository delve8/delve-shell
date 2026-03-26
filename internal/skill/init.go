package skill

// Register wires skill-related slash commands, overlays, and message providers into the UI. Call from [bootstrap.Install].
func Register() {
	registerSlashExecutionProvider()
	registerOverlayFeature()
	registerSlashOptionsProvider()
}
