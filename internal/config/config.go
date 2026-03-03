package config

import (
	"fmt"
	"os"
	"strings"

	"gopkg.in/yaml.v3"
)

// Config is the main app config (config.yaml; allowlist is allowlist.yaml via LoadAllowlist).
type Config struct {
	// Language for UI, e.g. en, zh; default en
	Language string `yaml:"language"`
	// LLM API config; strings support $VAR or ${VAR} env expansion
	LLM LLMConfig `yaml:"llm"`
	// History retention policy
	History HistoryConfig `yaml:"history"`
}

// LLMConfig is the LLM API config.
type LLMConfig struct {
	BaseURL      string `yaml:"base_url,omitempty"`      // OpenAI-compatible API URL; empty = official; not written when default model
	APIKey       string `yaml:"api_key"`                 // API key; supports $VAR
	Model        string `yaml:"model,omitempty"`         // model name; empty = default gpt-4o-mini; not written when default
	SystemPrompt string `yaml:"system_prompt"`           // system prompt; empty = built-in default; supports $VAR and multiline
}

// AllowlistEntry is one allowlist entry; Pattern is always a regex.
type AllowlistEntry struct {
	Pattern string `yaml:"pattern"`
}

// sensitivePatternsFile is the structure of sensitive_patterns.yaml (regex patterns).
type sensitivePatternsFile struct {
	Patterns []string `yaml:"patterns"`
}

// HistoryConfig is the history retention policy.
type HistoryConfig struct {
	MaxDays    int `yaml:"max_days"`    // keep last N days; 0 = no day-based cleanup
	MaxEntries int `yaml:"max_entries"` // max entries per session or global; 0 = no limit
}

// Load reads config from the default path. If file does not exist, writes default config to config.yaml and returns it.
func Load() (*Config, error) {
	path := ConfigPath()
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			def := Default()
			if err := EnsureRootDir(); err != nil {
				return nil, err
			}
			if err := Write(def); err != nil {
				return nil, err
			}
			return def, nil
		}
		return nil, err
	}
	var c Config
	if err := yaml.Unmarshal(data, &c); err != nil {
		return nil, err
	}
	return &c, nil
}

// Default returns the default config (allowlist is separate: allowlist.yaml / LoadAllowlist).
func Default() *Config {
	return &Config{
		Language: "en",
		LLM:      LLMConfig{},
		History: HistoryConfig{
			MaxDays:    30,
			MaxEntries: 0,
		},
	}
}

// ExpandEnv replaces $VAR and ${VAR} in s with env values (shell-compatible).
func ExpandEnv(s string) string {
	return os.Expand(s, func(key string) string { return os.Getenv(key) })
}

// LLMSummary returns a read-only summary of current LLM config (api_key masked) for /config show.
func (c *Config) LLMSummary() string {
	baseURL := c.LLM.BaseURL
	model := c.LLM.Model
	key := c.LLM.APIKey
	if key != "" {
		if len(key) > 8 {
			key = key[:4] + "***" + key[len(key)-4:]
		} else {
			key = "***"
		}
	} else {
		key = "(not set)"
	}
	if baseURL == "" {
		baseURL = "(default)"
	}
	if model == "" {
		model = "gpt-4o-mini"
	}
	sp := c.LLM.SystemPrompt
	if sp == "" {
		sp = "(default)"
	} else {
		sp = "(custom, " + fmt.Sprintf("%d", len(sp)) + " chars)"
	}
	return "language: " + c.languageResolved() + "\nllm.base_url: " + baseURL + "\nllm.api_key: " + key + "\nllm.model: " + model + "\nllm.system_prompt: " + sp
}

func (c *Config) languageResolved() string {
	if c.Language != "" {
		return c.Language
	}
	return "en"
}

// LLMResolved returns LLM config with env vars expanded for actual requests. Empty base_url defaults to OpenAI.
// Trims base_url, api_key, model to avoid 401 from leading/trailing spaces when pasting.
func (c *Config) LLMResolved() (baseURL, apiKey, model string) {
	baseURL = strings.TrimSpace(ExpandEnv(c.LLM.BaseURL))
	baseURL = strings.TrimRight(baseURL, "/")
	apiKey = strings.TrimSpace(ExpandEnv(c.LLM.APIKey))
	model = strings.TrimSpace(ExpandEnv(c.LLM.Model))
	if model == "" {
		model = "gpt-4o-mini"
	}
	if baseURL == "" && apiKey != "" {
		baseURL = "https://api.openai.com/v1"
	}
	return baseURL, apiKey, model
}

// allowlistFile is the allowlist.yaml file structure.
type allowlistFile struct {
	Allowlist []AllowlistEntry `yaml:"allowlist"`
}

// LoadAllowlist loads allowlist from allowlist.yaml. If missing, writes default and returns it.
func LoadAllowlist() ([]AllowlistEntry, error) {
	path := AllowlistPath()
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			def := defaultAllowlist()
			if err := EnsureRootDir(); err != nil {
				return nil, err
			}
			if err := WriteAllowlist(def); err != nil {
				return nil, err
			}
			return def, nil
		}
		return nil, err
	}
	var f allowlistFile
	if err := yaml.Unmarshal(data, &f); err != nil {
		return nil, err
	}
	return f.Allowlist, nil
}

// WriteAllowlist writes the allowlist to allowlist.yaml (call after changes; EnsureRootDir before first write).
func WriteAllowlist(entries []AllowlistEntry) error {
	data, err := yaml.Marshal(allowlistFile{Allowlist: entries})
	if err != nil {
		return err
	}
	return os.WriteFile(AllowlistPath(), data, 0600)
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

// DefaultAllowlist returns the built-in default allowlist (read-only commands); used by /config allowlist update etc.
func DefaultAllowlist() []AllowlistEntry {
	return defaultAllowlist()
}

// AllowlistUpdateWithDefaults merges current allowlist with built-in default: keep existing, add missing patterns. Returns number added.
func AllowlistUpdateWithDefaults() (added int, err error) {
	path := AllowlistPath()
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			if err := EnsureRootDir(); err != nil {
				return 0, err
			}
			def := defaultAllowlist()
			if err := WriteAllowlist(def); err != nil {
				return 0, err
			}
			return len(def), nil
		}
		return 0, err
	}
	var f allowlistFile
	if err := yaml.Unmarshal(data, &f); err != nil {
		return 0, err
	}
	have := make(map[string]bool)
	for _, e := range f.Allowlist {
		have[e.Pattern] = true
	}
	out := f.Allowlist
	for _, e := range defaultAllowlist() {
		if !have[e.Pattern] {
			out = append(out, e)
			have[e.Pattern] = true
			added++
		}
	}
	if added == 0 {
		return 0, nil
	}
	if err := WriteAllowlist(out); err != nil {
		return 0, err
	}
	return added, nil
}

// defaultAllowlist is the built-in default: read-only commands; each Pattern is a regex.
func defaultAllowlist() []AllowlistEntry {
	return []AllowlistEntry{
		// dirs and paths
		{Pattern: `\bpwd\b`},
		{Pattern: `\bls\b`},
		{Pattern: `\bdir\b`}, // some envs alias
		// user and env
		{Pattern: `\bwhoami\b`},
		{Pattern: `\bid\b`},
		{Pattern: `\benv\b`},
		{Pattern: `\bprintenv\b`},
		// system info
		{Pattern: `\buname\b`},
		{Pattern: `\bhostname\b`},
		{Pattern: `\bdate\b`},
		// command lookup
		{Pattern: `\bwhich\b`},
		{Pattern: `\bwhereis\b`},
		{Pattern: `\btype\b`},
		// read-only file view (cat/head/tail/less/more read-only; cat can read any file)
		{Pattern: `\bcat\b`},
		{Pattern: `\bhead\b`},
		{Pattern: `\btail\b`},
		{Pattern: `\bless\b`},
		{Pattern: `\bmore\b`},
		// file info and stats
		{Pattern: `\bfile\b`},
		{Pattern: `\bstat\b`},
		{Pattern: `\bwc\b`},
		// checksum and encoding (read-only)
		{Pattern: `\bmd5sum\b`},
		{Pattern: `\bsha256sum\b`},
		{Pattern: `\bsha1sum\b`},
		{Pattern: `\bshasum\b`},   // macOS
		{Pattern: `\bbase64\b`},
		{Pattern: `\bcksum\b`},
		// find: common read-only usage only (-name/-type/-maxdepth), no -exec/-delete
		{Pattern: `find\s+\S+(\s+-(name|type|maxdepth|iname)\s+\S+)*\s*$`},
		// grep/egrep/fgrep: read-only search
		{Pattern: `\bgrep\b`},
		{Pattern: `\begrep\b`},
		{Pattern: `\bfgrep\b`},
		// output and pipes (read-only)
		{Pattern: `\becho\b`},
		{Pattern: `\bprintf\b`},
		// text compare and process (read-only, no file write)
		{Pattern: `\bdiff\b`},
		{Pattern: `\bcmp\b`},
		{Pattern: `\bcut\b`},
		{Pattern: `\btr\b`},
		{Pattern: `\buniq\b`},
		{Pattern: `\bnl\b`},
		{Pattern: `\bcolumn\b`},
		{Pattern: `\bod\b`},
		{Pattern: `\bxxd\b`},
		{Pattern: `\bhexdump\b`},
		// decompress to stdout (read-only)
		{Pattern: `\bzcat\b`},
		{Pattern: `\bbzcat\b`},
		{Pattern: `\bxzcat\b`},
		// process and system resources (read-only)
		{Pattern: `\bps\b`},
		{Pattern: `\buptime\b`},
		{Pattern: `\bdf\b`},
		{Pattern: `\bdu\b`},
		{Pattern: `\bfree\b`},
		{Pattern: `\blsblk\b`},
		// user and permissions (read-only)
		{Pattern: `\bgroups\b`},
		{Pattern: `\bgetent\b`},
		{Pattern: `\blocale\b`},
		// network read-only (DNS, connectivity)
		{Pattern: `\bping\b`},
		{Pattern: `\bnslookup\b`},
		{Pattern: `\bdig\b`},
		{Pattern: `\bhost\b`},
		// other read-only
		{Pattern: `\btrue\b`},
		{Pattern: `\bfalse\b`},
		{Pattern: `\bseq\b`},
		{Pattern: `\bsleep\b`},
		// kubectl read-only subcommands
		{Pattern: `kubectl\s+get\s`},
		{Pattern: `kubectl\s+describe\s`},
		{Pattern: `kubectl\s+logs\s`},
		{Pattern: `kubectl\s+top\s`},
		{Pattern: `kubectl\s+explain\s`},
		{Pattern: `kubectl\s+api-resources`},
		{Pattern: `kubectl\s+api-versions`},
		{Pattern: `kubectl\s+cluster-info(?!\s+dump)`}, // view read-only; dump writes so excluded
		{Pattern: `kubectl\s+config\s+view`},
		{Pattern: `kubectl\s+version`},
		{Pattern: `kubectl\s+auth\s+can-i`},
		{Pattern: `kubectl\s+auth\s+whoami`},
		{Pattern: `kubectl\s+rollout\s+status`},
		{Pattern: `kubectl\s+diff\s`},
		{Pattern: `kubectl\s+.*--help`},
		// git read-only commands
		{Pattern: `git\s+status\s`},
		{Pattern: `git\s+status\s*$`},
		{Pattern: `git\s+diff\s`},
		{Pattern: `git\s+diff\s*$`},
		{Pattern: `git\s+log\s`},
		{Pattern: `git\s+log\s*$`},
		{Pattern: `git\s+show\s`},
		{Pattern: `git\s+show\s*$`},
		{Pattern: `git\s+branch(?:\s+-(?:a|v|r)|\s+--(?:list|show-current))?(?:\s|$)`},
		{Pattern: `git\s+tag(?:\s+-(?:l|list)|\s+--list)(?:\s|$)`},
		{Pattern: `git\s+tag\s*$`},
		{Pattern: `git\s+remote(?:\s+-(?:v)|\s+show)(?:\s|$)`},
		{Pattern: `git\s+config\s+(?:--get|--list|--get-all)(?:\s|$)`},
		{Pattern: `git\s+rev-parse(?:\s|$)`},
		{Pattern: `git\s+describe(?:\s|$)`},
		{Pattern: `git\s+stash\s+list(?:\s|$)`},
		{Pattern: `git\s+reflog\s`},
		{Pattern: `git\s+reflog\s*$`},
		{Pattern: `git\s+blame\s`},
		{Pattern: `git\s+ls-files\s`},
		{Pattern: `git\s+ls-tree\s`},
		{Pattern: `git\s+cat-file\s`},
		{Pattern: `git\s+for-each-ref\s`},
		{Pattern: `git\s+symbolic-ref\s`},
		{Pattern: `git\s+help\s`},
		{Pattern: `git\s+version\s*$`},
		{Pattern: `git\s+--help`},
		// other CLI help
		{Pattern: `docker\s+.*--help`},
		{Pattern: `\b--help\b`}, // most GNU tools: command --help
	}
}

// EnsureRootDir creates root and subdirs if missing.
func EnsureRootDir() error {
	for _, dir := range []string{RootDir(), RulesDir(), HistoryDir()} {
		if err := os.MkdirAll(dir, 0700); err != nil {
			return err
		}
	}
	return nil
}

// Write writes config to the default path (call after modifying; EnsureRootDir before first write).
func Write(c *Config) error {
	data, err := yaml.Marshal(c)
	if err != nil {
		return err
	}
	return os.WriteFile(ConfigPath(), data, 0600)
}
