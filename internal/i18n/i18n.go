package i18n

import "fmt"

// Msg keys for user-facing strings. Code error messages stay in English in callers.
const (
	KeyHelpText              = "help_text"
	KeyUsageRun              = "usage_run"
	KeyUnknownCmd            = "unknown_cmd"
	KeyCancelled             = "cancelled"
	KeyErrorPrefix           = "error_prefix"
	KeyConfigPrefix          = "config_prefix"
	KeyConfigUnknownField    = "config_unknown_field"
	KeyConfigSaved           = "config_saved"
	KeyWaitOrCancel          = "wait_or_cancel"
	KeyPlaceholderInput      = "placeholder_input"
	KeyInputHintApprove      = "input_hint_approve"       // placeholder when waiting for 1/2 (approval)
	KeyInputHintApproveThree = "input_hint_approve_three" // placeholder when waiting for 1/2/3 (Run/Dismiss/Copy)
	KeyInputHintSensitive    = "input_hint_sensitive"     // placeholder when waiting for 1/2/3 (sensitive)
	KeyInputHistBrowsingHint = "input_hist_browsing_hint" // one line under input while walking local input history
	// Choice menu labels (for Up/Down + Enter selection list)
	KeyChoiceApprove            = "choice_approve"
	KeyChoiceReject             = "choice_reject"
	KeyChoiceRefuse             = "choice_refuse"
	KeyChoiceRunStore           = "choice_run_store"
	KeyChoiceRunNoStore         = "choice_run_no_store"
	KeyChoiceCopy               = "choice_copy"
	KeyChoiceDismiss            = "choice_dismiss"
	KeyApprovalPrompt           = "approval_prompt"
	KeyApprovalSummary          = "approval_summary"
	KeyApprovalWhy              = "approval_why"
	KeyApproveYN                = "approve_yn"       // 2 options
	KeyApproveYNThree           = "approve_yn_three" // 3 options: Run, Dismiss, Copy
	KeyApprovalDecisionApproved = "approval_decision_approved"
	KeyApprovalDecisionRejected = "approval_decision_rejected"
	KeyRiskReadOnly             = "risk_read_only"
	KeyRiskLow                  = "risk_low"
	KeyRiskHigh                 = "risk_high"
	KeyRunTagApproved           = "run_tag_approved"
	KeyRunTagDirect             = "run_tag_direct"
	KeyRunTagAllowlist          = "run_tag_allowlist"
	KeyResultSensitive          = "result_sensitive"
	KeySensitivePrompt          = "sensitive_prompt"
	KeySensitiveChoice1         = "sensitive_choice_1"
	KeySensitiveChoice2         = "sensitive_choice_2"
	KeySensitiveChoice3         = "sensitive_choice_3"
	KeySensitivePressKey        = "sensitive_press_key"
	KeyErrLLMNotConfigured      = "err_llm_not_configured"
	KeyUserLabel                = "user_label"
	KeyAILabel                  = "ai_label"
	KeyDelveLabel               = "delve_label" // tool/system message prefix, e.g. "Delve:"
	KeyRunLabel                 = "run_label"
	KeySkillLine                = "skill_line" // format: Skill: %s
	// Slash option descriptions (cmd as suffix for consistency)
	KeyDescExit   = "desc_exit"
	KeyDescRun    = "desc_run"
	KeyDescSh     = "desc_sh"
	KeyDescConfig = "desc_config"
	KeyDescHelp   = "desc_help"
	// /config 子项说明（仅在选择 /config 后显示）
	KeyDescConfigLLMBaseURL        = "desc_config_llm_base_url"
	KeyDescConfigLLMApiKey         = "desc_config_llm_api_key"
	KeyDescConfigLLMModel          = "desc_config_llm_model"
	KeyDescConfigAllowlistUpdate   = "desc_config_allowlist_update"
	KeyDescConfigRemoveRemote      = "desc_config_remove_remote"
	KeyAllowlistUpdateDone         = "allowlist_update_done" // format: added count
	KeyModeRequired                = "mode_required"
	KeyRunTagSuggested             = "run_tag_suggested"
	KeySuggestedCopyHint           = "suggested_copy_hint"
	KeySuggestedCardTitle          = "suggested_card_title"
	KeySuggestedCardHint           = "suggested_card_hint"
	KeySuggestedCopied             = "suggested_copied"
	KeyModeSetTo                   = "mode_set_to"        // deprecated; was mode
	KeyConfigRemoteAdded           = "config_remote_added"   // format: name, target
	KeyConfigRemoteRemoved         = "config_remote_removed" // format: name

	// Status bar (title): IDLE / RUNNING / pending approval / suggest card
	KeyStatusIdle            = "status_idle"
	KeyStatusRunning         = "status_running"
	KeyStatusPendingApproval = "status_pending_approval"
	KeyStatusSuggest         = "status_suggest"
	KeyNeedConfirmationHint  = "need_confirmation_hint"

	// First-time wizard (before lang is chosen use "en" for intro; after language step use chosen lang)
	KeyWizardTitle          = "wizard_title"
	KeyWizardConfigPath     = "wizard_config_path" // format: %s
	KeyWizardIntroDesc1     = "wizard_intro_desc_1"
	KeyWizardIntroDesc2     = "wizard_intro_desc_2"
	KeyWizardIntroEnv       = "wizard_intro_env"
	KeyWizardLangPrompt     = "wizard_lang_prompt"
	KeyWizardLangInvalid    = "wizard_lang_invalid"
	KeyWizardBaseURLPrompt  = "wizard_base_url_prompt"
	KeyWizardAPIKeyPrompt   = "wizard_api_key_prompt"
	KeyWizardAPIKeyRequired = "wizard_api_key_required"
	KeyWizardModelPrompt    = "wizard_model_prompt"
	KeyWizardDone           = "wizard_done"

	// Session picker (startup)
	KeySessionTitle                  = "session_title"
	KeySessionNew                    = "session_new"
	KeySessionPrompt                 = "session_prompt"
	KeySessionSwitched               = "session_switched"
	KeySessionSwitchedTo             = "session_switched_to" // format: "Switched to session: %s" (session id)
	KeySessionSelect                 = "session_select"
	KeyDescSessions                  = "desc_sessions"
	KeySessionNone                   = "session_none"
	KeyRemoteNone                    = "remote_none"         // No remotes configured
	KeyDelRemoteNoHosts              = "del_remote_no_hosts" // slash dropdown when no remotes to remove (Cmd-only row, like KeySkillNone)
	KeyDescRemoteOn                  = "desc_remote_on"
	KeyDescRemoteOff                 = "desc_remote_off"
	KeyRemoteManualHint              = "remote_manual_hint" // hint when no remotes or "or type user@host"
	KeyHelpTitle                     = "help_title"
	KeyAddRemoteTitle                = "add_remote_title"
	KeyConfigSavedLLM                = "config_saved_llm"
	KeyConfigLLMTitle                = "config_llm_title"
	KeyConfigLLMBaseURLLabel         = "config_llm_base_url_label"
	KeyConfigLLMApiKeyLabel          = "config_llm_api_key_label"
	KeyConfigLLMModelLabel           = "config_llm_model_label"
	KeyConfigLLMMaxMessagesLabel     = "config_llm_max_messages_label"
	KeyConfigLLMMaxCharsLabel        = "config_llm_max_chars_label"
	KeyConfigLLMApiKeyRequired       = "config_llm_api_key_required"
	KeyConfigLLMModelRequired        = "config_llm_model_required"
	KeyConfigLLMChecking             = "config_llm_checking"                // "Checking..."
	KeyConfigLLMCheckOK              = "config_llm_check_ok"                // after save: "LLM check OK."
	KeyConfigLLMCheckFailed          = "config_llm_check_failed"            // format: "LLM check failed: %v"
	KeyConfigLLMBaseURLAutoCorrected = "config_llm_base_url_auto_corrected" // format: "Base URL updated to %s (added /v1)."
	KeyDescConfigLLM                 = "desc_config_llm"
	// Skill
	KeyDescSkill             = "desc_skill"
	KeyUsageSkill            = "usage_skill"
	KeySkillNotFound         = "skill_not_found"
	KeySkillScriptNotFound   = "skill_script_not_found"
	KeySkillNone             = "skill_none"
	KeyDescSkillInstall      = "desc_skill_install"
	KeyDescSkillRemove       = "desc_skill_remove"
	KeyDescConfigUpdateSkill = "desc_config_update_skill"
	KeyAddSkillTitle         = "add_skill_title"
	KeyAddSkillURLLabel      = "add_skill_url_label"
	KeyAddSkillRefLabel      = "add_skill_ref_label"
	KeyAddSkillPathLabel     = "add_skill_path_label"
	KeyAddSkillNameLabel     = "add_skill_name_label"
	KeyAddSkillAuthHint      = "add_skill_auth_hint"
	KeyAddSkillURLRequired   = "add_skill_url_required"
	KeySkillInstalled        = "skill_installed"
	KeySkillRemoved          = "skill_removed"
	KeySkillInstallFailed    = "skill_install_failed"
	KeySkillRemoveFailed     = "skill_remove_failed"
	KeyUsageSkillInstall     = "usage_skill_install"
	KeyUsageSkillRemove      = "usage_skill_remove"
	KeySkillAlreadyExists    = "skill_already_exists"

	// Overlay keyboard hints (dim footer / pick lists; full line per key for translation word order).
	KeyOverlayFormFooter          = "overlay_form_footer"
	KeyOverlayPicklistHint        = "overlay_picklist_hint"
	KeyOverlayUpdateSkillRefTitle = "overlay_update_skill_ref_title"
	KeyOverlayEscCancel           = "overlay_esc_cancel"
	KeyOverlayEnterContinueEsc    = "overlay_enter_continue_esc"
	KeyOverlay12SelectEsc         = "overlay_12_select_esc"
	KeyOverlayEnterSubmitEsc      = "overlay_enter_submit_esc"
)

var messages = map[string]map[string]string{
	"en": {
		KeyHelpText: `delve-shell — AI-assisted ops. Every command runs only after you approve.

What it does:
  Describe a task in natural language; the AI suggests commands. Commands that match the allowlist (and have no shell write redirection) run without a card; all others show a card (Run, Dismiss, or Copy). An empty allowlist matches nothing, so every command shows the card. All runs are recorded in session history for audit.

Quick start:
  1. Type your task and press Enter.
  2. Multi-line messages: Shift+Enter, Alt+Enter, or Ctrl+J inserts a newline; Enter sends. Many terminals treat Shift+Enter the same as Enter—Alt+Enter or Ctrl+J remains reliable.
  3. When a command card appears, press 1 to run, 2 to dismiss without running, 3 to copy the command.
  4. Up/Down recall recent submitted lines (chat and slash). While a recalled line starts with /, keep using Up/Down to walk history; slash completion applies after you edit (any non–↑/↓ key) or finish browsing.
  5. Type / for slash suggestions (Up/Down while editing a / line, then Enter).
  6. Scroll the log with PgUp/PgDown when needed; /help opens this panel.

Slash commands (command line, then description; blank line between entries):

/help
Show this help

/config
Config subcommands (see list below)

/config del-remote
Remove a remote

/config add-skill <url> [ref] [path]
Install a skill from a git repo (path = subpath if the repo has multiple skills, e.g. skills/foo)

/config del-skill <skill_name>
Remove an installed skill

/config update-skill <skill_name>
Update an installed skill from its git source (branch/tag selectable in dialog)

/config update auto-run list
Merge default allowlist

/config model
Configure model (LLM API)

/access
Connect over SSH: dropdown lists saved hosts first, then /access New (add target), then /access Local (use local executor). Host segment in saved targets must be lowercase so /access Local and /access New do not collide with host names.

/access New
Open Add Remote (new SSH target; optional save to config)

/access Local
Disconnect from remote and run commands locally

/access [user@host or host]
Connect to a saved host or enter user@host

/new
Start a new session

/session
List and switch to another session

/skill <name> [detail]
Use skill; optional detail for the AI

/exec <cmd>
Run one command directly (no AI)

` + helpEnBashSection + `/quit
Quit (Ctrl+C also works)`,
		KeyUsageRun:                      "Usage: /exec <command> — e.g. /exec ls -la",
		KeyUnknownCmd:                    "Unknown command. Type /help for the full list, or try /quit, /exec <cmd>, /config.",
		KeyDelveLabel:                    "Delve:",
		KeyCancelled:                     "(Cancelled)",
		KeyErrorPrefix:                   "Error: ",
		KeyConfigPrefix:                  "Config: ",
		KeyConfigUnknownField:            "unknown field ",
		KeyConfigSaved:                   "Config saved (llm.%s).",
		KeyWaitOrCancel:                  "(Please wait for the current response, or press Esc to cancel)",
		KeyPlaceholderInput:              "Type your question or / for slash commands.",
		KeyInputHistBrowsingHint:         "↑/↓ input history · Enter to send · any other key edits",
		KeyInputHintApprove:              "1 or 2",
		KeyInputHintApproveThree:         "1, 2 or 3",
		KeyInputHintSensitive:            "1, 2 or 3",
		KeyChoiceApprove:                 "Approve",
		KeyChoiceReject:                  "Reject",
		KeyChoiceRefuse:                  "Refuse (do not run)",
		KeyChoiceRunStore:                "Run, return to AI, store in history",
		KeyChoiceRunNoStore:              "Run, return to AI, do not store",
		KeyChoiceCopy:                    "Copy",
		KeyChoiceDismiss:                 "Dismiss",
		KeyApprovalPrompt:                "Command to run (approval required):",
		KeyApprovalSummary:               "Summary:",
		KeyApprovalWhy:                   "Why:",
		KeyApproveYN:                     "1=approve, 2=reject",
		KeyApproveYNThree:                "1=Run, 2=Dismiss, 3=Copy",
		KeyApprovalDecisionApproved:      "Decision: approved",
		KeyApprovalDecisionRejected:      "Decision: rejected",
		KeyRiskReadOnly:                  "READ-ONLY",
		KeyRiskLow:                       "LOW-RISK",
		KeyRiskHigh:                      "HIGH-RISK",
		KeyRunTagApproved:                "approved",
		KeyRunTagDirect:                  "direct",
		KeyRunTagAllowlist:               "allowlist",
		KeyResultSensitive:               "(Result contains sensitive data; not stored in history.)",
		KeySensitivePrompt:               "This command may access sensitive file(s). Choose:",
		KeySensitiveChoice1:              "1 = Refuse (do not run)",
		KeySensitiveChoice2:              "2 = Run, return result to AI, store in history",
		KeySensitiveChoice3:              "3 = Run, return result to AI, do not store in history",
		KeySensitivePressKey:             "Press 1, 2, or 3: ",
		KeyErrLLMNotConfigured:           "LLM not configured. Use /config model or edit llm.api_key (and llm.base_url, llm.model), then send a message again (no restart needed). Supports $VAR or ${VAR} for env. Config path: %s",
		KeyUserLabel:                     "User: ",
		KeyAILabel:                       "AI: ",
		KeyRunLabel:                      "Run: ",
		KeySkillLine:                     "Skill: %s",
		KeyDescExit:                      "Quit delve-shell",
		KeyDescRun:                       "Execute a command directly (no AI)",
		KeyDescSh:                        "Spawn bash",
		KeyDescConfig:                    "Config subcommands",
		KeyDescHelp:                      "Show this help",
		KeyDescConfigLLMBaseURL:          "Set LLM API base URL",
		KeyDescConfigLLMApiKey:           "Set LLM API key",
		KeyDescConfigLLMModel:            "Set LLM model name",
		KeyDescConfigAllowlistUpdate:     "Merge default allowlist",
		KeyDescConfigRemoveRemote:        "Remove a remote",
		KeyAllowlistUpdateDone:           "Allowlist updated: %d new pattern(s) added. Changes apply immediately.",
		KeyModeRequired:                  "Usage: /mode suggest or /mode run",
		KeyRunTagSuggested:               "suggested",
		KeySuggestedCopyHint:             "Select the command above to copy, or use /exec <cmd> to run it.",
		KeySuggestedCardTitle:            "Suggested command (not executed):",
		KeySuggestedCardHint:             "1=copy, 2=dismiss",
		KeySuggestedCopied:               "Copied to clipboard.",
		KeyConfigRemoteAdded:             "Remote added: %s.",
		KeyConfigRemoteRemoved:           "Remote removed: %s.",
		KeyStatusIdle:                    "[IDLE]",
		KeyStatusRunning:                 "[PROCESSING]",
		KeyStatusPendingApproval:         "[NEED APPROVAL]",
		KeyStatusSuggest:                 "[SUGGEST]",
		KeyNeedConfirmationHint:          "Your confirmation required.",
		KeyWizardTitle:                   "=== delve-shell first-time setup ===",
		KeyWizardConfigPath:              "Config path: %s",
		KeyWizardIntroDesc1:              "This wizard will set LLM config (base_url, api_key, model).",
		KeyWizardIntroDesc2:              "",
		KeyWizardIntroEnv:                "Values support $VAR or ${VAR} environment references.",
		KeyWizardBaseURLPrompt:           "LLM base URL (optional; empty uses provider default, e.g. OpenAI official). For OpenAI-compatible APIs, use the /v1 endpoint (e.g. https://api.openai.com/v1): ",
		KeyWizardAPIKeyPrompt:            "LLM api_key (optional; supports $VAR or ${VAR}. Leave empty for local models): ",
		KeyWizardAPIKeyRequired:          "api_key is required. Use an env reference if you prefer not to store the key directly.",
		KeyWizardModelPrompt:             "LLM model (optional; default: gpt-4o-mini): ",
		KeyWizardDone:                    "Config captured; it will be saved and delve-shell will start.",
		KeySessionTitle:                  "=== Session ===",
		KeySessionNew:                    "New session",
		KeySessionPrompt:                 "Choose (0 = new, 1–%d = continue): ",
		KeySessionSwitched:               "Switched to session.",
		KeySessionSwitchedTo:             "Switched to session: %s",
		KeySessionSelect:                 "Up/Down select, Enter switch",
		KeyDescSessions:                  "Switch session",
		KeySessionNone:                   "No previous sessions.",
		KeyRemoteNone:                    "No remotes configured.",
		KeyDelRemoteNoHosts:              "No hosts.",
		KeyDescRemoteOn:                  "Connect to host",
		KeyDescRemoteOff:                 "Disconnect from remote host",
		KeyRemoteManualHint:              "Open remote connection dialog",
		KeyHelpTitle:                     "Help",
		KeyAddRemoteTitle:                "Add Remote",
		KeyConfigSavedLLM:                "Config saved (llm).",
		KeyConfigLLMTitle:                "Config LLM",
		KeyConfigLLMBaseURLLabel:         "Base URL (optional; empty = OpenAI official API):",
		KeyConfigLLMApiKeyLabel:          "API key (optional; supports $VAR):",
		KeyConfigLLMModelLabel:           "Model:",
		KeyConfigLLMMaxMessagesLabel:     "Max context messages (default 50):",
		KeyConfigLLMMaxCharsLabel:        "Max context chars (default: no limit or auto from API):",
		KeyConfigLLMApiKeyRequired:       "API key is required.",
		KeyConfigLLMModelRequired:        "Model is required.",
		KeyConfigLLMChecking:             "Checking...",
		KeyConfigLLMCheckOK:              "LLM check OK.",
		KeyConfigLLMCheckFailed:          "LLM check failed: %v",
		KeyConfigLLMBaseURLAutoCorrected: "Base URL updated to %s (added /v1).",
		KeyDescConfigLLM:                 "Configure model (LLM API)",
		KeyDescSkill:                     "Use skill; optional detail for the AI",
		KeyUsageSkill:                    "Usage: /skill <name> [detail] — text after the name is optional context",
		KeySkillNotFound:                 "Skill not found.",
		KeySkillScriptNotFound:           "Script not found in skill.",
		KeySkillNone:                     "No skills (add dirs with SKILL.md under ~/.delve-shell/skills/)",
		KeyDescSkillInstall:              "Install a skill from a git repo",
		KeyDescSkillRemove:               "Remove an installed skill",
		KeyDescConfigUpdateSkill:         "Update an installed skill from its git source",
		KeyAddSkillTitle:                 "Add skill",
		KeyAddSkillURLLabel:              "Git URL:",
		KeyAddSkillRefLabel:              "Ref — branch or tag:",
		KeyAddSkillPathLabel:             "Path in repo — e.g. skills/foo:",
		KeyAddSkillNameLabel:             "Local skill name:",
		KeyAddSkillAuthHint:              "Private repo: HTTPS — git credential helper or GITHUB_TOKEN; SSH — ssh-add.",
		KeyAddSkillURLRequired:           "URL is required.",
		KeySkillInstalled:                "Skill installed: %s",
		KeySkillRemoved:                  "Skill removed: %s",
		KeySkillInstallFailed:            "Skill install failed: %v",
		KeySkillRemoveFailed:             "Skill remove failed: %v",
		KeyUsageSkillInstall:             "Usage: /config add-skill <url> [ref] [path] — path required if repo has multiple skills (e.g. skills/foo)",
		KeyUsageSkillRemove:              "Usage: /config del-skill <skill_name>",
		KeySkillAlreadyExists:            "Skill already exists. Remove it first or use another name, or use /config update-skill <name> to update it.",
		KeyOverlayFormFooter:             "Up/Down to move · Enter to apply · Esc to cancel",
		KeyOverlayPicklistHint:           "  Up/Down to move · Enter or Tab to apply",
		KeyOverlayUpdateSkillRefTitle:    "Ref · Up/Down to move · Enter to update · Esc to cancel:",
		KeyOverlayEscCancel:              "Esc to cancel",
		KeyOverlayEnterContinueEsc:       "Enter to continue · Esc to cancel",
		KeyOverlay12SelectEsc:            "1 or 2 to select · Esc to cancel",
		KeyOverlayEnterSubmitEsc:         "Enter to submit · Esc to cancel",
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
