package skillsvc

import (
	"errors"
	"testing"
)

func TestInstallUpdateRemove_DelegateToImpl(t *testing.T) {
	t.Cleanup(func() {
		SetImplForTest(nil, nil, nil)
	})

	installCalled := false
	updateCalled := false
	removeCalled := false

	SetImplForTest(
		func(url, ref, name, path string) (string, error) {
			installCalled = true
			if url != "u" || ref != "r" || name != "n" || path != "p" {
				t.Fatalf("unexpected install args: %q %q %q %q", url, ref, name, path)
			}
			return "final", nil
		},
		func(name, newRef string) error {
			updateCalled = true
			if name != "s" || newRef != "main" {
				t.Fatalf("unexpected update args: %q %q", name, newRef)
			}
			return nil
		},
		func(name string) error {
			removeCalled = true
			if name != "x" {
				t.Fatalf("unexpected remove args: %q", name)
			}
			return nil
		},
	)

	if got, err := InstallFromGit("u", "r", "n", "p"); err != nil || got != "final" {
		t.Fatalf("install: got=%q err=%v", got, err)
	}
	if err := Update("s", "main"); err != nil {
		t.Fatalf("update: %v", err)
	}
	if err := Remove("x"); err != nil {
		t.Fatalf("remove: %v", err)
	}
	if !installCalled || !updateCalled || !removeCalled {
		t.Fatalf("expected all impl called")
	}
}

func TestUpdate_PropagatesError(t *testing.T) {
	t.Cleanup(func() {
		SetImplForTest(nil, nil, nil)
	})
	want := errors.New("boom")
	SetImplForTest(nil, func(name, newRef string) error { return want }, nil)
	if err := Update("s", "r"); err == nil {
		t.Fatalf("expected error")
	}
}
