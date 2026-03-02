package config

import (
	"fmt"
	"os"
	"strings"

	"gopkg.in/yaml.v3"
)

// Config 应用配置（主配置 config.yaml；允许列表见 allowlist.yaml 与 LoadAllowlist）
type Config struct {
	// Language 界面语言，如 en、zh；默认 en
	Language string `yaml:"language"`
	// LLM 大模型 API 配置；字符串中支持 $VAR 或 ${VAR} 引用环境变量
	LLM LLMConfig `yaml:"llm"`
	// History 历史保留策略
	History HistoryConfig `yaml:"history"`
}

// LLMConfig LLM API 配置
type LLMConfig struct {
	BaseURL      string `yaml:"base_url,omitempty"`      // OpenAI 兼容 API 地址；空则用官方，默认模型时不写入
	APIKey       string `yaml:"api_key"`                // API Key，支持 $VAR
	Model        string `yaml:"model,omitempty"`        // 模型名；空则默认 gpt-4o-mini，默认时不写入
	SystemPrompt string `yaml:"system_prompt"`          // 系统提示词，空则使用代码内置默认；支持 $VAR 与多行
}

// AllowlistEntry 允许列表一条：Pattern 始终按正则匹配
type AllowlistEntry struct {
	Pattern string `yaml:"pattern"`
}

// HistoryConfig 历史保留策略
type HistoryConfig struct {
	MaxDays   int `yaml:"max_days"`   // 保留最近 N 天，0 表示不按天数清理
	MaxEntries int `yaml:"max_entries"` // 每个会话或全局最大条数，0 表示不限制
}

// Load 从默认路径加载配置。文件不存在时使用内置默认配置并写回 config.yaml，后续均从文件读取
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

// Default 返回默认配置（允许列表单独见 allowlist.yaml / LoadAllowlist）
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

// ExpandEnv 将 s 中的 $VAR 与 ${VAR} 替换为环境变量值（与 shell 一致）
func ExpandEnv(s string) string {
	return os.Expand(s, func(key string) string { return os.Getenv(key) })
}

// LLMSummary 返回当前 LLM 配置的只读摘要（api_key 脱敏），用于 /config show
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

// LLMResolved 返回展开环境变量后的 LLM 配置；用于实际请求。空 base_url 默认为 OpenAI 官方。
// 会对 base_url、api_key、model 做 TrimSpace，避免粘贴或配置时带入首尾空格导致 401。
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

// allowlistFile 允许列表文件结构（allowlist.yaml）
type allowlistFile struct {
	Allowlist []AllowlistEntry `yaml:"allowlist"`
}

// LoadAllowlist 从 allowlist.yaml 加载允许列表。文件不存在时写入默认并返回
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

// WriteAllowlist 将允许列表写回 allowlist.yaml（调用方在修改后使用；首次写入前应 EnsureRootDir）
func WriteAllowlist(entries []AllowlistEntry) error {
	data, err := yaml.Marshal(allowlistFile{Allowlist: entries})
	if err != nil {
		return err
	}
	return os.WriteFile(AllowlistPath(), data, 0600)
}

// DefaultAllowlist 返回内置默认允许列表（只读类命令）；供 /config allowlist update 等合并使用
func DefaultAllowlist() []AllowlistEntry {
	return defaultAllowlist()
}

// AllowlistUpdateWithDefaults 将当前允许列表与内置默认合并：保留已有条目，追加默认里尚未存在的 pattern。返回本次新增条数。
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

// defaultAllowlist 内置默认允许列表：只读类命令，不修改文件系统或系统状态；每条 Pattern 均为正则
func defaultAllowlist() []AllowlistEntry {
	return []AllowlistEntry{
		// 目录与路径
		{Pattern: `\bpwd\b`},
		{Pattern: `\bls\b`},
		{Pattern: `\bdir\b`}, // 部分环境别名
		// 用户与环境
		{Pattern: `\bwhoami\b`},
		{Pattern: `\bid\b`},
		{Pattern: `\benv\b`},
		{Pattern: `\bprintenv\b`},
		// 系统信息
		{Pattern: `\buname\b`},
		{Pattern: `\bhostname\b`},
		{Pattern: `\bdate\b`},
		// 命令查找
		{Pattern: `\bwhich\b`},
		{Pattern: `\bwhereis\b`},
		{Pattern: `\btype\b`},
		// 只读查看文件（cat/head/tail/less/more 仅读；注意 cat 可读任意文件）
		{Pattern: `\bcat\b`},
		{Pattern: `\bhead\b`},
		{Pattern: `\btail\b`},
		{Pattern: `\bless\b`},
		{Pattern: `\bmore\b`},
		// 文件信息与统计
		{Pattern: `\bfile\b`},
		{Pattern: `\bstat\b`},
		{Pattern: `\bwc\b`},
		// find：仅允许常见只读用法（-name/-type/-maxdepth），不含 -exec/-delete
		{Pattern: `find\s+\S+(\s+-(name|type|maxdepth|iname)\s+\S+)*\s*$`},
		// grep/egrep/fgrep：只读搜索
		{Pattern: `\bgrep\b`},
		{Pattern: `\begrep\b`},
		{Pattern: `\bfgrep\b`},
		// 其他只读
		{Pattern: `\btrue\b`},
		{Pattern: `\bfalse\b`},
		{Pattern: `\bseq\b`},
		// kubectl 只读子命令
		{Pattern: `kubectl\s+get\s`},
		{Pattern: `kubectl\s+describe\s`},
		{Pattern: `kubectl\s+logs\s`},
		{Pattern: `kubectl\s+top\s`},
		{Pattern: `kubectl\s+explain\s`},
		{Pattern: `kubectl\s+api-resources`},
		{Pattern: `kubectl\s+api-versions`},
		{Pattern: `kubectl\s+cluster-info(?!\s+dump)`}, // view 只读；dump 会写文件故排除
		{Pattern: `kubectl\s+config\s+view`},
		{Pattern: `kubectl\s+version`},
		{Pattern: `kubectl\s+auth\s+can-i`},
		{Pattern: `kubectl\s+auth\s+whoami`},
		{Pattern: `kubectl\s+rollout\s+status`},
		{Pattern: `kubectl\s+diff\s`},
		{Pattern: `kubectl\s+.*--help`},
		// git 只读命令
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
		// 其他 CLI help
		{Pattern: `docker\s+.*--help`},
		{Pattern: `\b--help\b`}, // 多数 GNU 工具 command --help
	}
}

// EnsureRootDir 确保根目录及子目录存在
func EnsureRootDir() error {
	for _, dir := range []string{RootDir(), RulesDir(), HistoryDir()} {
		if err := os.MkdirAll(dir, 0700); err != nil {
			return err
		}
	}
	return nil
}

// Write 将配置写回默认路径（调用方在修改配置后使用；首次写入前应 EnsureRootDir）
func Write(c *Config) error {
	data, err := yaml.Marshal(c)
	if err != nil {
		return err
	}
	return os.WriteFile(ConfigPath(), data, 0600)
}
