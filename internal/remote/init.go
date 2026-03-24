package remote

func init() {
	registerSlashExactHandlers()
	registerSlashPrefixHandlers()
	registerProviders()
}
