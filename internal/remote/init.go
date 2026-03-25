package remote

// Register wires remote slash commands and UI providers. Call from [bootstrap.Install].
func Register() {
	registerSlashExecutionProvider()
	registerProviders()
}
