package i18n

import "fmt"

// Msg keys for user-facing strings. Code error messages stay in English in callers.
const (
	KeyHelpText            = "help_text"
	KeyNoRequestInProgress = "no_request_in_progress"
	KeyUsageRun            = "usage_run"
	KeyUnknownCmd          = "unknown_cmd"
	KeyConfigReloaded      = "config_reloaded"
	KeyCancelled           = "cancelled"
	KeyErrorPrefix         = "error_prefix"
	KeyConfigPrefix        = "config_prefix"
	KeyConfigUnknownField   = "config_unknown_field"
	KeyConfigLanguageRequired = "config_language_required"
	KeyConfigSaved         = "config_saved"
	KeyConfigSavedLanguage  = "config_saved_language"
	KeyWaitOrCancel        = "wait_or_cancel"
	KeyPlaceholderInput    = "placeholder_input"
	KeyTitleHeader         = "title_header"
	KeyApprovalPrompt      = "approval_prompt"
	KeyApproveYN           = "approve_yn"
	KeyRunTagApproved      = "run_tag_approved"
	KeyRunTagDirect        = "run_tag_direct"
	KeyRunTagWhitelist     = "run_tag_whitelist"
	KeyResultSensitive     = "result_sensitive"
	KeyErrLLMNotConfigured = "err_llm_not_configured"
	KeyUserLabel           = "user_label"
	KeyAILabel             = "ai_label"
	KeyRunLabel            = "run_label"
	// Slash option descriptions (cmd as suffix for consistency)
	KeyDescExit   = "desc_exit"
	KeyDescRun    = "desc_run"
	KeyDescSh     = "desc_sh"
	KeyDescCancel = "desc_cancel"
	KeyDescConfig = "desc_config"
	KeyDescReload = "desc_reload"
	KeyDescHelp   = "desc_help"
	// /config 子项说明（仅在选择 /config 后显示）
	KeyDescConfigShow       = "desc_config_show"
	KeyDescConfigLLMBaseURL = "desc_config_llm_base_url"
	KeyDescConfigLLMApiKey  = "desc_config_llm_api_key"
	KeyDescConfigLLMModel   = "desc_config_llm_model"
	KeyDescConfigLanguage   = "desc_config_language"
)

var messages = map[string]map[string]string{
	"en": {
		KeyHelpText: `delve-shell — AI-assisted ops, commands run after your approval.

Slash commands:
  /exit          Quit
  /run <cmd>     Run a command directly (no AI)
  /sh            Spawn bash; return here when done
  /cancel        Cancel current AI request
  /config        Set or show config: /config llm base_url <url>, /config llm api_key <key>, /config llm model <name>, /config show, /config language <en|zh>
  /reload        Reload config and whitelist (no restart)
  /help          Show this help

Scroll: Up/Down, PgUp/PgDown. Text selection: use terminal mouse (no mouse reporting).`,
		KeyNoRequestInProgress: "(No request in progress)",
		KeyUsageRun:            "Usage: /run <command>",
		KeyUnknownCmd:          "Unknown command. Use /exit, /run <cmd>, /sh, /cancel, /config, /reload, /help",
		KeyConfigReloaded:      "Config and whitelist reloaded. Next message will use new config.",
		KeyCancelled:           "(Cancelled)",
		KeyErrorPrefix:         "Error: ",
		KeyConfigPrefix:        "Config: ",
		KeyConfigUnknownField:   "unknown field ",
		KeyConfigLanguageRequired: "language: value required (e.g. en, zh)",
		KeyConfigSaved:         "Config saved (llm.%s).",
		KeyConfigSavedLanguage:  "Config saved (language: %s).",
		KeyWaitOrCancel:        "(Please wait for the current response, or /cancel)",
		KeyPlaceholderInput:    "Type a command or / for slash commands...",
		KeyTitleHeader:         "delve-shell — Enter to send, ctrl+c to quit | Up/Down/PgUp/PgDown scroll",
		KeyApprovalPrompt:      "Command to run (approval required):",
		KeyApproveYN:           "Approve? (y/n): ",
		KeyRunTagApproved:      "approved",
		KeyRunTagDirect:        "direct",
		KeyRunTagWhitelist:     "whitelist",
		KeyResultSensitive:     "(Result contains sensitive data; not stored in history.)",
		KeyErrLLMNotConfigured: "LLM not configured. Use /config to set llm.api_key (and llm.base_url, llm.model), then send a message again (no restart needed). Supports $VAR or ${VAR} for env. Config path: %s",
		KeyUserLabel:           "User: ",
		KeyAILabel:             "AI: ",
		KeyRunLabel:            "Run: ",
		KeyDescExit:            "Quit delve-shell",
		KeyDescRun:             "Run a command directly (no AI)",
		KeyDescSh:              "Spawn bash; return here when done",
		KeyDescCancel:          "Cancel current AI request",
		KeyDescConfig:          "Set or show config (e.g. /config llm base_url <url>, /config language en)",
		KeyDescReload:          "Reload config and whitelist (no restart)",
		KeyDescHelp:            "Show this help",
		KeyDescConfigShow:      "Show current config path and LLM summary",
		KeyDescConfigLLMBaseURL: "Set LLM API base URL",
		KeyDescConfigLLMApiKey:  "Set LLM API key",
		KeyDescConfigLLMModel:   "Set LLM model name",
		KeyDescConfigLanguage:   "Set UI language (en, zh)",
	},
	"zh": {
		KeyHelpText: `delve-shell — AI 辅助运维，命令经你确认后执行。

斜杠命令：
  /exit          退出
  /run <cmd>     直接执行命令（不经 AI）
  /sh            启动 bash；结束后返回
  /cancel        取消当前 AI 请求
  /config        设置或查看配置：/config llm base_url <url>、/config llm api_key <key>、/config llm model <name>、/config show、/config language <en|zh>
  /reload        重载配置与白名单（无需重启）
  /help          显示此帮助

滚动：Up/Down、PgUp/PgDown。文本选择：使用终端鼠标（无需 mouse reporting）。`,
		KeyNoRequestInProgress: "（当前无进行中的请求）",
		KeyUsageRun:            "用法：/run <命令>",
		KeyUnknownCmd:          "未知命令。可用：/exit、/run <cmd>、/sh、/cancel、/config、/reload、/help",
		KeyConfigReloaded:      "配置与白名单已重载，下一条消息将使用新配置。",
		KeyCancelled:           "（已取消）",
		KeyErrorPrefix:         "错误：",
		KeyConfigPrefix:        "配置：",
		KeyConfigUnknownField:   "未知字段 ",
		KeyConfigLanguageRequired: "language: 需提供值（如 en、zh）",
		KeyConfigSaved:         "配置已保存（llm.%s）。",
		KeyConfigSavedLanguage: "配置已保存（language: %s）。",
		KeyWaitOrCancel:        "（请等待当前回复，或使用 /cancel）",
		KeyPlaceholderInput:    "输入命令或 / 查看斜杠命令…",
		KeyTitleHeader:         "delve-shell — Enter 发送，ctrl+c 退出 | Up/Down/PgUp/PgDown 滚动",
		KeyApprovalPrompt:      "待执行的命令（需你确认）：",
		KeyApproveYN:           "批准？(y/n)：",
		KeyRunTagApproved:      "已批准",
		KeyRunTagDirect:        "直接执行",
		KeyRunTagWhitelist:     "白名单",
		KeyResultSensitive:     "（结果含敏感数据，未写入历史。）",
		KeyErrLLMNotConfigured: "LLM 未配置。请用 /config 设置 llm.api_key（以及 llm.base_url、llm.model），设置后直接发消息即可，无需重启。支持 $VAR 或 ${VAR} 引用环境变量。配置文件路径：%s",
		KeyUserLabel:           "用户：",
		KeyAILabel:             "AI：",
		KeyRunLabel:            "执行：",
		KeyDescExit:            "退出 delve-shell",
		KeyDescRun:             "直接执行命令（不经 AI）",
		KeyDescSh:              "启动 bash；结束后返回",
		KeyDescCancel:          "取消当前 AI 请求",
		KeyDescConfig:          "设置或查看配置（如 /config llm base_url <url>、/config language zh）",
		KeyDescReload:          "重载配置与白名单（无需重启）",
		KeyDescHelp:            "显示此帮助",
		KeyDescConfigShow:      "显示当前配置路径与 LLM 摘要",
		KeyDescConfigLLMBaseURL: "设置 LLM API base URL",
		KeyDescConfigLLMApiKey:  "设置 LLM API key",
		KeyDescConfigLLMModel:   "设置 LLM 模型名",
		KeyDescConfigLanguage:   "设置界面语言（en、zh）",
	},
}

// T returns the message for lang and key. If lang or key is missing, falls back to "en" then key as-is.
func T(lang, key string) string {
	if lang == "" {
		lang = "en"
	}
	if m, ok := messages[lang]; ok {
		if s, ok := m[key]; ok {
			return s
		}
	}
	if lang != "en" {
		if m, ok := messages["en"]; ok {
			if s, ok := m[key]; ok {
				return s
			}
		}
	}
	return key
}

// Tf returns T(lang, key) formatted with fmt.Sprintf. Key must be a format key (e.g. KeyConfigSaved, KeyErrLLMNotConfigured).
func Tf(lang, key string, a ...interface{}) string {
	return fmt.Sprintf(T(lang, key), a...)
}
