package remotesvc

import "testing"

func TestDelegation(t *testing.T) {
	t.Cleanup(func() { SetImplForTest(nil, nil, nil) })

	addCalled := false
	updateCalled := false
	removeCalled := false

	SetImplForTest(
		func(target, name, identityFile string) error {
			addCalled = true
			if target != "t" || name != "n" || identityFile != "k" {
				t.Fatalf("unexpected add args")
			}
			return nil
		},
		func(target, name, identityFile string) error {
			updateCalled = true
			return nil
		},
		func(nameOrTarget string) error {
			removeCalled = true
			if nameOrTarget != "x" {
				t.Fatalf("unexpected remove arg")
			}
			return nil
		},
	)

	if err := Add("t", "n", "k"); err != nil {
		t.Fatalf("add: %v", err)
	}
	if err := Update("t", "n2", "k2"); err != nil {
		t.Fatalf("update: %v", err)
	}
	if err := Remove("x"); err != nil {
		t.Fatalf("remove: %v", err)
	}
	if !addCalled || !updateCalled || !removeCalled {
		t.Fatalf("expected all functions called")
	}
}
