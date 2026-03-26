package wiring

import "testing"

func TestBindAllowlistAutoRun_GetterVisible(t *testing.T) {
	rt := installTestRuntime(t)
	BindAllowlistAutoRun(rt, func() bool { return false }, func(bool) {})
	if rt.AllowlistAutoRunEnabled() {
		t.Fatal("getter should return false")
	}
	BindAllowlistAutoRun(rt, func() bool { return true }, func(bool) {})
	if !rt.AllowlistAutoRunEnabled() {
		t.Fatal("getter should return true")
	}
}

func TestBindAllowlistAutoRun_RebindOverrides(t *testing.T) {
	rt := installTestRuntime(t)
	BindAllowlistAutoRun(rt, func() bool { return false }, func(bool) {})
	BindAllowlistAutoRun(rt, func() bool { return true }, func(bool) {})
	if !rt.AllowlistAutoRunEnabled() {
		t.Fatal("second bind should win")
	}
}

func TestBindAllowlistAutoRun_NilGetterFallsBackToTrue(t *testing.T) {
	rt := installTestRuntime(t)
	BindAllowlistAutoRun(rt, nil, func(bool) {})
	if !rt.AllowlistAutoRunEnabled() {
		t.Fatal("nil getter should default to true in AllowlistAutoRunEnabled")
	}
}
