package execcancel

import (
	"context"
	"testing"
)

func TestHub_Cancel_invokesAllActive(t *testing.T) {
	h := New()
	ctx1, cancel1 := context.WithCancel(context.Background())
	ctx2, cancel2 := context.WithCancel(context.Background())
	unreg1 := h.Register(cancel1)
	unreg2 := h.Register(cancel2)

	if !h.Cancel() {
		t.Fatal("expected Cancel true with two registrations")
	}
	if err := ctx1.Err(); err == nil {
		t.Error("ctx1 should be canceled")
	}
	if err := ctx2.Err(); err == nil {
		t.Error("ctx2 should be canceled")
	}

	unreg1()
	unreg2()
	if h.Cancel() {
		t.Error("expected Cancel false after unregister")
	}
}

func TestHub_unregisterOnlyRemovesSelf(t *testing.T) {
	h := New()
	_, cancel1 := context.WithCancel(context.Background())
	ctx2, cancel2 := context.WithCancel(context.Background())
	unreg1 := h.Register(cancel1)
	unreg2 := h.Register(cancel2)
	unreg1()

	if !h.Cancel() {
		t.Fatal("expected Cancel true while ctx2 still registered")
	}
	if ctx2.Err() == nil {
		t.Error("ctx2 should be canceled")
	}
	unreg2()
}
