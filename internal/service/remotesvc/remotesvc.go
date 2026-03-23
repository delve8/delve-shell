// Package remotesvc is a thin facade over internal/config for remote target CRUD.
// Call from UI instead of importing config for these mutations; tests may use SetImplForTest.
package remotesvc

import (
	"delve-shell/internal/config"
)

var (
	addFn    = config.AddRemote
	updateFn = config.UpdateRemote
	removeFn = config.RemoveRemoteByName
)

// SetImplForTest overrides underlying functions for unit tests.
func SetImplForTest(add func(target, name, identityFile string) error, update func(target, name, identityFile string) error, remove func(nameOrTarget string) error) {
	if add != nil {
		addFn = add
	}
	if update != nil {
		updateFn = update
	}
	if remove != nil {
		removeFn = remove
	}
}

func Add(target, name, identityFile string) error {
	return addFn(target, name, identityFile)
}

func Update(target, name, identityFile string) error {
	return updateFn(target, name, identityFile)
}

func Remove(nameOrTarget string) error {
	return removeFn(nameOrTarget)
}
