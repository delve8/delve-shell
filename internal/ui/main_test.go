package ui

import (
	"os"
	"testing"
)

func TestMain(m *testing.M) {
	ensureTestFeatureMirrorsRegistered()
	os.Exit(m.Run())
}
