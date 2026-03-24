package ui

import "testing"

func TestTitleBarLeadingSegment(t *testing.T) {
	t.Run("default local when inactive", func(t *testing.T) {
		var m Model
		if got := m.titleBarLeadingSegment(); got != "Local" {
			t.Fatalf("got %q want Local", got)
		}
	})
	t.Run("remote without label", func(t *testing.T) {
		m := Model{RemoteActive: true}
		if got := m.titleBarLeadingSegment(); got != "Remote" {
			t.Fatalf("got %q want Remote", got)
		}
	})
	t.Run("remote with label", func(t *testing.T) {
		m := Model{RemoteActive: true, RemoteLabel: "prod"}
		if got := m.titleBarLeadingSegment(); got != "Remote prod" {
			t.Fatalf("got %q want Remote prod", got)
		}
	})
}
