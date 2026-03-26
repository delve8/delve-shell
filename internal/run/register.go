package run

// Register wires /run, shared /config exact routes, slash completion, and overlay hooks into the UI.
// Order matches historical package init order (lexicographic file names). Call from [bootstrap.Install].
func Register() {
	registerSlashExecutionProvider()
	registerSlashOptionsProviders()
}
