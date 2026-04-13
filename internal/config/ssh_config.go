package config

import (
	"os"
	osuser "os/user"
	"path/filepath"
	"strconv"
	"strings"

	sshconfig "github.com/kevinburke/ssh_config"
)

// SSHConfigHost is one explicit Host alias loaded from ~/.ssh/config.
// Wildcard patterns are skipped; only directly addressable aliases are returned.
type SSHConfigHost struct {
	Alias        string
	HostName     string
	User         string
	Port         string
	IdentityFile string
	Target       string
}

// SSHUserConfigPath returns the user's OpenSSH client config path (~/.ssh/config).
func SSHUserConfigPath() string {
	home, err := os.UserHomeDir()
	if err != nil || strings.TrimSpace(home) == "" {
		return ""
	}
	return filepath.Join(home, ".ssh", "config")
}

// LoadSSHConfigHosts returns explicit Host aliases from ~/.ssh/config, resolved to concrete
// target values usable by the in-process SSH executor. Wildcard and negated Host patterns are skipped.
func LoadSSHConfigHosts() ([]SSHConfigHost, error) {
	cfg, err := loadUserSSHConfig()
	if err != nil || cfg == nil {
		return nil, err
	}

	seen := make(map[string]struct{})
	hosts := make([]SSHConfigHost, 0, 8)
	for _, host := range cfg.Hosts {
		if host == nil {
			continue
		}
		for _, pattern := range host.Patterns {
			if pattern == nil {
				continue
			}
			alias := strings.TrimSpace(pattern.String())
			if !isExplicitSSHConfigAlias(alias) {
				continue
			}
			key := strings.ToLower(alias)
			if _, ok := seen[key]; ok {
				continue
			}
			entry, ok := resolveSSHConfigAlias(cfg, alias)
			if !ok {
				continue
			}
			seen[key] = struct{}{}
			hosts = append(hosts, entry)
		}
	}
	return hosts, nil
}

// ResolveSSHConfigHost resolves one ~/.ssh/config host entry to a concrete target and first identity file.
// Matching is case-insensitive against explicit Host aliases, resolved HostName, and concrete target hostnames.
func ResolveSSHConfigHost(alias string) (SSHConfigHost, bool, error) {
	cfg, err := loadUserSSHConfig()
	if err != nil || cfg == nil {
		return SSHConfigHost{}, false, err
	}

	want := strings.TrimSpace(alias)
	if want == "" {
		return SSHConfigHost{}, false, nil
	}
	for _, host := range cfg.Hosts {
		if host == nil {
			continue
		}
		for _, pattern := range host.Patterns {
			if pattern == nil {
				continue
			}
			candidate := strings.TrimSpace(pattern.String())
			if !isExplicitSSHConfigAlias(candidate) {
				continue
			}
			entry, ok := resolveSSHConfigAlias(cfg, candidate)
			if !ok {
				continue
			}
			if sshConfigHostMatches(entry, want) {
				return entry, true, nil
			}
		}
	}
	return SSHConfigHost{}, false, nil
}

func sshConfigHostMatches(entry SSHConfigHost, want string) bool {
	if strings.EqualFold(entry.Alias, want) || strings.EqualFold(entry.HostName, want) || strings.EqualFold(entry.Target, want) {
		return true
	}
	targetHost := HostFromTarget(want)
	if targetHost == "" {
		return false
	}
	return strings.EqualFold(entry.HostName, targetHost) || strings.EqualFold(HostFromTarget(entry.Target), targetHost)
}

func loadUserSSHConfig() (*sshconfig.Config, error) {
	path := SSHUserConfigPath()
	if strings.TrimSpace(path) == "" {
		return nil, nil
	}
	f, err := os.Open(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, err
	}
	defer f.Close()
	return sshconfig.Decode(f)
}

func isExplicitSSHConfigAlias(alias string) bool {
	alias = strings.TrimSpace(alias)
	if alias == "" {
		return false
	}
	if strings.HasPrefix(alias, "!") {
		return false
	}
	return !strings.ContainsAny(alias, "*?")
}

func resolveSSHConfigAlias(cfg *sshconfig.Config, alias string) (SSHConfigHost, bool) {
	if cfg == nil {
		return SSHConfigHost{}, false
	}
	alias = strings.TrimSpace(alias)
	if alias == "" {
		return SSHConfigHost{}, false
	}

	hostName, _ := cfg.Get(alias, "HostName")
	hostName = strings.TrimSpace(hostName)

	port, _ := cfg.Get(alias, "Port")
	port = strings.TrimSpace(port)
	if port == "" {
		port = "22"
	}
	if _, err := strconv.Atoi(port); err != nil {
		return SSHConfigHost{}, false
	}

	remoteUser, _ := cfg.Get(alias, "User")
	remoteUser = strings.TrimSpace(remoteUser)
	if remoteUser == "" {
		remoteUser = localUsername()
	}
	if remoteUser == "" {
		return SSHConfigHost{}, false
	}

	if hostName == "" {
		hostName = alias
	} else {
		hostName = expandSSHConfigTokens(hostName, alias, alias, port, remoteUser)
	}

	identityFiles, _ := cfg.GetAll(alias, "IdentityFile")
	identityFile := ""
	for _, candidate := range identityFiles {
		candidate = strings.TrimSpace(candidate)
		if candidate == "" {
			continue
		}
		identityFile = expandSSHConfigPath(candidate, alias, hostName, port, remoteUser)
		break
	}

	target := remoteUser + "@" + hostName
	if port != "" && port != "22" {
		target += ":" + port
	}

	return SSHConfigHost{
		Alias:        alias,
		HostName:     hostName,
		User:         remoteUser,
		Port:         port,
		IdentityFile: identityFile,
		Target:       target,
	}, true
}

func localUsername() string {
	if u := strings.TrimSpace(os.Getenv("USER")); u != "" {
		return u
	}
	if u := strings.TrimSpace(os.Getenv("LOGNAME")); u != "" {
		return u
	}
	current, err := osuser.Current()
	if err != nil {
		return ""
	}
	return strings.TrimSpace(current.Username)
}

func expandSSHConfigPath(raw, alias, hostName, port, remoteUser string) string {
	expanded := expandSSHConfigTokens(raw, alias, hostName, port, remoteUser)
	if strings.HasPrefix(expanded, "~") {
		if home, err := os.UserHomeDir(); err == nil && strings.TrimSpace(home) != "" {
			expanded = filepath.Join(home, strings.TrimPrefix(expanded, "~"))
		}
	}
	return expanded
}

func expandSSHConfigTokens(raw, alias, hostName, port, remoteUser string) string {
	home, _ := os.UserHomeDir()
	localUser := localUsername()
	const percentSentinel = "\x00"
	replacer := strings.NewReplacer(
		"%%", percentSentinel,
		"%d", home,
		"%h", hostName,
		"%n", alias,
		"%p", port,
		"%r", remoteUser,
		"%u", localUser,
	)
	return strings.ReplaceAll(replacer.Replace(raw), percentSentinel, "%")
}
