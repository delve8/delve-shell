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
	KeyInputHintApprove    = "input_hint_approve"     // placeholder when waiting for 1/2 (approval)
	KeyInputHintApproveThree = "input_hint_approve_three" // placeholder when waiting for 1/2/3 (Run/Copy/Dismiss)
	KeyInputHintSensitive  = "input_hint_sensitive"  // placeholder when waiting for 1/2/3 (sensitive)
	// Choice menu labels (for Up/Down + Enter selection list)
	KeyChoiceApprove   = "choice_approve"
	KeyChoiceReject    = "choice_reject"
	KeyChoiceRefuse    = "choice_refuse"
	KeyChoiceRunStore  = "choice_run_store"
	KeyChoiceRunNoStore = "choice_run_no_store"
	KeyChoiceCopy      = "choice_copy"
	KeyChoiceDismiss   = "choice_dismiss"
	KeyTitleHeader         = "title_header"
	KeyApprovalPrompt      = "approval_prompt"
	KeyApprovalWhy         = "approval_why"
	KeyApproveYN           = "approve_yn"       // 2 options
	KeyApproveYNThree      = "approve_yn_three" // 3 options: Run, Copy, Dismiss
	KeyApprovalDecisionApproved = "approval_decision_approved"
	KeyApprovalDecisionRejected = "approval_decision_rejected"
	KeyRiskReadOnly        = "risk_read_only"
	KeyRiskLow             = "risk_low"
	KeyRiskHigh            = "risk_high"
	KeyRunTagApproved      = "run_tag_approved"
	KeyRunTagDirect        = "run_tag_direct"
	KeyRunTagAllowlist     = "run_tag_allowlist"
	KeyResultSensitive     = "result_sensitive"
	KeySensitivePrompt     = "sensitive_prompt"
	KeySensitiveChoice1    = "sensitive_choice_1"
	KeySensitiveChoice2    = "sensitive_choice_2"
	KeySensitiveChoice3    = "sensitive_choice_3"
	KeySensitivePressKey   = "sensitive_press_key"
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
	KeyDescConfigLanguage     = "desc_config_language"
	KeyDescConfigAllowlistUpdate = "desc_config_allowlist_update"
	KeyAllowlistUpdateDone    = "allowlist_update_done" // format: added count
	KeyDescAutoRunListOnly = "desc_auto_run_list_only"
	KeyDescAutoRunDisable  = "desc_auto_run_disable"
	KeyModeRequired           = "mode_required"
	KeyRunTagSuggested        = "run_tag_suggested"
	KeySuggestedCopyHint      = "suggested_copy_hint"
	KeySuggestedCardTitle     = "suggested_card_title"
	KeySuggestedCardHint      = "suggested_card_hint"
	KeySuggestedCopied        = "suggested_copied"
	KeyAutoRunLabel           = "auto_run_label"   // "Auto-Run: " (EN) / "自动执行：" (ZH)
	KeyAutoRunListOnly        = "auto_run_list_only" // "List Only" / "名单内的"
	KeyAutoRunNone            = "auto_run_none"     // "Disabled" (EN) / "已关闭" (ZH)
	KeyModeSetTo              = "mode_set_to"       // deprecated; was mode
	KeyAllowlistAutoRunSetTo  = "allowlist_auto_run_set_to" // "Auto-run set to %s (this session only)."
	KeyConfigSavedAllowlistAutoRun = "config_saved_allowlist_auto_run" // "Config saved (allowlist_auto_run: %s)."
	KeyConfigAutoRunRequired = "config_auto_run_required"

	// Status bar (title): IDLE / RUNNING / pending approval / suggest card
	KeyStatusIdle             = "status_idle"
	KeyStatusRunning          = "status_running"
	KeyStatusPendingApproval  = "status_pending_approval"
	KeyStatusSuggest          = "status_suggest"
	KeyNeedConfirmationHint   = "need_confirmation_hint"

	// First-time wizard (before lang is chosen use "en" for intro; after language step use chosen lang)
	KeyWizardTitle        = "wizard_title"
	KeyWizardConfigPath   = "wizard_config_path"   // format: %s
	KeyWizardIntroDesc1   = "wizard_intro_desc_1"
	KeyWizardIntroDesc2   = "wizard_intro_desc_2"
	KeyWizardIntroEnv     = "wizard_intro_env"
	KeyWizardLangPrompt   = "wizard_lang_prompt"
	KeyWizardLangInvalid  = "wizard_lang_invalid"
	KeyWizardBaseURLPrompt   = "wizard_base_url_prompt"
	KeyWizardAPIKeyPrompt   = "wizard_api_key_prompt"
	KeyWizardAPIKeyRequired = "wizard_api_key_required"
	KeyWizardModelPrompt  = "wizard_model_prompt"
	KeyWizardDone         = "wizard_done"

	// Session picker (startup)
	KeySessionTitle    = "session_title"
	KeySessionNew      = "session_new"
	KeySessionPrompt   = "session_prompt"
	KeySessionSwitched   = "session_switched"
	KeySessionSwitchedTo = "session_switched_to" // format: "Switched to session: %s" (session id)
	KeySessionSelect     = "session_select"
	KeyDescSessions    = "desc_sessions"
	KeySessionNone     = "session_none"
)

var messages = map[string]map[string]string{
	"en": {
		KeyHelpText: `delve-shell — AI-assisted ops, commands run after your approval.

Slash commands:
  /exit, /q      Quit
  /run <cmd>     Run a command directly (no AI)
  /sh            Spawn bash; return here when done
  /cancel        Cancel current AI request
  /config        Set or show config: /config show, /config auto-run <list-only|disable>, /config llm ..., /config language <en|zh>
  /config auto-run list-only   Listed commands run without confirmation (saved to config)
  /config auto-run disable    Every command shows Run/Copy/Dismiss (saved to config)
  /reload        Reload config and allowlist (no restart)
  /help          Show this help

Scroll: Up/Down, PgUp/PgDown. Text selection: use terminal mouse (no mouse reporting).`,
		KeyNoRequestInProgress: "(No request in progress)",
		KeyUsageRun:            "Usage: /run <command>",
		KeyUnknownCmd:          "Unknown command. Use /exit or /q, /run <cmd>, /sh, /cancel, /config, /reload, /help",
		KeyConfigReloaded:      "Config and allowlist reloaded. Next message will use new config.",
		KeyCancelled:           "(Cancelled)",
		KeyErrorPrefix:         "Error: ",
		KeyConfigPrefix:        "Config: ",
		KeyConfigUnknownField:   "unknown field ",
		KeyConfigLanguageRequired: "language: value required (e.g. en, zh)",
		KeyConfigSaved:         "Config saved (llm.%s).",
		KeyConfigSavedLanguage:  "Config saved (language: %s).",
		KeyWaitOrCancel:        "(Please wait for the current response, or /cancel)",
		KeyPlaceholderInput:    "Type your question or task, or / for slash commands...",
		KeyInputHintApprove:     "1 or 2",
		KeyInputHintApproveThree: "1, 2 or 3",
		KeyInputHintSensitive:  "1, 2 or 3",
		KeyChoiceApprove:       "Approve",
		KeyChoiceReject:        "Reject",
		KeyChoiceRefuse:        "Refuse (do not run)",
		KeyChoiceRunStore:      "Run, return to AI, store in history",
		KeyChoiceRunNoStore:    "Run, return to AI, do not store",
		KeyChoiceCopy:          "Copy",
		KeyChoiceDismiss:       "Dismiss",
		KeyTitleHeader:         "delve-shell — Enter to send, ctrl+c to quit | Up/Down/PgUp/PgDown scroll",
		KeyApprovalPrompt:           "Command to run (approval required):",
		KeyApprovalWhy:              "Why:",
		KeyApproveYN:                "1=approve, 2=reject",
		KeyApproveYNThree:           "1=Run, 2=Copy, 3=Dismiss",
		KeyApprovalDecisionApproved: "Decision: approved",
		KeyApprovalDecisionRejected: "Decision: rejected",
		KeyRiskReadOnly:       "READ-ONLY",
		KeyRiskLow:            "LOW-RISK",
		KeyRiskHigh:           "HIGH-RISK",
		KeyRunTagApproved:      "approved",
		KeyRunTagDirect:        "direct",
		KeyRunTagAllowlist:     "allowlist",
		KeyResultSensitive:     "(Result contains sensitive data; not stored in history.)",
		KeySensitivePrompt:     "This command may access sensitive file(s). Choose:",
		KeySensitiveChoice1:    "1 = Refuse (do not run)",
		KeySensitiveChoice2:    "2 = Run, return result to AI, store in history",
		KeySensitiveChoice3:    "3 = Run, return result to AI, do not store in history",
		KeySensitivePressKey:   "Press 1, 2, or 3: ",
		KeyErrLLMNotConfigured: "LLM not configured. Use /config to set llm.api_key (and llm.base_url, llm.model), then send a message again (no restart needed). Supports $VAR or ${VAR} for env. Config path: %s",
		KeyUserLabel:           "User: ",
		KeyAILabel:             "AI: ",
		KeyRunLabel:            "Run: ",
		KeyDescExit:            "Quit delve-shell",
		KeyDescRun:             "Run a command directly (no AI)",
		KeyDescSh:              "Spawn bash; return here when done",
		KeyDescCancel:          "Cancel current AI request",
		KeyDescConfig:          "Set or show config (e.g. /config llm base_url <url>, /config language en)",
		KeyDescReload:          "Reload config and allowlist (no restart)",
		KeyDescHelp:            "Show this help",
		KeyDescConfigShow:      "Show current config path and LLM summary",
		KeyDescConfigLLMBaseURL: "Set LLM API base URL",
		KeyDescConfigLLMApiKey:  "Set LLM API key",
		KeyDescConfigLLMModel:   "Set LLM model name",
		KeyDescConfigLanguage:     "Set UI language (en, zh)",
		KeyDescConfigAllowlistUpdate: "Merge built-in default allowlist into current (add missing entries)",
		KeyAllowlistUpdateDone:    "Allowlist updated: %d new pattern(s) added. Use /reload to apply.",
		KeyDescAutoRunListOnly: "Listed commands run without confirmation (saved to config)",
		KeyDescAutoRunDisable:  "Every command shows Run/Copy/Dismiss (saved to config)",
		KeyModeRequired:         "Usage: /mode suggest or /mode run",
		KeyRunTagSuggested:       "suggested",
		KeySuggestedCopyHint:     "Select the command above to copy, or use /run <cmd> to run it.",
		KeySuggestedCardTitle:    "Suggested command (not executed):",
		KeySuggestedCardHint:     "1=copy, 2=dismiss",
		KeySuggestedCopied:       "Copied to clipboard.",
		KeyAutoRunLabel:          "Auto-Run: ",
		KeyAutoRunListOnly:       "List Only",
		KeyAutoRunNone:           "Disabled",
		KeyAllowlistAutoRunSetTo: "Auto-Run set to %s (this session only).",
		KeyConfigSavedAllowlistAutoRun: "Config saved (auto-run: %s). Use /reload to apply as default.",
		KeyConfigAutoRunRequired: "auto-run: use list-only or disable",
		KeyStatusIdle:           "[IDLE]",
		KeyStatusRunning:         "[PROCESSING]",
		KeyStatusPendingApproval: "[NEED APPROVAL]",
		KeyStatusSuggest:         "[SUGGEST]",
		KeyNeedConfirmationHint:  "Your confirmation required.",
		KeyWizardTitle:           "=== delve-shell first-time setup ===",
		KeyWizardConfigPath:      "Config path: %s",
		KeyWizardIntroDesc1:     "This wizard will set UI language and LLM config (base_url, api_key, model).",
		KeyWizardIntroDesc2:     "",
		KeyWizardIntroEnv:       "Values support $VAR or ${VAR} environment references.",
		KeyWizardLangPrompt:     "UI language [en/zh] (default: en): ",
		KeyWizardLangInvalid:    "Please enter 'en' or 'zh'.",
		KeyWizardBaseURLPrompt:  "LLM base URL (optional; empty uses provider default, e.g. OpenAI official): ",
		KeyWizardAPIKeyPrompt:   "LLM api_key (required; supports $VAR or ${VAR}): ",
		KeyWizardAPIKeyRequired: "api_key is required. Use an env reference if you prefer not to store the key directly.",
		KeyWizardModelPrompt:    "LLM model (optional; default: gpt-4o-mini): ",
		KeyWizardDone:           "Config captured; it will be saved and delve-shell will start.",
		KeySessionTitle:         "=== Session ===",
		KeySessionNew:           "New session",
		KeySessionPrompt:        "Choose (0 = new, 1–%d = continue): ",
		KeySessionSwitched:      "Switched to session.",
		KeySessionSwitchedTo:    "Switched to session: %s",
		KeySessionSelect:        "Up/Down select, Enter switch",
		KeyDescSessions:         "Switch session",
		KeySessionNone:          "No previous sessions.",
	},
	"zh": {
		KeyHelpText: `delve-shell — AI 辅助运维，命令经你确认后执行。

斜杠命令：
  /exit, /q      退出
  /run <cmd>     直接执行命令（不经 AI）
  /sh            启动 bash；结束后返回
  /cancel        取消当前 AI 请求
  /config        设置或查看配置：/config show、/config auto-run <list-only|disable>、/config llm ...、/config language <en|zh>
  /config auto-run list-only  名单内命令免确认执行（写入配置）
  /config auto-run disable   每条命令均显示 执行/复制/关闭（写入配置）
  /reload        重载配置与允许列表（无需重启）
  /help          显示此帮助

滚动：Up/Down、PgUp/PgDown。文本选择：使用终端鼠标（无需 mouse reporting）。`,
		KeyNoRequestInProgress: "（当前无进行中的请求）",
		KeyUsageRun:            "用法：/run <命令>",
		KeyUnknownCmd:          "未知命令。可用：/exit 或 /q、/run <cmd>、/sh、/cancel、/config、/reload、/help",
		KeyConfigReloaded:      "配置与允许列表已重载，下一条消息将使用新配置。",
		KeyCancelled:           "（已取消）",
		KeyErrorPrefix:         "错误：",
		KeyConfigPrefix:        "配置：",
		KeyConfigUnknownField:   "未知字段 ",
		KeyConfigLanguageRequired: "language: 需提供值（如 en、zh）",
		KeyConfigSaved:         "配置已保存（llm.%s）。",
		KeyConfigSavedLanguage: "配置已保存（language: %s）。",
		KeyWaitOrCancel:        "（请等待当前回复，或使用 /cancel）",
		KeyPlaceholderInput:    "输入问题或任务，或 / 查看斜杠命令…",
		KeyInputHintApprove:     "1 或 2",
		KeyInputHintApproveThree: "1、2 或 3",
		KeyInputHintSensitive:  "1、2 或 3",
		KeyChoiceApprove:       "批准",
		KeyChoiceReject:        "拒绝",
		KeyChoiceRefuse:        "拒绝（不执行）",
		KeyChoiceRunStore:      "执行并写入历史",
		KeyChoiceRunNoStore:    "执行但不写入历史",
		KeyChoiceCopy:          "复制",
		KeyChoiceDismiss:       "关闭",
		KeyTitleHeader:         "delve-shell — Enter 发送，ctrl+c 退出 | Up/Down/PgUp/PgDown 滚动",
		KeyApprovalPrompt:           "待执行的命令（需你确认）：",
		KeyApprovalWhy:              "原因：",
		KeyApproveYN:                "1=批准，2=拒绝",
		KeyApproveYNThree:           "1=执行，2=复制，3=关闭",
		KeyApprovalDecisionApproved: "决定：已批准",
		KeyApprovalDecisionRejected: "决定：已拒绝",
		KeyRiskReadOnly:       "只读",
		KeyRiskLow:            "低风险",
		KeyRiskHigh:           "高风险",
		KeyRunTagApproved:      "已批准",
		KeyRunTagDirect:        "直接执行",
		KeyRunTagAllowlist:     "允许列表",
		KeyResultSensitive:     "（结果含敏感数据，未写入历史。）",
		KeySensitivePrompt:     "该命令可能访问敏感文件。请选择：",
		KeySensitiveChoice1:    "1 = 拒绝（不执行）",
		KeySensitiveChoice2:    "2 = 执行，结果返回 AI 并写入历史",
		KeySensitiveChoice3:    "3 = 执行，结果返回 AI，但不写入历史",
		KeySensitivePressKey:   "请按 1、2 或 3：",
		KeyErrLLMNotConfigured: "LLM 未配置。请用 /config 设置 llm.api_key（以及 llm.base_url、llm.model），设置后直接发消息即可，无需重启。支持 $VAR 或 ${VAR} 引用环境变量。配置文件路径：%s",
		KeyUserLabel:           "用户：",
		KeyAILabel:             "AI：",
		KeyRunLabel:            "执行：",
		KeyDescExit:            "退出 delve-shell",
		KeyDescRun:             "直接执行命令（不经 AI）",
		KeyDescSh:              "启动 bash；结束后返回",
		KeyDescCancel:          "取消当前 AI 请求",
		KeyDescConfig:          "设置或查看配置（如 /config llm base_url <url>、/config language zh）",
		KeyDescReload:          "重载配置与允许列表（无需重启）",
		KeyDescHelp:            "显示此帮助",
		KeyDescConfigShow:      "显示当前配置路径与 LLM 摘要",
		KeyDescConfigLLMBaseURL: "设置 LLM API base URL",
		KeyDescConfigLLMApiKey:  "设置 LLM API key",
		KeyDescConfigLLMModel:   "设置 LLM 模型名",
		KeyDescConfigLanguage:     "设置界面语言（en、zh）",
		KeyDescConfigAllowlistUpdate: "将内置默认允许列表合并到当前（仅追加缺失项）",
		KeyAllowlistUpdateDone:    "允许列表已更新：新增 %d 条。使用 /reload 生效。",
		KeyDescAutoRunListOnly: "名单内命令免确认执行（写入配置）",
		KeyDescAutoRunDisable:  "每条命令均显示 执行/复制/关闭（写入配置）",
		KeyModeRequired:          "用法：/mode suggest 或 /mode run",
		KeyRunTagSuggested:       "建议",
		KeySuggestedCopyHint:     "可选中上方命令复制，或使用 /run <cmd> 执行。",
		KeySuggestedCardTitle:    "建议的命令（未执行）：",
		KeySuggestedCardHint:     "1=复制，2=关闭",
		KeySuggestedCopied:       "已复制到剪贴板。",
		KeyAutoRunLabel:          "自动执行：",
		KeyAutoRunListOnly:       "名单内的",
		KeyAutoRunNone:           "已关闭",
		KeyAllowlistAutoRunSetTo: "当前会话已设为 %s（不写入配置）。",
		KeyConfigSavedAllowlistAutoRun: "配置已保存（auto-run: %s）。使用 /reload 后作为默认生效。",
		KeyConfigAutoRunRequired: "auto-run：请使用 list-only 或 disable",
		KeyStatusIdle:           "[空闲]",
		KeyStatusRunning:         "[处理中]",
		KeyStatusPendingApproval: "[待确认]",
		KeyStatusSuggest:         "[建议]",
		KeyNeedConfirmationHint:  "需要你的确认。",
		KeyWizardTitle:           "=== delve-shell 首次启动向导 ===",
		KeyWizardConfigPath:      "配置文件路径：%s",
		KeyWizardIntroDesc1:     "本向导将设置界面语言和 LLM 配置（base_url、api_key、model）。",
		KeyWizardIntroDesc2:     "",
		KeyWizardIntroEnv:       "以上字段均支持使用环境变量引用（如 $VAR 或 ${VAR}）。",
		KeyWizardLangPrompt:     "界面语言 [en/zh]（默认 en）：",
		KeyWizardLangInvalid:    "请输入 en 或 zh。",
		KeyWizardBaseURLPrompt:  "LLM base URL（可选；留空则使用默认，例如 OpenAI 官方接口）：",
		KeyWizardAPIKeyPrompt:   "LLM api_key（必填；支持 $VAR 或 ${VAR}）：",
		KeyWizardAPIKeyRequired: "api_key 为必填项。若不希望直接写入密钥，可只填环境变量引用。",
		KeyWizardModelPrompt:    "LLM model（可选；默认 gpt-4o-mini）：",
		KeyWizardDone:           "配置已写入内存，稍后将保存到文件并启动 delve-shell。",
		KeySessionTitle:         "=== 会话 ===",
		KeySessionNew:           "新会话",
		KeySessionPrompt:        "选择（0=新会话，1–%d=继续）：",
		KeySessionSwitched:      "已切换会话。",
		KeySessionSwitchedTo:    "已切换到会话：%s",
		KeySessionSelect:        "上下选择，Enter 切换",
		KeyDescSessions:         "切换会话",
		KeySessionNone:          "没有往期会话。",
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
