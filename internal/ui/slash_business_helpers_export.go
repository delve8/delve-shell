package ui

// ApplyConfigAddRemoteArgs exposes applyConfigAddRemote to feature packages.
func (m Model) ApplyConfigAddRemoteArgs(args string) Model {
	return m.applyConfigAddRemote(args)
}

// ApplyConfigRemoveRemote exposes applyConfigRemoveRemote to feature packages.
func (m Model) ApplyConfigRemoveRemote(nameOrTarget string) Model {
	return m.applyConfigRemoveRemote(nameOrTarget)
}
