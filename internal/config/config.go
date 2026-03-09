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
	// AllowlistAutoRun: when true, allowlisted commands run without confirmation; when false, every command shows approval card (Run/Copy/Dismiss). Default true.
	AllowlistAutoRun *bool `yaml:"allowlist_auto_run,omitempty"`
	// Mode: deprecated, use AllowlistAutoRun. suggest -> false, run -> true. Kept for reading old config.
	Mode string `yaml:"mode,omitempty"`
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
		Mode: "run",
	}
}

// AllowlistAutoRunResolved returns whether allowlisted commands run without confirmation. Default true.
// When AllowlistAutoRun is unset, migrates from Mode (run->true, suggest->false).
func (c *Config) AllowlistAutoRunResolved() bool {
	if c.AllowlistAutoRun != nil {
		return *c.AllowlistAutoRun
	}
	s := strings.TrimSpace(strings.ToLower(c.Mode))
	return s != "suggest"
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
	autoRun := "List Only"
	if !c.AllowlistAutoRunResolved() {
		autoRun = "Disabled"
	}
	return "language: " + c.languageResolved() + "\nallowlist_auto_run: " + autoRun + "\nllm.base_url: " + baseURL + "\nllm.api_key: " + key + "\nllm.model: " + model + "\nllm.system_prompt: " + sp
}

// ModeResolved returns "suggest" or "run" for backward compatibility (e.g. migration). Prefer AllowlistAutoRunResolved().
func (c *Config) ModeResolved() string {
	if c.AllowlistAutoRunResolved() {
		return "run"
	}
	return "suggest"
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

// oldAllowlistWords are single-command names that used to use \bword\b or (^|\s)word\b; migration removes those so (^|\s)word(\s|$) are the only ones.
var oldAllowlistWords = []string{
	"pwd", "ls", "dir", "whoami", "id", "env", "printenv", "uname", "hostname", "date", "which", "whereis", "type",
	"cat", "head", "tail", "less", "more", "file", "stat", "wc", "md5sum", "sha256sum", "sha1sum", "shasum", "base64", "cksum",
	"grep", "egrep", "fgrep", "echo", "printf", "diff", "cmp", "cut", "tr", "uniq", "nl", "column", "od", "xxd", "hexdump",
	"zcat", "bzcat", "xzcat", "ps", "uptime", "df", "du", "free", "lsblk", "groups", "getent", "locale",
	"ping", "nslookup", "dig", "host", "true", "false", "seq", "sleep", "--help",
}

func buildOldLoosePatternsMap() map[string]bool {
	m := make(map[string]bool, len(oldAllowlistWords)*2)
	for _, w := range oldAllowlistWords {
		m[`\b`+w+`\b`] = true
		m[`(^|\s)`+w+`\b`] = true
	}
	return m
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
	oldLoosePatterns := buildOldLoosePatternsMap()
	outFiltered := out[:0]
	for _, e := range out {
		if !oldLoosePatterns[e.Pattern] {
			outFiltered = append(outFiltered, e)
		}
	}
	out = outFiltered
	for _, e := range defaultAllowlist() {
		if !have[e.Pattern] {
			out = append(out, e)
			have[e.Pattern] = true
			added++
		}
	}
	needWrite := added > 0 || len(out) < len(f.Allowlist)
	if !needWrite {
		return 0, nil
	}
	if err := WriteAllowlist(out); err != nil {
		return 0, err
	}
	return added, nil
}

// defaultAllowlist is the built-in default: read-only commands; each Pattern is a regex.
// Single-command patterns use (^|\s)word(\s|$) so the word is only matched as command name:
// left side must be start or space (not -word option); right side must be space or end (not word-xxx).
func defaultAllowlist() []AllowlistEntry {
	return []AllowlistEntry{
		// dirs and paths
		{Pattern: `(^|\s)pwd(\s|$)`},
		{Pattern: `(^|\s)ls(\s|$)`},
		{Pattern: `(^|\s)dir(\s|$)`}, // some envs alias
		// user and env
		{Pattern: `(^|\s)whoami(\s|$)`},
		{Pattern: `(^|\s)id(\s|$)`},
		{Pattern: `(^|\s)env(\s|$)`},
		{Pattern: `(^|\s)printenv(\s|$)`},
		// system info
		{Pattern: `(^|\s)uname(\s|$)`},
		{Pattern: `(^|\s)hostname(\s|$)`},
		{Pattern: `(^|\s)date(\s|$)`},
		// command lookup
		{Pattern: `(^|\s)which(\s|$)`},
		{Pattern: `(^|\s)whereis(\s|$)`},
		{Pattern: `(^|\s)type(\s|$)`},
		// read-only file view (cat/head/tail/less/more read-only; cat can read any file)
		{Pattern: `(^|\s)cat(\s|$)`},
		{Pattern: `(^|\s)head(\s|$)`},
		{Pattern: `(^|\s)tail(\s|$)`},
		{Pattern: `(^|\s)less(\s|$)`},
		{Pattern: `(^|\s)more(\s|$)`},
		// file info and stats
		{Pattern: `(^|\s)file(\s|$)`},
		{Pattern: `(^|\s)stat(\s|$)`},
		{Pattern: `(^|\s)wc(\s|$)`},
		// checksum and encoding (read-only)
		{Pattern: `(^|\s)md5sum(\s|$)`},
		{Pattern: `(^|\s)sha256sum(\s|$)`},
		{Pattern: `(^|\s)sha1sum(\s|$)`},
		{Pattern: `(^|\s)shasum(\s|$)`},   // macOS
		{Pattern: `(^|\s)base64(\s|$)`},
		{Pattern: `(^|\s)cksum(\s|$)`},
		// find: common read-only usage only (-name/-type/-maxdepth), no -exec/-delete
		{Pattern: `find\s+\S+(\s+-(name|type|maxdepth|iname)\s+\S+)*\s*$`},
		// grep/egrep/fgrep: read-only search
		{Pattern: `(^|\s)grep(\s|$)`},
		{Pattern: `(^|\s)egrep(\s|$)`},
		{Pattern: `(^|\s)fgrep(\s|$)`},
		// output and pipes (read-only)
		{Pattern: `(^|\s)echo(\s|$)`},
		{Pattern: `(^|\s)printf(\s|$)`},
		// text compare and process (read-only, no file write)
		{Pattern: `(^|\s)diff(\s|$)`},
		{Pattern: `(^|\s)cmp(\s|$)`},
		{Pattern: `(^|\s)cut(\s|$)`},
		{Pattern: `(^|\s)tr(\s|$)`},
		{Pattern: `(^|\s)uniq(\s|$)`},
		{Pattern: `(^|\s)nl(\s|$)`},
		{Pattern: `(^|\s)column(\s|$)`},
		{Pattern: `(^|\s)od(\s|$)`},
		{Pattern: `(^|\s)xxd(\s|$)`},
		{Pattern: `(^|\s)hexdump(\s|$)`},
		// decompress to stdout (read-only)
		{Pattern: `(^|\s)zcat(\s|$)`},
		{Pattern: `(^|\s)bzcat(\s|$)`},
		{Pattern: `(^|\s)xzcat(\s|$)`},
		// process and system resources (read-only)
		{Pattern: `(^|\s)ps(\s|$)`},
		{Pattern: `(^|\s)uptime(\s|$)`},
		{Pattern: `(^|\s)df(\s|$)`},
		{Pattern: `(^|\s)du(\s|$)`},
		{Pattern: `(^|\s)free(\s|$)`},
		{Pattern: `(^|\s)lsblk(\s|$)`},
		// user and permissions (read-only)
		{Pattern: `(^|\s)groups(\s|$)`},
		{Pattern: `(^|\s)getent(\s|$)`},
		{Pattern: `(^|\s)locale(\s|$)`},
		// network read-only (DNS, connectivity)
		{Pattern: `(^|\s)ping(\s|$)`},
		{Pattern: `(^|\s)nslookup(\s|$)`},
		{Pattern: `(^|\s)dig(\s|$)`},
		{Pattern: `(^|\s)host(\s|$)`},
		// other read-only
		{Pattern: `(^|\s)true(\s|$)`},
		{Pattern: `(^|\s)false(\s|$)`},
		{Pattern: `(^|\s)seq(\s|$)`},
		{Pattern: `(^|\s)sleep(\s|$)`},
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
		{Pattern: `(^|\s)--help(\s|$)`}, // most GNU tools: command --help
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
