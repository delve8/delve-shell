package run

// Register wires shared /config slash completion, /bash, and related UI hooks into the UI.
// Order matches historical package init order (lexicographic file names). Call from [bootstrap.Install].
func Register() {
	registerSlashOptionsProviders()
}
