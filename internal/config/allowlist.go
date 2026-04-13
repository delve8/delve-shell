package config

import (
	"bytes"
	"fmt"
	"os"

	"gopkg.in/yaml.v3"
)

// LoadAllowlist loads the built-in allowlist.yaml plus the user-editable allowlist_custom.yaml overlay.
// The built-in file is repaired to defaults when missing/invalid; the custom overlay is created empty when missing.
func LoadAllowlist() (*LoadedAllowlist, error) {
	base, err := loadBuiltInAllowlistFile()
	if err != nil {
		return nil, err
	}
	custom, err := LoadCustomAllowlist()
	if err != nil {
		return nil, err
	}
	for name, pol := range custom.Commands {
		base.Commands[name] = pol
	}
	base.Commands = NormalizeReadOnlyCLIPolicies(base.Commands)
	return base, nil
}

func loadBuiltInAllowlistFile() (*LoadedAllowlist, error) {
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

// EmptyCustomLoadedAllowlist returns an empty custom allowlist overlay.
func EmptyCustomLoadedAllowlist() *LoadedAllowlist {
	return &LoadedAllowlist{
		Version:  AllowlistSchemaVersion,
		Commands: map[string]ReadOnlyCLIPolicy{},
	}
}

// LoadCustomAllowlist loads allowlist_custom.yaml. Missing file is created as an empty overlay.
func LoadCustomAllowlist() (*LoadedAllowlist, error) {
	path := CustomAllowlistPath()
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			empty := EmptyCustomLoadedAllowlist()
			if err := EnsureRootDir(); err != nil {
				return nil, err
			}
			if err := WriteLoadedAllowlistToPath(path, empty); err != nil {
				return nil, err
			}
			return empty, nil
		}
		return nil, err
	}
	ld, err := ParseAllowlistYAML(data)
	if err != nil {
		return nil, err
	}
	if err := ValidateLoadedAllowlistAllowEmpty(ld); err != nil {
		return nil, err
	}
	ld.Commands = NormalizeReadOnlyCLIPolicies(ld.Commands)
	return ld, nil
}

// EncodeAllowlistYAML encodes the allowlist as YAML with 2-space indentation.
func EncodeAllowlistYAML(ld *LoadedAllowlist) ([]byte, error) {
	if ld == nil {
		return nil, fmt.Errorf("allowlist: nil document")
	}
	var buf bytes.Buffer
	enc := yaml.NewEncoder(&buf)
	enc.SetIndent(2)
	if err := enc.Encode(ld); err != nil {
		return nil, err
	}
	if err := enc.Close(); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

// WriteLoadedAllowlist writes allowlist.yaml (EnsureRootDir before first write).
func WriteLoadedAllowlist(ld *LoadedAllowlist) error {
	if ld == nil {
		ld = DefaultLoadedAllowlist()
	}
	return WriteLoadedAllowlistToPath(AllowlistPath(), ld)
}

// WriteLoadedAllowlistToPath writes an allowlist document to path.
func WriteLoadedAllowlistToPath(path string, ld *LoadedAllowlist) error {
	if ld == nil {
		ld = EmptyCustomLoadedAllowlist()
	}
	data, err := EncodeAllowlistYAML(ld)
	if err != nil {
		return err
	}
	return os.WriteFile(path, data, 0600)
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
