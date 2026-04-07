package config

import (
	"fmt"
	"maps"
	"os"

	"gopkg.in/yaml.v3"
)

// LoadAllowlist loads allowlist.yaml. If missing, writes the default file and returns it.
// If the file is unreadable, not schema v2, or fails validation, it is replaced with the default and rewritten.
func LoadAllowlist() (*LoadedAllowlist, error) {
	path := AllowlistPath()
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			def := DefaultLoadedAllowlist()
			if err := EnsureRootDir(); err != nil {
				return nil, err
			}
			if err := WriteLoadedAllowlist(def); err != nil {
				return nil, err
			}
			return def, nil
		}
		return nil, err
	}
	ld, err := ParseAllowlistYAML(data)
	if err != nil || ld.Version != AllowlistSchemaVersion {
		def := DefaultLoadedAllowlist()
		if werr := EnsureRootDir(); werr != nil {
			return nil, werr
		}
		if werr := WriteLoadedAllowlist(def); werr != nil {
			return nil, werr
		}
		return def, nil
	}
	if err := ValidateLoadedAllowlist(ld); err != nil {
		def := DefaultLoadedAllowlist()
		if werr := EnsureRootDir(); werr != nil {
			return nil, werr
		}
		if werr := WriteLoadedAllowlist(def); werr != nil {
			return nil, werr
		}
		return def, nil
	}
	ld.Commands = NormalizeReadOnlyCLIPolicies(ld.Commands)
	return ld, nil
}

// WriteLoadedAllowlist writes allowlist.yaml (EnsureRootDir before first write).
func WriteLoadedAllowlist(ld *LoadedAllowlist) error {
	if ld == nil {
		ld = DefaultLoadedAllowlist()
	}
	data, err := yaml.Marshal(ld)
	if err != nil {
		return err
	}
	return os.WriteFile(AllowlistPath(), data, 0600)
}

// ParseAllowlistYAML parses YAML bytes into LoadedAllowlist (strict v2: commands map).
func ParseAllowlistYAML(data []byte) (*LoadedAllowlist, error) {
	var ld LoadedAllowlist
	if err := yaml.Unmarshal(data, &ld); err != nil {
		return nil, err
	}
	if ld.Commands == nil {
		return nil, fmt.Errorf("allowlist: missing commands")
	}
	if ld.Version != AllowlistSchemaVersion {
		return nil, fmt.Errorf("allowlist: version %d, want %d", ld.Version, AllowlistSchemaVersion)
	}
	return &ld, nil
}

// AllowlistUpdateWithDefaults merges built-in defaults into the current file (missing command keys). Returns how many were added.
func AllowlistUpdateWithDefaults() (added int, err error) {
	path := AllowlistPath()
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			if err := EnsureRootDir(); err != nil {
				return 0, err
			}
			def := DefaultLoadedAllowlist()
			if err := WriteLoadedAllowlist(def); err != nil {
				return 0, err
			}
			return len(def.Commands), nil
		}
		return 0, err
	}
	cur, err := ParseAllowlistYAML(data)
	if err != nil || cur.Version != AllowlistSchemaVersion {
		def := DefaultLoadedAllowlist()
		if err := EnsureRootDir(); err != nil {
			return 0, err
		}
		if err := WriteLoadedAllowlist(def); err != nil {
			return 0, err
		}
		return len(def.Commands), nil
	}
	if err := ValidateLoadedAllowlist(cur); err != nil {
		def := DefaultLoadedAllowlist()
		if err := EnsureRootDir(); err != nil {
			return 0, err
		}
		if err := WriteLoadedAllowlist(def); err != nil {
			return 0, err
		}
		return len(def.Commands), nil
	}
	def := DefaultLoadedAllowlist()
	outCmd := maps.Clone(cur.Commands)
	for k, p := range def.Commands {
		if _, ok := outCmd[k]; !ok {
			outCmd[k] = p
			added++
		}
	}
	out := &LoadedAllowlist{Version: AllowlistSchemaVersion, Commands: NormalizeReadOnlyCLIPolicies(outCmd)}
	if err := WriteLoadedAllowlist(out); err != nil {
		return 0, err
	}
	return added, nil
}
