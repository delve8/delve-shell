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
	KeyConfigUnknownField  = "config_unknown_field"
	KeyConfigSaved         = "config_saved"
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
		KeyDescConfigAllowlistUpdate = "desc_config_allowlist_update"
		KeyDescConfigAddRemote       = "desc_config_add_remote"
		KeyDescConfigRemoveRemote    = "desc_config_remove_remote"
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
		KeyConfigRemoteAdded     = "config_remote_added"     // format: name, target
		KeyConfigRemoteRemoved   = "config_remote_removed"  // format: name

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
	KeyRemoteNone      = "remote_none"       // No remotes configured
	KeyDescRemoteOn    = "desc_remote_on"
	KeyDescRemoteOff   = "desc_remote_off"
	KeyRemoteManualHint = "remote_manual_hint" // hint when no remotes or "or type user@host"
	KeyHelpTitle       = "help_title"
	KeyAddRemoteTitle  = "add_remote_title"
	KeyConfigSavedLLM  = "config_saved_llm"
	KeyConfigLLMTitle  = "config_llm_title"
	KeyConfigLLMBaseURLLabel  = "config_llm_base_url_label"
	KeyConfigLLMApiKeyLabel   = "config_llm_api_key_label"
	KeyConfigLLMModelLabel    = "config_llm_model_label"
		KeyConfigLLMHint         = "config_llm_hint"
	KeyConfigLLMApiKeyRequired = "config_llm_api_key_required"
	KeyDescConfigLLM  = "desc_config_llm"
	KeyConfigHint     = "config_hint" // when /config or /config show is used: point to /config llm and header
)

var messages = map[string]map[string]string{
	"en": {
		KeyHelpText: `delve-shell — AI-assisted ops. Every command runs only after you approve.

What it does:
  Describe a task in natural language; the AI suggests commands. Commands on the allowlist run automatically; others show a card (Run / Reject or Run / Copy / Dismiss). All runs are recorded in session history for audit.

Quick start:
  1. Type your task and press Enter.
  2. When a command card appears, press 1 to run, 2 to reject (or copy/dismiss when auto-run is off).
  3. Type / to list slash commands; use /help anytime for this panel.

Slash commands (each line: command, next line: description):

  /help
    Show this help
  /cancel
    Cancel current AI request
  /config
    Set or show config (see /config subcommands below)
  /config add-remote
    Add a remote (opens form)
  /config remove-remote
    Remove a remote target by name
  /config auto-run list-only
    Listed commands run without confirmation (saved to config)
  /config auto-run disable
    Every command shows Run/Copy/Dismiss (saved to config)
  /config update auto-run list
    Merge built-in default allowlist into current (add missing entries)
  /config llm
    Set LLM (base URL, API key, model)
  /remote on [user@host]
    Connect to a remote host (pick from config or type user@host)
  /remote off
    Disconnect from remote and run commands locally
  /new
    Start a new session
  /sessions
    List and switch to another session
  /reload
    Reload config and allowlist (no restart)
  /run <cmd>
    Run one command directly (no AI)
  /sh
    Spawn shell; exit shell to return here
  /exit, /q
    Quit (Ctrl+C also works)

Keyboard: Up/Down, PgUp/PgDown scroll. When input starts with /, Up/Down pick a suggestion, Enter fills then run.`,
		KeyNoRequestInProgress: "(No request in progress. /cancel only applies when waiting for AI.)",
		KeyUsageRun:            "Usage: /run <command> — e.g. /run ls -la",
		KeyUnknownCmd:          "Unknown command. Type /help for the full list, or try /exit, /run <cmd>, /config, /reload.",
		KeyConfigReloaded:      "Config and allowlist reloaded. Next message will use new config.",
		KeyCancelled:           "(Cancelled)",
		KeyErrorPrefix:         "Error: ",
		KeyConfigPrefix:        "Config: ",
		KeyConfigUnknownField:   "unknown field ",
		KeyConfigSaved:         "Config saved (llm.%s).",
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
		KeyDescConfig:          "Set or show config (e.g. /config llm base_url <url>)",
		KeyDescReload:          "Reload config and allowlist (no restart)",
		KeyDescHelp:            "Show this help",
		KeyDescConfigShow:      "Show current config path and LLM summary",
		KeyDescConfigLLMBaseURL: "Set LLM API base URL",
		KeyDescConfigLLMApiKey:  "Set LLM API key",
		KeyDescConfigLLMModel:   "Set LLM model name",
		KeyDescConfigAllowlistUpdate: "Merge built-in default allowlist into current (add missing entries)",
		KeyDescConfigAddRemote:       "Add a remote (opens form)",
		KeyDescConfigRemoveRemote:    "Remove a remote target by name",
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
		KeyConfigRemoteAdded:     "Remote added: %s.",
		KeyConfigRemoteRemoved:   "Remote removed: %s.",
		KeyStatusIdle:           "[IDLE]",
		KeyStatusRunning:         "[PROCESSING]",
		KeyStatusPendingApproval: "[NEED APPROVAL]",
		KeyStatusSuggest:         "[SUGGEST]",
		KeyNeedConfirmationHint:  "Your confirmation required.",
		KeyWizardTitle:           "=== delve-shell first-time setup ===",
		KeyWizardConfigPath:      "Config path: %s",
		KeyWizardIntroDesc1:     "This wizard will set LLM config (base_url, api_key, model).",
		KeyWizardIntroDesc2:     "",
		KeyWizardIntroEnv:       "Values support $VAR or ${VAR} environment references.",
		KeyWizardBaseURLPrompt:  "LLM base URL (optional; empty uses provider default, e.g. OpenAI official). For OpenAI-compatible APIs, use the /v1 endpoint (e.g. https://api.openai.com/v1): ",
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
		KeyRemoteNone:           "No remotes configured.",
		KeyDescRemoteOn:         "Connect to a remote host (pick from config or type user@host)",
		KeyDescRemoteOff:        "Disconnect from remote and run commands locally",
		KeyRemoteManualHint:     "Or type user@host (e.g. root@1.2.3.4)",
		KeyHelpTitle:            "Help",
		KeyAddRemoteTitle:       "Add Remote",
		KeyConfigSavedLLM:       "Config saved (llm).",
		KeyConfigLLMTitle:       "Config LLM",
		KeyConfigLLMBaseURLLabel: "Base URL (optional; empty = default; for OpenAI-compatible APIs use the /v1 endpoint):",
		KeyConfigLLMApiKeyLabel:  "API key (required; supports $VAR):",
		KeyConfigLLMModelLabel:   "Model (optional; empty = gpt-4o-mini):",
		KeyConfigLLMHint:         "Up/Down to move between fields, Enter to save, Esc to cancel.",
		KeyConfigLLMApiKeyRequired: "API key is required.",
		KeyDescConfigLLM:        "Set LLM (base URL, API key, model)",
		KeyConfigHint:           "Use /config llm for LLM; auto-run is in header.",
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
