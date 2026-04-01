package ui

import "testing"

func TestJoinMainBodyAboveBottomChrome(t *testing.T) {
	t.Run("no trailing newline inserts boundary", func(t *testing.T) {
		got := joinMainBodyAboveBottomChrome("a\nb", "sep\n")
		want := "a\nb\nsep\n"
		if got != want {
			t.Fatalf("got %q want %q", got, want)
		}
	})
	t.Run("trailing newline unchanged", func(t *testing.T) {
		got := joinMainBodyAboveBottomChrome("a\nb\n", "sep\n")
		want := "a\nb\nsep\n"
		if got != want {
			t.Fatalf("got %q want %q", got, want)
		}
	})
	t.Run("empty mainBody", func(t *testing.T) {
		if got := joinMainBodyAboveBottomChrome("", "only"); got != "only" {
			t.Fatalf("got %q want only", got)
		}
	})
}
