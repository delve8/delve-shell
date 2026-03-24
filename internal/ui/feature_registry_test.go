package ui

import "sync"

var testMirrorRegisterOnce sync.Once

// ensureTestFeatureMirrorsRegistered installs test-only fallback registrations so
// package ui tests can run without importing feature packages directly.
func ensureTestFeatureMirrorsRegistered() {
	testMirrorRegisterOnce.Do(func() {
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
	})
}
