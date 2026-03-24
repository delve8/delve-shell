package ui

// Test-only fallback registrations so internal/ui unit tests can run without importing
// feature packages directly. Keeping mirrors in package ui avoids import cycles in ui tests.
func init() {
	registerTestTitleBarMirror()
	registerTestExactOverlayMirrors()
	registerTestSlashExactMirrors()
	registerTestStaticSlashOptionsMirror()
	registerTestStartupOverlayMirror()
	registerTestSkillPrefixMirrors()
	registerTestConfigPrefixMirrors()
	registerTestSessionMessageMirror()
	registerTestSlashSelectedMirrors()
	registerTestSessionSlashOptionsMirror()
	registerTestOverlayCloseResetMirror()
}
