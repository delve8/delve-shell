package config

import (
	"fmt"
	"os"
	"strings"

	"gopkg.in/yaml.v3"
)

// Config 应用配置（主配置 config.yaml；白名单见 whitelist.yaml 与 LoadWhitelist）
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

// WhitelistEntry 白名单一条：Pattern 为正则或普通字符串，IsRegex 为 true 时按正则匹配
type WhitelistEntry struct {
	Pattern string `yaml:"pattern"`
	IsRegex bool   `yaml:"is_regex"`
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

// Default 返回默认配置（白名单单独见 whitelist.yaml / LoadWhitelist）
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

// LLMResolved 返回展开环境变量后的 LLM 配置；用于实际请求。空 base_url 默认为 OpenAI 官方
func (c *Config) LLMResolved() (baseURL, apiKey, model string) {
	baseURL = strings.TrimRight(ExpandEnv(c.LLM.BaseURL), "/")
	apiKey = ExpandEnv(c.LLM.APIKey)
	model = ExpandEnv(c.LLM.Model)
	if model == "" {
		model = "gpt-4o-mini"
	}
	if baseURL == "" && apiKey != "" {
		baseURL = "https://api.openai.com/v1"
	}
	return baseURL, apiKey, model
}

// whitelistFile 白名单文件结构（whitelist.yaml）
type whitelistFile struct {
	Whitelist []WhitelistEntry `yaml:"whitelist"`
}

// LoadWhitelist 从 whitelist.yaml 加载白名单。文件不存在时写入默认白名单并返回
func LoadWhitelist() ([]WhitelistEntry, error) {
	path := WhitelistPath()
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			def := defaultWhitelist()
			if err := EnsureRootDir(); err != nil {
				return nil, err
			}
			if err := WriteWhitelist(def); err != nil {
				return nil, err
			}
			return def, nil
		}
		return nil, err
	}
	var f whitelistFile
	if err := yaml.Unmarshal(data, &f); err != nil {
		return nil, err
	}
	return f.Whitelist, nil
}

// WriteWhitelist 将白名单写回 whitelist.yaml（调用方在修改后使用；首次写入前应 EnsureRootDir）
func WriteWhitelist(entries []WhitelistEntry) error {
	data, err := yaml.Marshal(whitelistFile{Whitelist: entries})
	if err != nil {
		return err
	}
	return os.WriteFile(WhitelistPath(), data, 0600)
}

// defaultWhitelist 内置默认白名单：只读类命令，不修改文件系统或系统状态
func defaultWhitelist() []WhitelistEntry {
	return []WhitelistEntry{
		// 目录与路径
		{Pattern: `\bpwd\b`, IsRegex: true},
		{Pattern: `\bls\b`, IsRegex: true},
		{Pattern: `\bdir\b`, IsRegex: true}, // 部分环境别名
		// 用户与环境
		{Pattern: `\bwhoami\b`, IsRegex: true},
		{Pattern: `\bid\b`, IsRegex: true},
		{Pattern: `\benv\b`, IsRegex: true},
		{Pattern: `\bprintenv\b`, IsRegex: true},
		// 系统信息
		{Pattern: `\buname\b`, IsRegex: true},
		{Pattern: `\bhostname\b`, IsRegex: true},
		{Pattern: `\bdate\b`, IsRegex: true},
		// 命令查找
		{Pattern: `\bwhich\b`, IsRegex: true},
		{Pattern: `\bwhereis\b`, IsRegex: true},
		{Pattern: `\btype\b`, IsRegex: true},
		// 只读查看文件（cat/head/tail/less/more 仅读；注意 cat 可读任意文件）
		{Pattern: `\bcat\b`, IsRegex: true},
		{Pattern: `\bhead\b`, IsRegex: true},
		{Pattern: `\btail\b`, IsRegex: true},
		{Pattern: `\bless\b`, IsRegex: true},
		{Pattern: `\bmore\b`, IsRegex: true},
		// 文件信息与统计
		{Pattern: `\bfile\b`, IsRegex: true},
		{Pattern: `\bstat\b`, IsRegex: true},
		{Pattern: `\bwc\b`, IsRegex: true},
		// find：仅允许常见只读用法（-name/-type/-maxdepth），不含 -exec/-delete
		{Pattern: `find\s+\S+(\s+-(name|type|maxdepth|iname)\s+\S+)*\s*$`, IsRegex: true},
		// grep/egrep/fgrep：只读搜索
		{Pattern: `\bgrep\b`, IsRegex: true},
		{Pattern: `\begrep\b`, IsRegex: true},
		{Pattern: `\bfgrep\b`, IsRegex: true},
		// 其他只读
		{Pattern: `\btrue\b`, IsRegex: true},
		{Pattern: `\bfalse\b`, IsRegex: true},
		{Pattern: `\bseq\b`, IsRegex: true},
		// kubectl 只读子命令
		{Pattern: `kubectl\s+get\s`, IsRegex: true},
		{Pattern: `kubectl\s+describe\s`, IsRegex: true},
		{Pattern: `kubectl\s+logs\s`, IsRegex: true},
		{Pattern: `kubectl\s+top\s`, IsRegex: true},
		{Pattern: `kubectl\s+explain\s`, IsRegex: true},
		{Pattern: `kubectl\s+api-resources`, IsRegex: true},
		{Pattern: `kubectl\s+api-versions`, IsRegex: true},
		{Pattern: `kubectl\s+cluster-info(?!\s+dump)`, IsRegex: true}, // view 只读；dump 会写文件故排除
		{Pattern: `kubectl\s+config\s+view`, IsRegex: true},
		{Pattern: `kubectl\s+version`, IsRegex: true},
		{Pattern: `kubectl\s+auth\s+can-i`, IsRegex: true},
		{Pattern: `kubectl\s+auth\s+whoami`, IsRegex: true},
		{Pattern: `kubectl\s+rollout\s+status`, IsRegex: true},
		{Pattern: `kubectl\s+diff\s`, IsRegex: true},
		{Pattern: `kubectl\s+.*--help`, IsRegex: true},
		// 其他 CLI help
		{Pattern: `docker\s+.*--help`, IsRegex: true},
		{Pattern: `git\s+--help`, IsRegex: true},
		{Pattern: `\b--help\b`, IsRegex: true}, // 多数 GNU 工具 command --help
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
