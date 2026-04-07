package run

// Register wires /exec, shared /config slash completion, and overlay hooks into the UI.
// Order matches historical package init order (lexicographic file names). Call from [bootstrap.Install].
func Register() {
	registerSlashOptionsProviders()
}
