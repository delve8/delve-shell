package config

import (
	"bytes"
	"fmt"
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
	data, err := EncodeAllowlistYAML(ld)
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

