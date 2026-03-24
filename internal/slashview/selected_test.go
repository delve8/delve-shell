package slashview

import "testing"

func TestSelectedByVisibleIndex_ReturnsOption(t *testing.T) {
	opts := []Option{{Cmd: "/help"}, {Cmd: "/run <cmd>"}, {Cmd: "/remote on"}}
	vis := []int{1, 2}
	got, ok := SelectedByVisibleIndex(opts, vis, 0)
	if !ok || got.Cmd != "/run <cmd>" {
		t.Fatalf("unexpected selected option: %+v ok=%v", got, ok)
	}
}

func TestSelectedByVisibleIndex_OutOfRange(t *testing.T) {
	opts := []Option{{Cmd: "/help"}}
	vis := []int{0}
	_, ok := SelectedByVisibleIndex(opts, vis, 1)
	if ok {
		t.Fatalf("expected not found for out-of-range index")
	}
}
