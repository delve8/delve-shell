package config

import (
	"fmt"
	"os"
	"strings"

	"gopkg.in/yaml.v3"
)

// remotesFile is the remotes.yaml file structure.
type remotesFile struct {
	Remotes []RemoteTarget `yaml:"remotes"`
}

// LoadRemotes loads remotes from remotes.yaml. If the file does not exist, creates it with empty list and returns it.
func LoadRemotes() ([]RemoteTarget, error) {
	path := RemotesPath()
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			if err := EnsureRootDir(); err != nil {
				return nil, err
			}
			if err := WriteRemotes(nil); err != nil {
				return nil, err
			}
			return nil, nil
		}
		return nil, err
	}
	var f remotesFile
	if err := yaml.Unmarshal(data, &f); err != nil {
		return nil, err
	}
	return f.Remotes, nil
}

// WriteRemotes writes remotes to remotes.yaml (call after changes; EnsureRootDir before first write).
func WriteRemotes(remotes []RemoteTarget) error {
	if remotes == nil {
		remotes = []RemoteTarget{}
	}
	data, err := yaml.Marshal(remotesFile{Remotes: remotes})
	if err != nil {
		return err
	}
	return os.WriteFile(RemotesPath(), data, 0600)
}

// HostFromTarget returns the host part of target (user@host or user@host:port). If no "@", returns target as-is.
func HostFromTarget(target string) string {
	target = strings.TrimSpace(target)
	if i := strings.Index(target, "@"); i >= 0 && i < len(target)-1 {
		return target[i+1:]
	}
	return target
}

// isUserAtHost returns true if s looks like user@host or user@host:port (at least one @ with non-empty parts).
func isUserAtHost(s string) bool {
	i := strings.Index(s, "@")
	if i <= 0 || i >= len(s)-1 {
		return false
	}
	if strings.Count(s, "@") != 1 {
		return false
	}
	return true
}

// AddRemote appends a remote to remotes.yaml. Target is required (user@host or user@host:port); name is an optional label.
// Duplicate is checked by target (same target cannot be added twice).
func AddRemote(target, name, identityFile string) error {
	target = strings.TrimSpace(target)
	if target == "" {
		return fmt.Errorf("target (user@host) is required")
	}
	if !isUserAtHost(target) {
		return fmt.Errorf("target must be user@host or user@host:port (e.g. root@192.168.1.1)")
	}
	remotes, err := LoadRemotes()
	if err != nil {
		return err
	}
	for _, r := range remotes {
		if r.Target == target {
			return fmt.Errorf("remote target already exists: %s", target)
		}
	}
	remotes = append(remotes, RemoteTarget{
		Name:         strings.TrimSpace(name),
		Target:       target,
		IdentityFile: strings.TrimSpace(identityFile),
	})
	return WriteRemotes(remotes)
}

// UpdateRemote updates an existing remote with the same target (name and identity file). Returns error if target not found.
func UpdateRemote(target, name, identityFile string) error {
	target = strings.TrimSpace(target)
	if target == "" {
		return fmt.Errorf("target (user@host) is required")
	}
	if !isUserAtHost(target) {
		return fmt.Errorf("target must be user@host or user@host:port (e.g. root@192.168.1.1)")
	}
	remotes, err := LoadRemotes()
	if err != nil {
		return err
	}
	found := false
	for i := range remotes {
		if remotes[i].Target == target {
			remotes[i].Name = strings.TrimSpace(name)
			remotes[i].IdentityFile = strings.TrimSpace(identityFile)
			found = true
			break
		}
	}
	if !found {
		return fmt.Errorf("remote not found: %s", target)
	}
	return WriteRemotes(remotes)
}

// RemoveRemoteByName removes the remote matching the given name or target from remotes.yaml. Returns error if not found.
func RemoveRemoteByName(nameOrTarget string) error {
	nameOrTarget = strings.TrimSpace(nameOrTarget)
	if nameOrTarget == "" {
		return fmt.Errorf("name or target is required")
	}
	remotes, err := LoadRemotes()
	if err != nil {
		return err
	}
	out := remotes[:0]
	for _, r := range remotes {
		if r.Name != nameOrTarget && r.Target != nameOrTarget && HostFromTarget(r.Target) != nameOrTarget {
			out = append(out, r)
		}
	}
	if len(out) == len(remotes) {
		return fmt.Errorf("remote not found: %s", nameOrTarget)
	}
	return WriteRemotes(out)
}
