package config

import (
	"os"

	"gopkg.in/yaml.v3"
)

// sensitivePatternsFile is the structure of sensitive_patterns.yaml (regex patterns).
type sensitivePatternsFile struct {
	Patterns []string `yaml:"patterns"`
}

// LoadSensitivePatterns loads regex patterns from sensitive_patterns.yaml.
// If the file does not exist, writes default patterns (same as DefaultSensitivePatterns) and returns them.
func LoadSensitivePatterns() ([]string, error) {
	path := SensitivePatternsPath()
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			def := defaultSensitivePatterns()
			if err := EnsureRootDir(); err != nil {
				return nil, err
			}
			if err := WriteSensitivePatterns(def); err != nil {
				return nil, err
			}
			return def, nil
		}
		return nil, err
	}
	var f sensitivePatternsFile
	if err := yaml.Unmarshal(data, &f); err != nil {
		return nil, err
	}
	return f.Patterns, nil
}

// WriteSensitivePatterns writes patterns to sensitive_patterns.yaml.
func WriteSensitivePatterns(patterns []string) error {
	data, err := yaml.Marshal(sensitivePatternsFile{Patterns: patterns})
	if err != nil {
		return err
	}
	return os.WriteFile(SensitivePatternsPath(), data, 0600)
}

// DefaultSensitivePatterns returns the built-in default patterns (sensitive paths); visible to the user in sensitive_patterns.yaml.
func DefaultSensitivePatterns() []string {
	return defaultSensitivePatterns()
}

// defaultSensitivePatterns is the built-in list of path patterns; only applied when the command verb is a content reader (see DefaultContentReaderCommands).
func defaultSensitivePatterns() []string {
	return []string{
		`/etc/shadow\b`,
		`/etc/gpg/`,
		`\.env\b`,
		`\b\.env\s`,
		`\s\.env\s*$`,
		`\.pem\b`,
		`\b\.pem\s`,
		`\s\.pem\s*$`,
		`\.key\b`,
		`\b\.key\s`,
		`\s\.key\s*$`,
		`id_rsa\b`,
		`id_ed25519\b`,
		`id_ecdsa\b`,
		`\.aws/credentials\b`,
		`\b\.aws/credentials\b`,
		`/secrets?/`,
		`\bsecrets?/`,
	}
}

// SensitivePatternsUpdateWithDefaults merges current sensitive_patterns.yaml with built-in default: keep existing, add missing patterns. Returns number added.
func SensitivePatternsUpdateWithDefaults() (added int, err error) {
	path := SensitivePatternsPath()
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			if err := EnsureRootDir(); err != nil {
				return 0, err
			}
			def := defaultSensitivePatterns()
			if err := WriteSensitivePatterns(def); err != nil {
				return 0, err
			}
			return len(def), nil
		}
		return 0, err
	}
	var f sensitivePatternsFile
	if err := yaml.Unmarshal(data, &f); err != nil {
		return 0, err
	}
	have := make(map[string]bool)
	for _, p := range f.Patterns {
		have[p] = true
	}
	out := f.Patterns
	for _, p := range defaultSensitivePatterns() {
		if !have[p] {
			out = append(out, p)
			have[p] = true
			added++
		}
	}
	if added == 0 {
		return 0, nil
	}
	if err := WriteSensitivePatterns(out); err != nil {
		return 0, err
	}
	return added, nil
}
