package skillsvc

import (
	"errors"
	"os"

	"delve-shell/internal/skills"
)

// Small indirection layer so UI can call app/service code and tests can stub behavior.
var (
	installFromGitFn = skills.InstallFromGit
	updateFn         = skills.Update
	removeFn         = skills.Remove
)

// SetImplForTest overrides the underlying implementation functions for unit tests.
// Pass nil to keep the default.
func SetImplForTest(install func(url, ref, name, path string) (string, error), update func(name, newRef string) error, remove func(name string) error) {
	if install != nil {
		installFromGitFn = install
	}
	if update != nil {
		updateFn = update
	}
	if remove != nil {
		removeFn = remove
	}
}

func InstallFromGit(url, ref, localName, path string) (string, error) {
	return installFromGitFn(url, ref, localName, path)
}

func Update(name, newRef string) error {
	return updateFn(name, newRef)
}

func Remove(name string) error {
	err := removeFn(name)
	// Keep UI-friendly behavior: when a skill doesn't exist, treat as error with original cause.
	if err != nil && errors.Is(err, os.ErrNotExist) {
		return err
	}
	return err
}

