package i18n

import (
	"fmt"
	"sync"
)

var (
	langMu      sync.RWMutex
	currentLang = "en"
)

// SetLang sets the active locale for [T] and [Tf]. Empty lang defaults to "en".
func SetLang(lang string) {
	langMu.Lock()
	defer langMu.Unlock()
	if lang == "" {
		currentLang = "en"
		return
	}
	currentLang = lang
}

// Lang returns the active locale set by [SetLang].
func Lang() string {
	langMu.RLock()
	defer langMu.RUnlock()
	return currentLang
}

// Msg keys for user-facing strings. Code error messages stay in English in callers.
const (
	KeyHelpText              = "help_text"
	KeyUsageRun              = "usage_run"
	KeyUnknownCmd            = "unknown_cmd"
	KeyErrorPrefix           = "error_prefix"
	KeyConfigPrefix          = "config_prefix"
	KeyWaitOrCancel          = "wait_or_cancel"
	KeyPlaceholderInput      = "placeholder_input"
	KeyInputHintApproveThree = "input_hint_approve_three" // placeholder when waiting for 1/2/3 (Run/Dismiss/Copy)
	KeyInputHintSensitive    = "input_hint_sensitive"     // placeholder when waiting for 1/2/3 (sensitive)
	KeyInputHistBrowsingHint = "input_hist_browsing_hint" // one line under input while walking local input history
	// Choice menu labels (for Up/Down + Enter selection list)
	KeyChoiceApprove            = "choice_approve"
	KeyChoiceRefuse             = "choice_refuse"
	KeyChoiceRunStore           = "choice_run_store"
	KeyChoiceRunNoStore         = "choice_run_no_store"
	KeyChoiceCopy               = "choice_copy"
	KeyChoiceDismiss            = "choice_dismiss"
	KeyApprovalPrompt           = "approval_prompt"
	KeyApprovalSummary          = "approval_summary"
	KeyApprovalWhy              = "approval_why"
	KeyApprovalDecisionApproved = "approval_decision_approved"
	KeyApprovalDecisionRejected = "approval_decision_rejected"
	KeyRiskReadOnly             = "risk_read_only"
	KeyRiskLow                  = "risk_low"
	KeyRiskHigh                 = "risk_high"
	KeySensitivePrompt          = "sensitive_prompt"
	KeySensitiveChoice1         = "sensitive_choice_1"
	KeySensitiveChoice2         = "sensitive_choice_2"
	KeySensitiveChoice3         = "sensitive_choice_3"
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
	KeyDescConfigAllowlistUpdate = "desc_config_allowlist_update"
	KeyDescConfigRemoveRemote    = "desc_config_remove_remote"
	KeyAllowlistUpdateDone       = "allowlist_update_done" // format: added count
	KeyRunTagSuggested           = "run_tag_suggested"
	KeySuggestedCopyHint         = "suggested_copy_hint"
	KeySuggestedCopied           = "suggested_copied"
	KeyConfigRemoteAdded         = "config_remote_added"   // format: name, target
	KeyConfigRemoteRemoved       = "config_remote_removed" // format: name

	// Status bar (title): IDLE / RUNNING / pending approval / suggest card
	KeyStatusIdle             = "status_idle"
	KeyStatusRunning          = "status_running"
	KeyStatusPendingApproval  = "status_pending_approval"
	KeyStatusWaitingUserInput = "status_waiting_user_input"
	KeyStatusSuggest          = "status_suggest"

	KeySessionNew                    = "session_new"
	KeySessionSwitchedTo             = "session_switched_to"    // format: "Switched to session: %s" (session id; /new banner)
	KeyHistorySwitchedTo             = "history_switched_to"    // format: after /history <id>; transcript is not loaded
	KeyHistoryPreviewTitle           = "history_preview_title"  // format: overlay title, e.g. "History · %s"
	KeyHistoryPreviewEmpty           = "history_preview_empty"  // overlay body when file has no lines yet
	KeyHistoryPreviewFooter          = "history_preview_footer" // hint under preview (Esc / scroll)
	KeyDescSessions                  = "desc_sessions"          // slash: /history description
	KeySessionNone                   = "session_none"
	KeyDelRemoteNoHosts              = "del_remote_no_hosts" // slash dropdown when no remotes to remove (Cmd-only row, like KeySkillNone)
	KeyDescRemoteOn                  = "desc_remote_on"
	KeyDescRemoteOff                 = "desc_remote_off"
	KeyDescAccessOffline             = "desc_access_offline"
	KeyOfflinePasteTitle             = "offline_paste_title"
	KeyOfflinePasteIntro             = "offline_paste_intro"
	KeyOfflinePasteReview            = "offline_paste_review"
	KeyOfflinePasteHint              = "offline_paste_hint"
	KeyOfflinePasteCopyFailed        = "offline_paste_copy_failed"
	KeyOfflinePastePlaceholder       = "offline_paste_placeholder"
	KeyOfflineExecBashDisabled       = "offline_exec_bash_disabled"
	KeyOfflineSlashExecDisabled      = "offline_slash_exec_disabled"
	KeyOfflineSlashSkillDisabled     = "offline_slash_skill_disabled"
	KeyHelpTitle                     = "help_title"
	KeyAddRemoteTitle                = "add_remote_title"
	KeyAddRemoteScreenTitle          = "add_remote_screen_title"
	KeyAddRemoteConnecting           = "add_remote_connecting"
	KeyAddRemoteOverwriteHint        = "add_remote_overwrite_hint"
	KeyAddRemoteHostLabel            = "add_remote_host_label"
	KeyAddRemoteUserLabel            = "add_remote_user_label"
	KeyAddRemoteKeyLabel             = "add_remote_key_label"
	KeyAddRemoteSaveLabel            = "add_remote_save_label"
	KeyAddRemoteNameLabel            = "add_remote_name_label"
	KeyRemoteTitleBarRemote          = "remote_title_bar_remote"
	KeyRemoteTitleBarOffline         = "remote_title_bar_offline"
	KeyRemoteAuthTitle               = "remote_auth_title"
	KeyAddRemoteHostPlaceholder      = "add_remote_host_placeholder"
	KeyAddRemoteUserPlaceholder      = "add_remote_user_placeholder"
	KeyAddRemoteKeyPlaceholder       = "add_remote_key_placeholder"
	KeyAddRemoteNamePlaceholder      = "add_remote_name_placeholder"
	KeyRemoteAuthPasswordPlaceholder = "remote_auth_password_placeholder"
	KeyRemoteAuthIdentityPlaceholder = "remote_auth_identity_placeholder"
	KeyRemoteAuthUsernameTitle       = "remote_auth_username_title"
	KeyRemoteAuthMethodTitle         = "remote_auth_method_title"
	KeyRemoteAuthPasswordChoice      = "remote_auth_password_choice"
	KeyRemoteAuthIdentityChoice      = "remote_auth_identity_choice"
	KeyRemoteAuthPasswordTitle       = "remote_auth_password_title"
	KeyRemoteAuthIdentityTitle       = "remote_auth_identity_title"
	KeyRemoteAuthAutoIdentityTitle   = "remote_auth_auto_identity_title"
	KeyRemoteAuthHostKeyTitle        = "remote_auth_host_key_title"
	KeyRemoteAuthConnecting          = "remote_auth_connecting"
	KeyRemoteAuthConfiguredKey       = "remote_auth_configured_key"
	KeyRemoteAuthKnownHostsUpdate    = "remote_auth_known_hosts_update"
	KeyRemoteAuthAcceptKnownHosts    = "remote_auth_accept_known_hosts"
	KeyRemoteAuthRejectKnownHosts    = "remote_auth_reject_known_hosts"
	KeyRemoteAuthTargetLabel         = "remote_auth_target_label"
	KeyRemoteAuthFingerprintLabel    = "remote_auth_fingerprint_label"
	KeyRemoteAuthHostKeyUnknown      = "remote_auth_host_key_unknown"
	KeyRemoteAuthHostKeyMismatch     = "remote_auth_host_key_mismatch"
	KeyTitleBarLocal                 = "title_bar_local"
	KeyConfigSavedLLM                = "config_saved_llm"
	KeyConfigLLMTitle                = "config_llm_title"
	KeyConfigLLMBaseURLLabel         = "config_llm_base_url_label"
	KeyConfigLLMApiKeyLabel          = "config_llm_api_key_label"
	KeyConfigLLMBaseURLPlaceholder   = "config_llm_base_url_placeholder"
	KeyConfigLLMApiKeyPlaceholder    = "config_llm_api_key_placeholder"
	KeyConfigLLMModelLabel           = "config_llm_model_label"
	KeyConfigLLMMaxMessagesLabel     = "config_llm_max_messages_label"
	KeyConfigLLMMaxCharsLabel        = "config_llm_max_chars_label"
	KeyConfigLLMModelRequired        = "config_llm_model_required"
	KeyConfigLLMChecking             = "config_llm_checking"                // "Checking..."
	KeyConfigLLMCheckOK              = "config_llm_check_ok"                // after save: "LLM check OK."
	KeyConfigLLMCheckFailed          = "config_llm_check_failed"            // format: "LLM check failed: %v"
	KeyConfigLLMBaseURLAutoCorrected = "config_llm_base_url_auto_corrected" // format: "Base URL updated to %s (added /v1)."
	KeyDescConfigLLM                 = "desc_config_llm"
	// Skill
	KeyDescSkill                     = "desc_skill"
	KeyUsageSkill                    = "usage_skill"
	KeySkillNotFound                 = "skill_not_found"
	KeySkillNone                     = "skill_none"
	KeyDescSkillInstall              = "desc_skill_install"
	KeyDescSkillRemove               = "desc_skill_remove"
	KeyDescConfigUpdateSkill         = "desc_config_update_skill"
	KeyAddSkillTitle                 = "add_skill_title"
	KeyAddSkillURLLabel              = "add_skill_url_label"
	KeyAddSkillRefLabel              = "add_skill_ref_label"
	KeyAddSkillPathLabel             = "add_skill_path_label"
	KeyAddSkillNameLabel             = "add_skill_name_label"
	KeyAddSkillURLRequired           = "add_skill_url_required"
	KeyAddSkillURLPlaceholder        = "add_skill_url_placeholder"
	KeyAddSkillRefPlaceholder        = "add_skill_ref_placeholder"
	KeyAddSkillPathPlaceholder       = "add_skill_path_placeholder"
	KeyAddSkillNamePlaceholder       = "add_skill_name_placeholder"
	KeyUpdateSkillTitle              = "update_skill_title"
	KeyUpdateSkillSkillLabel         = "update_skill_skill_label"
	KeyUpdateSkillURLLabel           = "update_skill_url_label"
	KeyUpdateSkillPathLabel          = "update_skill_path_label"
	KeyUpdateSkillCurrentCommitLabel = "update_skill_current_commit_label"
	KeyUpdateSkillLatestCommitLabel  = "update_skill_latest_commit_label"
	KeySkillInstalled                = "skill_installed"
	KeySkillRemoved                  = "skill_removed"
	KeySkillInstallFailed            = "skill_install_failed"
	KeySkillRemoveFailed             = "skill_remove_failed"
	KeyUsageSkillRemove              = "usage_skill_remove"
	KeySkillAlreadyExists            = "skill_already_exists"

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
		KeyHelpText: `delve-shell — AI-assisted ops. Commands run only after HIL approval (cards or allowlist path).

What it does:
  Natural-language tasks drive suggested commands. Allowlisted commands with no shell write redirection run without a card; others show a card (Run, Dismiss, or Copy). An empty allowlist matches nothing, so every command shows the card. Runs are recorded in session history for audit.

Quick start:
  1. Enter a task in the input line and press Enter to send.
  2. Multi-line input: Shift+Enter, Alt+Enter, or Ctrl+J inserts a newline; Enter sends. Many terminals map Shift+Enter like Enter—Alt+Enter or Ctrl+J is the reliable newline.
  3. On a command card: 1 runs, 2 dismisses without running, 3 copies the command.
  4. Up/Down recall recent submitted lines (chat and slash). While a recalled line starts with /, Up/Down continues history; slash completion resumes after any non–↑/↓ key or after leaving history browse.
  5. / opens slash suggestions (Up/Down on a / line; Tab or Enter inserts the highlighted row; Enter submits a complete slash command).
  6. PgUp/PgDown scrolls the log; /help opens this panel.

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

/history
List and switch history sessions. Flow: /history → pick a row (Tab/Enter fills /history <id>) → submit opens a read-only preview → Enter in the dialog switches the active session; Esc closes without switching. Only the first word after /history is the session id (trailing text is ignored). Dropdown lines show a one-line summary of the first turn (line breaks as \n; long text ends with ...).

/skill <name> [detail]
Use skill; optional detail for the AI

/exec <cmd>
Run one command directly (no AI)

` + helpEnBashSection + `/quit
Quit (Ctrl+C also works)`,
		KeyUsageRun:                      "Usage: /exec <command> (for example: /exec ls -la)",
		KeyUnknownCmd:                    "Unknown command. Type /help for the full list.",
		KeyDelveLabel:                    "Delve:",
		KeyErrorPrefix:                   "Error: ",
		KeyConfigPrefix:                  "Config: ",
		KeyWaitOrCancel:                  "(Please wait for the current response, or press Esc to cancel)",
		KeyPlaceholderInput:              "Type your question or / for slash commands.",
		KeyInputHistBrowsingHint:         "↑/↓ input history · Enter to send · any other key edits",
		KeyInputHintApproveThree:         "1, 2 or 3",
		KeyInputHintSensitive:            "1, 2 or 3",
		KeyChoiceApprove:                 "Approve",
		KeyChoiceRefuse:                  "Refuse (do not run)",
		KeyChoiceRunStore:                "Run, return to AI, store in history",
		KeyChoiceRunNoStore:              "Run, return to AI, do not store",
		KeyChoiceCopy:                    "Copy",
		KeyChoiceDismiss:                 "Dismiss",
		KeyApprovalPrompt:                "Command to run (approval required):",
		KeyApprovalSummary:               "Summary:",
		KeyApprovalWhy:                   "Why:",
		KeyApprovalDecisionApproved:      "Decision: approved",
		KeyApprovalDecisionRejected:      "Decision: rejected",
		KeyRiskReadOnly:                  "READ-ONLY",
		KeyRiskLow:                       "LOW-RISK",
		KeyRiskHigh:                      "HIGH-RISK",
		KeySensitivePrompt:               "This command may access sensitive files. Choose:",
		KeySensitiveChoice1:              "1 = Refuse (do not run)",
		KeySensitiveChoice2:              "2 = Run, return result to AI, store in history",
		KeySensitiveChoice3:              "3 = Run, return result to AI, do not store in history",
		KeyUserLabel:                     "User: ",
		KeyAILabel:                       "AI: ",
		KeyRunLabel:                      "Run: ",
		KeySkillLine:                     "Skill: %s",
		KeyDescExit:                      "Quit delve-shell",
		KeyDescRun:                       "Execute a command directly (no AI)",
		KeyDescSh:                        "Spawn bash",
		KeyDescConfig:                    "Config subcommands",
		KeyDescHelp:                      "Show this help",
		KeyDescConfigAllowlistUpdate:     "Merge default allowlist",
		KeyDescConfigRemoveRemote:        "Remove a remote",
		KeyAllowlistUpdateDone:           "Allowlist updated: %d new pattern(s) added. Changes apply immediately.",
		KeyRunTagSuggested:               "suggested",
		KeySuggestedCopyHint:             "Select the command above to copy, or use /exec <cmd> to run it.",
		KeySuggestedCopied:               "Copied to clipboard.",
		KeyConfigRemoteAdded:             "Remote added: %s.",
		KeyConfigRemoteRemoved:           "Remote removed: %s.",
		KeyStatusIdle:                    "[IDLE]",
		KeyStatusRunning:                 "[PROCESSING]",
		KeyStatusPendingApproval:         "[NEED APPROVAL]",
		KeyStatusWaitingUserInput:        "[WAIT INPUT]",
		KeyStatusSuggest:                 "[SUGGEST]",
		KeySessionNew:                    "New session",
		KeySessionSwitchedTo:             "Switched to session: %s",
		KeyHistorySwitchedTo:             "Switched. Active history: %s",
		KeyHistoryPreviewTitle:           "History · %s",
		KeyHistoryPreviewEmpty:           "(No messages in this history yet.)",
		KeyHistoryPreviewFooter:          "Enter to switch · PgUp/PgDn to scroll · Esc to cancel",
		KeyDescSessions:                  "List and switch history sessions",
		KeySessionNone:                   "No previous history.",
		KeyDelRemoteNoHosts:              "No hosts.",
		KeyDescRemoteOn:                  "Connect to host",
		KeyDescRemoteOff:                 "Disconnect from remote host",
		KeyDescAccessOffline:             "Offline mode (paste results back)",
		KeyOfflinePasteTitle:             "Offline — paste output in the box below",
		KeyOfflinePasteIntro:             "This command is not run here. Paste the results back after you run it elsewhere.",
		KeyOfflinePasteReview:            "Review the command before running it elsewhere.",
		KeyOfflinePasteHint:              "Enter: submit · Esc: cancel",
		KeyOfflinePasteCopyFailed:        "Could not copy to clipboard. Select the command line above or copy manually.",
		KeyOfflinePastePlaceholder:       "Paste output",
		KeyOfflineExecBashDisabled:       "/bash is not available in Offline mode.",
		KeyOfflineSlashExecDisabled:      "/exec is not available in Offline mode.",
		KeyOfflineSlashSkillDisabled:     "/skill is not available in Offline mode.",
		KeyHelpTitle:                     "Help",
		KeyAddRemoteTitle:                "Add Remote",
		KeyAddRemoteScreenTitle:          "Add remote",
		KeyAddRemoteConnecting:           "Connecting...",
		KeyAddRemoteOverwriteHint:        "Press y to overwrite, or edit the host or username and try again.",
		KeyAddRemoteHostLabel:            "Host (address or host:port):",
		KeyAddRemoteUserLabel:            "Username:",
		KeyAddRemoteKeyLabel:             "Key path (optional):",
		KeyAddRemoteSaveLabel:            "Save as remote (Space to toggle)",
		KeyAddRemoteNameLabel:            "Name (optional):",
		KeyRemoteTitleBarRemote:          "Remote",
		KeyRemoteTitleBarOffline:         "Offline",
		KeyRemoteAuthTitle:               "Remote Auth",
		KeyAddRemoteHostPlaceholder:      "host or host:22",
		KeyAddRemoteUserPlaceholder:      "e.g. root",
		KeyAddRemoteKeyPlaceholder:       "~/.ssh/id_rsa (optional)",
		KeyAddRemoteNamePlaceholder:      "name (optional)",
		KeyRemoteAuthPasswordPlaceholder: "SSH password",
		KeyRemoteAuthIdentityPlaceholder: "~/.ssh/id_rsa",
		KeyRemoteAuthUsernameTitle:       "SSH auth for %s",
		KeyRemoteAuthMethodTitle:         "Choose authentication method:",
		KeyRemoteAuthPasswordChoice:      "1. Password",
		KeyRemoteAuthIdentityChoice:      "2. Key file (identity file)",
		KeyRemoteAuthPasswordTitle:       "SSH password for %s",
		KeyRemoteAuthIdentityTitle:       "SSH key file path for %s",
		KeyRemoteAuthAutoIdentityTitle:   "SSH auth for %s",
		KeyRemoteAuthHostKeyTitle:        "Host key verification",
		KeyRemoteAuthConnecting:          "Connecting...",
		KeyRemoteAuthConfiguredKey:       "Connecting with configured SSH key...",
		KeyRemoteAuthKnownHostsUpdate:    "Updating known_hosts and reconnecting...",
		KeyRemoteAuthAcceptKnownHosts:    "1. Accept and update known_hosts",
		KeyRemoteAuthRejectKnownHosts:    "2. Reject and abort",
		KeyRemoteAuthTargetLabel:         "Target: %s",
		KeyRemoteAuthFingerprintLabel:    "Fingerprint: %s",
		KeyRemoteAuthHostKeyUnknown:      "Host key is not trusted yet. Accept to add/update known_hosts or reject to abort.",
		KeyRemoteAuthHostKeyMismatch:     "Host key mismatch detected. Accept to update known_hosts or reject to abort.",
		KeyTitleBarLocal:                 "Local",
		KeyConfigSavedLLM:                "Config saved (llm).",
		KeyConfigLLMTitle:                "Config LLM",
		KeyConfigLLMBaseURLLabel:         "Base URL (optional; empty = OpenAI official API):",
		KeyConfigLLMApiKeyLabel:          "API key (optional; supports $VAR):",
		KeyConfigLLMBaseURLPlaceholder:   "https://api.openai.com/v1 (optional)",
		KeyConfigLLMApiKeyPlaceholder:    "sk-... or $API_KEY",
		KeyConfigLLMModelLabel:           "Model:",
		KeyConfigLLMMaxMessagesLabel:     "Max context messages (default 50):",
		KeyConfigLLMMaxCharsLabel:        "Max context chars (default: no limit or auto from API):",
		KeyConfigLLMModelRequired:        "Model is required.",
		KeyConfigLLMChecking:             "Checking...",
		KeyConfigLLMCheckOK:              "LLM check OK.",
		KeyConfigLLMCheckFailed:          "LLM check failed: %v",
		KeyConfigLLMBaseURLAutoCorrected: "Base URL updated to %s (added /v1).",
		KeyDescConfigLLM:                 "Configure model (LLM API)",
		KeyDescSkill:                     "Use skill; optional detail for the AI",
		KeyUsageSkill:                    "Usage: /skill <name> [detail] (text after the name is optional)",
		KeySkillNotFound:                 "Skill not found.",
		KeySkillNone:                     "No skills (add dirs with SKILL.md under ~/.delve-shell/skills/)",
		KeyDescSkillInstall:              "Install a skill from a git repo",
		KeyDescSkillRemove:               "Remove an installed skill",
		KeyDescConfigUpdateSkill:         "Update an installed skill from its git source",
		KeyAddSkillTitle:                 "Add skill",
		KeyAddSkillURLLabel:              "Git URL:",
		KeyAddSkillRefLabel:              "Ref — branch or tag:",
		KeyAddSkillPathLabel:             "Path in repo — e.g. skills/foo:",
		KeyAddSkillNameLabel:             "Local skill name:",
		KeyAddSkillURLRequired:           "URL is required.",
		KeyAddSkillURLPlaceholder:        "https://github.com/owner/repo or owner/repo",
		KeyAddSkillRefPlaceholder:        "main",
		KeyAddSkillPathPlaceholder:       "skills/foo",
		KeyAddSkillNamePlaceholder:       "local skill name",
		KeyUpdateSkillTitle:              "Update skill",
		KeyUpdateSkillSkillLabel:         "Skill: %s",
		KeyUpdateSkillURLLabel:           "URL:   %s",
		KeyUpdateSkillPathLabel:          "Path:  %s",
		KeyUpdateSkillCurrentCommitLabel: "Current commit: %s",
		KeyUpdateSkillLatestCommitLabel:  "Latest commit:  %s",
		KeySkillInstalled:                "Skill installed: %s",
		KeySkillRemoved:                  "Skill removed: %s",
		KeySkillInstallFailed:            "Skill install failed: %v",
		KeySkillRemoveFailed:             "Skill remove failed: %v",
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

// T returns the message for key in the active language ([SetLang]). If the locale or key is missing, falls back to "en" then key as-is.
func T(key string) string {
	langMu.RLock()
	lang := currentLang
	langMu.RUnlock()
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

// Tf returns fmt.Sprintf(T(key), a...). Use only when the message for key is a format string.
func Tf(key string, a ...interface{}) string {
	return fmt.Sprintf(T(key), a...)
}
