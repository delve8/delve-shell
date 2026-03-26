package config

import (
	"os"
	"strings"

	"gopkg.in/yaml.v3"
)

// Config is the main app config (config.yaml; allowlist is allowlist.yaml, remotes is remotes.yaml).
type Config struct {
	// Language for UI, e.g. en, zh; default en
	Language string `yaml:"language"`
	// LLM API config; strings support $VAR or ${VAR} env expansion
	LLM LLMConfig `yaml:"llm"`
	// History retention policy
	History HistoryConfig `yaml:"history"`
	// AllowlistAutoRun: when true, allowlisted commands run without confirmation; when false, every command shows approval card (Run/Copy/Dismiss). Default true.
	AllowlistAutoRun *bool `yaml:"allowlist_auto_run,omitempty"`
}

// LLMConfig is the LLM API config.
type LLMConfig struct {
	BaseURL            string `yaml:"base_url,omitempty"`             // OpenAI-compatible API URL; empty = official; not written when default model
	APIKey             string `yaml:"api_key"`                        // API key; supports $VAR
	Model              string `yaml:"model,omitempty"`                // model name; empty = default gpt-4o-mini; not written when default
	SystemPrompt       string `yaml:"system_prompt"`                  // system prompt; empty = built-in default; supports $VAR and multiline
	MaxContextMessages int    `yaml:"max_context_messages,omitempty"` // max user+assistant messages to send as history; 0 = default 50; reduce for small-context models
	MaxContextChars    int    `yaml:"max_context_chars,omitempty"`    // approximate max chars for conversation history; 0 = no limit; helps avoid overflow on fixed-context models
}

// RemoteTarget is one named remote host that can be selected via /remote on.
type RemoteTarget struct {
	// Name is a short label like "dev" or "prod".
	Name string `yaml:"name"`
	// Target is SSH target like "user@host" or "user@host:22".
	Target string `yaml:"target"`
	// IdentityFile is an optional path to a private key, e.g. "~/.ssh/id_rsa".
	IdentityFile string `yaml:"identity_file,omitempty"`
}

// AllowlistEntry is one allowlist entry; Pattern is always a regex.
type AllowlistEntry struct {
	Pattern string `yaml:"pattern"`
}

// HistoryConfig is the history retention policy.
type HistoryConfig struct {
	MaxDays    int `yaml:"max_days"`    // keep last N days; 0 = no day-based cleanup
	MaxEntries int `yaml:"max_entries"` // max entries per session or global; 0 = no limit
}

// Load reads config from the default path. If file does not exist, writes default config to config.yaml and returns it.
// Remotes are stored in remotes.yaml; if config.yaml contains remotes (legacy), they are migrated to remotes.yaml on first load.
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
	// Decode with optional remotes for migration from old config.yaml.
	var file struct {
		Language         string         `yaml:"language"`
		Remotes          []RemoteTarget `yaml:"remotes,omitempty"`
		LLM              LLMConfig      `yaml:"llm"`
		History          HistoryConfig  `yaml:"history"`
		AllowlistAutoRun *bool          `yaml:"allowlist_auto_run,omitempty"`
	}
	if err := yaml.Unmarshal(data, &file); err != nil {
		return nil, err
	}
	// Migrate remotes from config.yaml to remotes.yaml if present and remotes.yaml does not exist.
	if len(file.Remotes) > 0 {
		_, errRemotes := os.Stat(RemotesPath())
		if errRemotes != nil && os.IsNotExist(errRemotes) {
			if err := EnsureRootDir(); err != nil {
				return nil, err
			}
			if err := WriteRemotes(file.Remotes); err != nil {
				return nil, err
			}
		}
	}
	c := &Config{
		Language:         file.Language,
		LLM:              file.LLM,
		History:          file.History,
		AllowlistAutoRun: file.AllowlistAutoRun,
	}
	return c, nil
}

// LoadEnsured ensures the config root directory exists, then loads config from config.yaml.
func LoadEnsured() (*Config, error) {
	if err := EnsureRootDir(); err != nil {
		return nil, err
	}
	return Load()
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

// AllowlistAutoRunResolved returns whether allowlisted commands run without confirmation. Default true.
func (c *Config) AllowlistAutoRunResolved() bool {
	if c.AllowlistAutoRun != nil {
		return *c.AllowlistAutoRun
	}
	return true
}

// ExpandEnv replaces $VAR and ${VAR} in s with env values (shell-compatible).
func ExpandEnv(s string) string {
	return os.Expand(s, func(key string) string { return os.Getenv(key) })
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
	if baseURL == "" && apiKey != "" {
		baseURL = "https://api.openai.com/v1"
	}
	return baseURL, apiKey, model
}

// DefaultMaxContextMessages is used when llm.max_context_messages is 0 or unset.
const DefaultMaxContextMessages = 50

// EnsureRootDir creates root and subdirs if missing (including skills dir for user-installed skills).
func EnsureRootDir() error {
	for _, dir := range []string{RootDir(), RulesDir(), HistoryDir(), SkillsDir()} {
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
