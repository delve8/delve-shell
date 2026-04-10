package i18n

import (
	"fmt"
	"strings"
	"sync"
)

var (
	langMu      sync.RWMutex
	currentLang = "en"
)

const (
	helpConfigSectionEN            = `Manage models, hosts and skills.`
	helpConfigDelRemoteSectionEN   = `Remove a remote host.`
	helpConfigDelSkillSectionEN    = `Remove an installed skill.`
	helpConfigUpdateSkillSectionEN = `Update an installed skill.`
	helpConfigModelSectionEN       = `Configure model settings.`
	helpAccessSectionEN            = `Switch the execution target. The dropdown lists saved hosts first, then **/access New** (add a host), **/access Local** (use the local executor), and **/access Offline** (manual relay mode). Host segments in saved targets must be lowercase so reserved rows like **/access Local** and **/access New** do not collide with host names.`
	helpAccessNewSectionEN         = `Add a remote host.`
	helpAccessLocalSectionEN       = `Switch to local execution.`
	helpAccessHostSectionEN        = `Connect to a saved host or enter a new host.`
	helpNewSectionEN               = `Start a new session.`
	helpHistorySectionEN           = `Browse and switch sessions. Flow: **/history** → pick a row (Tab/Enter fills **/history {id}**) → submit opens a read-only preview → Enter in the dialog switches the active session; Esc closes without switching. Only the first word after **/history** is the session id (trailing text is ignored). Dropdown lines show a one-line summary of the first turn.`
	helpSkillSectionEN             = `Use an installed skill for the current turn. Skills are directories under **~/.delve-shell/skills/** (each with **SKILL.md**). Type **/skill** to open the slash list of installed skills plus **/skill New** (install dialog), or type **/skill** followed by the skill directory name. Text after the first word (the skill name) is optional; when present it is passed to the AI as extra context for that turn. To install or remove skills, use **/skill New** and **/config del-skill**.`
	helpSkillNameSectionEN         = `Use a skill. Optional text after the name is passed to the AI for this turn.`
	helpSkillNewSectionEN          = `Open the add-skill dialog (Git URL, ref, path in repo, local name).`
	helpHelpSectionEN              = `Show help.`
	helpQuitSectionEN              = `Quit (**Ctrl+C** also works).`
)

func englishHelpText() string {
	var b strings.Builder
	b.WriteString(`# delve-shell

AI-assisted ops. Commands run only after HIL approval (cards or allowlist path).

## What it does

Natural-language tasks drive suggested commands. Allowlisted commands with no shell write redirection run without a card; others show a card (Run, Dismiss, or Copy). An empty allowlist matches nothing, so every command shows the card. Runs are recorded in session history for audit.

## Quick start

1. Enter a task in the input line and press **Enter** to send.
2. Multi-line input: **Shift+Enter**, **Alt+Enter**, or **Ctrl+J** inserts a newline; **Enter** sends. Many terminals map Shift+Enter like Enter—**Alt+Enter** or **Ctrl+J** is the reliable newline.
3. On a command card: **1** runs, **2** dismisses without running, **3** copies the command.
4. **Up/Down** recall recent submitted lines (chat and slash). While a recalled line starts with a slash, Up/Down continues history; slash completion resumes after any other key or after leaving history browse.
5. Type **/** to open slash suggestions (Up/Down on a slash line; Tab or Enter inserts the highlighted row; Enter submits a complete slash command).
6. **PgUp/PgDown** scrolls the log; **/help** opens this panel.

## Slash commands

**/exec** is not listed in the **/** menu; type **/exec {cmd}** directly when you need a one-off command without the AI.

### /help

` + helpHelpSectionEN + `

### /config

` + helpConfigSectionEN + `

### /config del-remote

` + helpConfigDelRemoteSectionEN + `

### /config del-skill {skill_name}

` + helpConfigDelSkillSectionEN + `

### /config update-skill {skill_name}

` + helpConfigUpdateSkillSectionEN + `

### /config model

` + helpConfigModelSectionEN + `

### /access

` + helpAccessSectionEN + `

### /access New

` + helpAccessNewSectionEN + `

### /access Local

` + helpAccessLocalSectionEN + `

### /access {user@host or host}

` + helpAccessHostSectionEN + `

### /new

` + helpNewSectionEN + `

### /history

` + helpHistorySectionEN + `

### /skill

` + helpSkillSectionEN + `

### /skill {name} [text]

` + helpSkillNameSectionEN + `

### /skill New

` + helpSkillNewSectionEN + `

` + helpEnBashSection + `### /quit

` + helpQuitSectionEN + `
`)
	return b.String()
}

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
	KeyHelpText                          = "help_text"
	KeyUsageRun                          = "usage_run"
	KeyUnknownCmd                        = "unknown_cmd"
	KeyErrorPrefix                       = "error_prefix"
	KeyConfigPrefix                      = "config_prefix"
	KeyWaitOrCancel                      = "wait_or_cancel"
	KeyCommandExecWaitOrCancel           = "command_exec_wait_or_cancel"
	KeyExecStreamPreviewHeader           = "exec_stream_preview_header"
	KeyExecStreamTranscriptTruncatedHint = "exec_stream_transcript_truncated_hint"
	KeyTranscriptReplayTruncatedNotice   = "transcript_replay_truncated_notice"
	KeyPlaceholderInput                  = "placeholder_input"
	KeyInputHintApproveThree             = "input_hint_approve_three" // placeholder when waiting for 1/2/3 (Run/Dismiss/Copy)
	KeyInputHintSensitive                = "input_hint_sensitive"     // placeholder when waiting for 1/2/3 (sensitive)
	KeyInputHistBrowsingHint             = "input_hist_browsing_hint" // one line under input while walking local input history
	// Choice menu labels (for Up/Down + Enter selection list)
	KeyChoiceApprove             = "choice_approve"
	KeyChoiceRefuse              = "choice_refuse"
	KeyChoiceRunStore            = "choice_run_store"
	KeyChoiceRunNoStore          = "choice_run_no_store"
	KeyChoiceCopy                = "choice_copy"
	KeyChoiceDismiss             = "choice_dismiss"
	KeyApprovalPrompt            = "approval_prompt"
	KeyApprovalSummary           = "approval_summary"
	KeyApprovalAutoApprovePolicy = "approval_auto_approve_policy" // section label before auto-approve Risk reason lines (Risk Hint)
	KeyApprovalWhy               = "approval_why"                 // label before user-stated run purpose
	KeyApprovalDecisionApproved  = "approval_decision_approved"
	KeyApprovalDecisionRejected  = "approval_decision_rejected"
	KeyRiskReadOnly              = "risk_read_only"
	KeyRiskLow                   = "risk_low"
	KeyRiskHigh                  = "risk_high"
	KeySensitivePrompt           = "sensitive_prompt"
	KeySensitiveChoice1          = "sensitive_choice_1"
	KeySensitiveChoice2          = "sensitive_choice_2"
	KeySensitiveChoice3          = "sensitive_choice_3"
	KeyUserLabel                 = "user_label"             // legacy transcript prefix; prefer KeyTranscriptUserPrompt in UI
	KeyAILabel                   = "ai_label"               // legacy; AI transcript no longer prefixes with this in TUI
	KeyTranscriptUserPrompt      = "transcript_user_prompt" // same as input prompt, e.g. "> "
	KeyInfoLabel                 = "info_label"             // non-error system hints, e.g. "Info: "
	KeyAgentReplyEmpty           = "agent_reply_empty"      // model finished with no assistant text (API/empty parse)
	KeyDelveLabel                = "delve_label"            // deprecated; use KeyInfoLabel for transcript hints
	KeyRunLabel                  = "run_label"              // legacy; prefer KeyRunLine* for execute transcript
	// Run line prefixes (execute_command / history replay); command follows; total width capped in UI.
	KeyRunLineAutoAllowed   = "run_line_auto_allowed" // built-in checks passed; no user approval card
	KeyRunLineApproved      = "run_line_approved"
	KeyRunLineDirect        = "run_line_direct"
	KeyRunLineOfflineManual = "run_line_offline_manual"
	KeyRunLineSuggested     = "run_line_suggested"
	KeySkillLine            = "skill_line" // format: Skill: %s
	// Slash option descriptions (cmd as suffix for consistency)
	KeyDescExit   = "desc_exit"
	KeyDescRun    = "desc_run"
	KeyDescSh     = "desc_sh"
	KeyDescConfig = "desc_config"
	KeyDescHelp   = "desc_help"
	// /config 子项说明（仅在选择 /config 后显示）
	KeyDescConfigRemoveRemote = "desc_config_remove_remote"
	KeyRunTagSuggested        = "run_tag_suggested"
	KeySuggestedCopyHint      = "suggested_copy_hint"
	KeySuggestedCopied        = "suggested_copied"
	KeyConfigRemoteAdded      = "config_remote_added"   // format: name, target
	KeyConfigRemoteRemoved    = "config_remote_removed" // format: name

	// Status bar (title): IDLE / RUNNING / pending approval / suggest card
	KeyStatusIdle             = "status_idle"
	KeyStatusExecuting        = "status_executing"
	KeyStatusRunning          = "status_running"
	KeyStatusPendingApproval  = "status_pending_approval"
	KeyStatusWaitingUserInput = "status_waiting_user_input"
	KeyStatusSuggest          = "status_suggest"

	KeySessionNew           = "session_new"
	KeySessionSwitchedTo    = "session_switched_to"    // format: "Switched to session: %s" (session id; /new banner)
	KeyHistorySwitchedTo    = "history_switched_to"    // format: after /history <id>; transcript is not loaded
	KeyHistoryPreviewTitle  = "history_preview_title"  // format: overlay title, e.g. "History · %s"
	KeyHistoryPreviewEmpty  = "history_preview_empty"  // overlay body when file has no lines yet
	KeyHistoryPreviewFooter = "history_preview_footer" // hint under preview (Esc / scroll)
	KeyDescSessions         = "desc_sessions"          // slash: /history description
	KeyDescNewSession       = "desc_new_session"
	// KeyHistorySessionCurrentSuffix is appended after the session id in the /history picker Cmd (e.g. " [Current]").
	KeyHistorySessionCurrentSuffix = "history_session_current_suffix"
	KeySessionNone                 = "session_none"
	KeyDescAccess                  = "desc_access"
	KeyDescAccessNew               = "desc_access_new"
	KeyDelRemoteNoHosts            = "del_remote_no_hosts" // slash dropdown when no remotes to remove (Cmd-only row, like KeySkillNone)
	KeyDescRemoteOn                = "desc_remote_on"
	KeyDescRemoteOff               = "desc_remote_off"
	KeyDescAccessOffline           = "desc_access_offline"
	KeyOfflinePasteTitle           = "offline_paste_title"
	KeyOfflinePasteIntro           = "offline_paste_intro"
	KeyOfflinePasteReview          = "offline_paste_review"
	KeyOfflinePasteHint            = "offline_paste_hint"
	KeyOfflinePasteCopyFailed      = "offline_paste_copy_failed"
	KeyOfflinePastePlaceholder     = "offline_paste_placeholder"
	KeyOfflineExecBashDisabled     = "offline_exec_bash_disabled"
	KeyBashReturnNotice            = "bash_return_notice"
	KeyOfflineSlashExecDisabled    = "offline_slash_exec_disabled"
	KeyOfflineSlashSkillDisabled   = "offline_slash_skill_disabled"
	KeySkillScriptsSyncRemote      = "skill_scripts_sync_remote"
	KeyHelpTitle                   = "help_title"
	// KeyHelpOverlayFooter is fixed below the help scroll area (same chrome as history preview).
	KeyHelpOverlayFooter               = "help_overlay_footer"
	KeyAddRemoteTitle                  = "add_remote_title"
	KeyAddRemoteScreenTitle            = "add_remote_screen_title"
	KeyAddRemoteConnecting             = "add_remote_connecting"
	KeyAddRemoteOverwriteHint          = "add_remote_overwrite_hint"
	KeyAddRemoteOverwriteChoice        = "add_remote_overwrite_choice"
	KeyAddRemoteKeepEditingChoice      = "add_remote_keep_editing_choice"
	KeyAddRemoteHostLabel              = "add_remote_host_label"
	KeyAddRemoteUserLabel              = "add_remote_user_label"
	KeyAddRemoteKeyLabel               = "add_remote_key_label"
	KeyAddRemoteSaveLabel              = "add_remote_save_label"
	KeyAddRemoteNameLabel              = "add_remote_name_label"
	KeyRemoteTitleBarRemote            = "remote_title_bar_remote"
	KeyRemoteTitleBarOffline           = "remote_title_bar_offline"
	KeyRemoteAuthTitle                 = "remote_auth_title"
	KeyAddRemoteHostPlaceholder        = "add_remote_host_placeholder"
	KeyAddRemoteUserPlaceholder        = "add_remote_user_placeholder"
	KeyAddRemoteKeyPlaceholder         = "add_remote_key_placeholder"
	KeyAddRemoteNamePlaceholder        = "add_remote_name_placeholder"
	KeyRemoteAuthPasswordPlaceholder   = "remote_auth_password_placeholder"
	KeyRemoteAuthIdentityPlaceholder   = "remote_auth_identity_placeholder"
	KeyRemoteAuthUsernameTitle         = "remote_auth_username_title"
	KeyRemoteAuthMethodTitle           = "remote_auth_method_title"
	KeyRemoteAuthPasswordChoice        = "remote_auth_password_choice"
	KeyRemoteAuthIdentityChoice        = "remote_auth_identity_choice"
	KeyRemoteAuthPasswordTitle         = "remote_auth_password_title"
	KeyRemoteAuthIdentityTitle         = "remote_auth_identity_title"
	KeyRemoteAuthAutoIdentityTitle     = "remote_auth_auto_identity_title"
	KeyRemoteAuthHostKeyTitle          = "remote_auth_host_key_title"
	KeyRemoteAuthConnecting            = "remote_auth_connecting"
	KeyRemoteAuthConfiguredKey         = "remote_auth_configured_key"
	KeyRemoteAuthKnownHostsUpdate      = "remote_auth_known_hosts_update"
	KeyRemoteAuthAcceptKnownHosts      = "remote_auth_accept_known_hosts"
	KeyRemoteAuthRejectKnownHosts      = "remote_auth_reject_known_hosts"
	KeyRemoteAuthTargetLabel           = "remote_auth_target_label"
	KeyRemoteAuthFingerprintLabel      = "remote_auth_fingerprint_label"
	KeyRemoteAuthHostKeyUnknown        = "remote_auth_host_key_unknown"
	KeyRemoteAuthHostKeyMismatch       = "remote_auth_host_key_mismatch"
	KeyTitleBarLocal                   = "title_bar_local"
	KeyConfigSavedModel                = "config_saved_model"
	KeyConfigModelTitle                = "config_model_title"
	KeyConfigModelBaseURLLabel         = "config_model_base_url_label"
	KeyConfigModelApiKeyLabel          = "config_model_api_key_label"
	KeyConfigModelBaseURLPlaceholder   = "config_model_base_url_placeholder"
	KeyConfigModelApiKeyPlaceholder    = "config_model_api_key_placeholder"
	KeyConfigModelModelLabel           = "config_model_model_label"
	KeyConfigModelMaxMessagesLabel     = "config_model_max_messages_label"
	KeyConfigModelMaxCharsLabel        = "config_model_max_chars_label"
	KeyConfigModelModelRequired        = "config_model_model_required"
	KeyConfigModelChecking             = "config_model_checking"                // "Checking..."
	KeyConfigModelCheckOK              = "config_model_check_ok"                // after save connectivity check succeeded
	KeyConfigModelCheckFailed          = "config_model_check_failed"            // format: "Model check failed: %v"
	KeyConfigModelBaseURLAutoCorrected = "config_model_base_url_auto_corrected" // format: "Base URL updated to %s (added /v1)."
	KeyDescConfigModel                 = "desc_config_model"
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
	KeyOverlayChoiceSelectEsc     = "overlay_choice_select_esc"
	KeyOverlayEnterSubmitEsc      = "overlay_enter_submit_esc"
	KeyOverlayEnterUpdateEsc      = "overlay_enter_update_esc"

	// Auto-approve highlight: why a span is Risk ([T] / [Tf] for localized Reason text).
	KeyAutoApproveHLWriteRedirection        = "auto_approve_hl_write_redirection"
	KeyAutoApproveHLShellParseError         = "auto_approve_hl_shell_parse_error"
	KeyAutoApproveHLUnsupportedConstruct    = "auto_approve_hl_unsupported_construct"
	KeyAutoApproveHLExpansionNotAllowed     = "auto_approve_hl_expansion_not_allowed"
	KeyAutoApproveHLEmptySegment            = "auto_approve_hl_empty_segment"
	KeyAutoApproveHLAllowlistNotLoaded      = "auto_approve_hl_allowlist_not_loaded"
	KeyAutoApproveHLCommandNotInAllowlist   = "auto_approve_hl_command_not_in_allowlist"
	KeyAutoApproveHLArgsPolicyMismatch      = "auto_approve_hl_args_policy_mismatch"
	KeyAutoApproveHLOpaqueArgv0             = "auto_approve_hl_opaque_argv0"
	KeyAutoApproveHLSegmentParseOrExpansion = "auto_approve_hl_segment_parse_or_expansion"
	KeyAutoApproveHLAwkFromFileOrFlags      = "auto_approve_hl_awk_from_file_or_flags"
	KeyAutoApproveHLAwkSourceError          = "auto_approve_hl_awk_source_error"
	KeyAutoApproveHLAwkReadonlyFailed       = "auto_approve_hl_awk_readonly_failed"
)

var messages = map[string]map[string]string{
	"en": {
		KeyHelpText:                             englishHelpText(),
		KeyUsageRun:                             "Usage: /exec <command> (for example: /exec ls -la)",
		KeyUnknownCmd:                           "Unknown command. Type /help for the full list.",
		KeyDelveLabel:                           "Delve:", // legacy
		KeyErrorPrefix:                          "Error: ",
		KeyConfigPrefix:                         "Config: ",
		KeyWaitOrCancel:                         "(Please wait for the current response, or press Esc to cancel)",
		KeyCommandExecWaitOrCancel:              "(Command running — press Esc to cancel)",
		KeyExecStreamPreviewHeader:              "Output (last lines):",
		KeyExecStreamTranscriptTruncatedHint:    "%d earlier output line(s) omitted in the transcript; full output is stored in session history.",
		KeyTranscriptReplayTruncatedNotice:      "------ Replay truncated to the latest %d lines. Use /history to view older content. ------",
		KeyPlaceholderInput:                     "Type your question or / for slash commands.",
		KeyInputHistBrowsingHint:                "↑/↓ input history · Enter to send · any other key edits",
		KeyInputHintApproveThree:                "1, 2 or 3",
		KeyInputHintSensitive:                   "1, 2 or 3",
		KeyChoiceApprove:                        "Approve",
		KeyChoiceRefuse:                         "Refuse (do not run)",
		KeyChoiceRunStore:                       "Run, return to AI, store in history",
		KeyChoiceRunNoStore:                     "Run, return to AI, do not store",
		KeyChoiceCopy:                           "Copy",
		KeyChoiceDismiss:                        "Dismiss",
		KeyApprovalPrompt:                       "Command to run (approval required):",
		KeyApprovalSummary:                      "Summary:",
		KeyApprovalAutoApprovePolicy:            "Risk Hint:",
		KeyApprovalWhy:                          "Purpose:",
		KeyApprovalDecisionApproved:             "Decision: approved",
		KeyApprovalDecisionRejected:             "Decision: rejected",
		KeyRiskReadOnly:                         "READ-ONLY",
		KeyRiskLow:                              "LOW-RISK",
		KeyRiskHigh:                             "HIGH-RISK",
		KeySensitivePrompt:                      "This command may access sensitive files. Choose:",
		KeySensitiveChoice1:                     "1 = Refuse (do not run)",
		KeySensitiveChoice2:                     "2 = Run, return result to AI, store in history",
		KeySensitiveChoice3:                     "3 = Run, return result to AI, do not store in history",
		KeyUserLabel:                            "User: ",
		KeyAILabel:                              "AI: ",
		KeyTranscriptUserPrompt:                 "> ",
		KeyInfoLabel:                            "Info: ",
		KeyAgentReplyEmpty:                      "The model returned an empty reply (no assistant text in the API response).",
		KeyRunLabel:                             "Run: ",
		KeyRunLineAutoAllowed:                   "Run (checks passed): ",
		KeyRunLineApproved:                      "Run (approved): ",
		KeyRunLineDirect:                        "Run (direct): ",
		KeyRunLineOfflineManual:                 "Run (manual): ",
		KeyRunLineSuggested:                     "Run (suggested): ",
		KeySkillLine:                            "Skill: %s",
		KeyDescExit:                             "Quit delve-shell",
		KeyDescRun:                              "Execute a command directly (no AI)",
		KeyDescSh:                               "Spawn bash",
		KeyDescConfig:                           "Manage models, hosts and skills",
		KeyDescHelp:                             "Show help",
		KeyDescConfigRemoveRemote:               "Remove a remote host",
		KeyRunTagSuggested:                      "suggested",
		KeySuggestedCopyHint:                    "Select the command above to copy, or use /exec <cmd> to run it.",
		KeySuggestedCopied:                      "Copied to clipboard.",
		KeyConfigRemoteAdded:                    "Remote added: %s.",
		KeyConfigRemoteRemoved:                  "Remote removed: %s.",
		KeyStatusIdle:                           "[IDLE]",
		KeyStatusExecuting:                      "[EXECUTING]",
		KeyStatusRunning:                        "[PROCESSING]",
		KeyStatusPendingApproval:                "[NEED APPROVAL]",
		KeyStatusWaitingUserInput:               "[WAIT INPUT]",
		KeyStatusSuggest:                        "[SUGGEST]",
		KeySessionNew:                           "New session",
		KeySessionSwitchedTo:                    "Switched to session: %s",
		KeyHistorySwitchedTo:                    "Switched. Active history: %s",
		KeyHistoryPreviewTitle:                  "History · %s",
		KeyHistoryPreviewEmpty:                  "(No messages in this history yet.)",
		KeyHistoryPreviewFooter:                 "Enter to switch · PgUp/PgDn to scroll · Esc to cancel",
		KeyDescSessions:                         "Browse and switch sessions",
		KeyDescNewSession:                       "Start a new session",
		KeyHistorySessionCurrentSuffix:          " [Current]",
		KeySessionNone:                          "No previous history.",
		KeyDescAccess:                           "Switch execution target",
		KeyDescAccessNew:                        "Add a remote host",
		KeyDelRemoteNoHosts:                     "No hosts.",
		KeyDescRemoteOn:                         "Connect to a host",
		KeyDescRemoteOff:                        "Switch to local execution",
		KeyDescAccessOffline:                    "Work offline and paste results",
		KeyOfflinePasteTitle:                    "Offline — paste output in the box below",
		KeyOfflinePasteIntro:                    "This command is not run here. Paste the results back after you run it elsewhere.",
		KeyOfflinePasteReview:                   "Review the command before running it elsewhere.",
		KeyOfflinePasteHint:                     "Enter: submit · Esc: cancel",
		KeyOfflinePasteCopyFailed:               "Could not copy to clipboard. Select the command line above or copy manually.",
		KeyOfflinePastePlaceholder:              "Paste output",
		KeyOfflineExecBashDisabled:              "/bash is not available in Offline mode.",
		KeyBashReturnNotice:                     "Returned from embedded shell (/bash).",
		KeyOfflineSlashExecDisabled:             "/exec is not available in Offline mode.",
		KeyOfflineSlashSkillDisabled:            "/skill is not available in Offline mode.",
		KeySkillScriptsSyncRemote:               "Syncing skill scripts to remote host…",
		KeyHelpTitle:                            "Help",
		KeyHelpOverlayFooter:                    "Esc to close · PgUp/PgDn to scroll",
		KeyAddRemoteTitle:                       "Add Remote",
		KeyAddRemoteScreenTitle:                 "Add remote",
		KeyAddRemoteConnecting:                  "Connecting...",
		KeyAddRemoteOverwriteHint:               "A saved remote with this target already exists. Choose an action:",
		KeyAddRemoteOverwriteChoice:             "1. Overwrite saved remote",
		KeyAddRemoteKeepEditingChoice:           "2. Keep editing",
		KeyAddRemoteHostLabel:                   "Host (address or host:port):",
		KeyAddRemoteUserLabel:                   "Username:",
		KeyAddRemoteKeyLabel:                    "Key path (optional):",
		KeyAddRemoteSaveLabel:                   "Save as remote (Space to toggle)",
		KeyAddRemoteNameLabel:                   "Name (optional):",
		KeyRemoteTitleBarRemote:                 "Remote",
		KeyRemoteTitleBarOffline:                "Offline",
		KeyRemoteAuthTitle:                      "Remote Auth",
		KeyAddRemoteHostPlaceholder:             "host or host:22",
		KeyAddRemoteUserPlaceholder:             "e.g. root",
		KeyAddRemoteKeyPlaceholder:              "~/.ssh/id_rsa (optional)",
		KeyAddRemoteNamePlaceholder:             "name (optional)",
		KeyRemoteAuthPasswordPlaceholder:        "SSH password",
		KeyRemoteAuthIdentityPlaceholder:        "~/.ssh/id_rsa",
		KeyRemoteAuthUsernameTitle:              "SSH auth for %s",
		KeyRemoteAuthMethodTitle:                "Choose authentication method:",
		KeyRemoteAuthPasswordChoice:             "1. Password",
		KeyRemoteAuthIdentityChoice:             "2. Key file (identity file)",
		KeyRemoteAuthPasswordTitle:              "SSH password for %s",
		KeyRemoteAuthIdentityTitle:              "SSH key file path for %s",
		KeyRemoteAuthAutoIdentityTitle:          "SSH auth for %s",
		KeyRemoteAuthHostKeyTitle:               "Host key verification",
		KeyRemoteAuthConnecting:                 "Connecting...",
		KeyRemoteAuthConfiguredKey:              "Connecting with configured SSH key...",
		KeyRemoteAuthKnownHostsUpdate:           "Updating known_hosts and reconnecting...",
		KeyRemoteAuthAcceptKnownHosts:           "1. Accept and update known_hosts",
		KeyRemoteAuthRejectKnownHosts:           "2. Reject and abort",
		KeyRemoteAuthTargetLabel:                "Target: %s",
		KeyRemoteAuthFingerprintLabel:           "Fingerprint: %s",
		KeyRemoteAuthHostKeyUnknown:             "Host key is not trusted yet. Accept to add/update known_hosts or reject to abort.",
		KeyRemoteAuthHostKeyMismatch:            "Host key mismatch detected. Accept to update known_hosts or reject to abort.",
		KeyTitleBarLocal:                        "Local",
		KeyConfigSavedModel:                     "Config saved.",
		KeyConfigModelTitle:                     "Config Model",
		KeyConfigModelBaseURLLabel:              "Base URL (optional; empty = OpenAI official API):",
		KeyConfigModelApiKeyLabel:               "API key (optional; supports $VAR):",
		KeyConfigModelBaseURLPlaceholder:        "https://api.openai.com/v1 (optional)",
		KeyConfigModelApiKeyPlaceholder:         "sk-... or $API_KEY",
		KeyConfigModelModelLabel:                "Model:",
		KeyConfigModelMaxMessagesLabel:          "Max context messages (default 50):",
		KeyConfigModelMaxCharsLabel:             "Max context chars (default: no limit or auto from API):",
		KeyConfigModelModelRequired:             "Model is required.",
		KeyConfigModelChecking:                  "Checking...",
		KeyConfigModelCheckOK:                   "Model check OK.",
		KeyConfigModelCheckFailed:               "Model check failed: %v",
		KeyConfigModelBaseURLAutoCorrected:      "Base URL updated to %s (added /v1).",
		KeyDescConfigModel:                      "Configure model settings",
		KeyDescSkill:                            "Use a skill",
		KeyUsageSkill:                           "Usage: /skill {name} [text] (text after the name is optional)",
		KeySkillNotFound:                        "Skill not found.",
		KeySkillNone:                            "No skills (add dirs with SKILL.md under ~/.delve-shell/skills/)",
		KeyDescSkillInstall:                     "Install a skill from a git repo",
		KeyDescSkillRemove:                      "Remove an installed skill",
		KeyDescConfigUpdateSkill:                "Update an installed skill",
		KeyAddSkillTitle:                        "Add skill",
		KeyAddSkillURLLabel:                     "Git URL:",
		KeyAddSkillRefLabel:                     "Ref — branch or tag:",
		KeyAddSkillPathLabel:                    "Path in repo — e.g. skills/foo:",
		KeyAddSkillNameLabel:                    "Local skill name:",
		KeyAddSkillURLRequired:                  "URL is required.",
		KeyAddSkillURLPlaceholder:               "https://github.com/owner/repo or owner/repo",
		KeyAddSkillRefPlaceholder:               "main",
		KeyAddSkillPathPlaceholder:              "skills/foo",
		KeyAddSkillNamePlaceholder:              "local skill name",
		KeyUpdateSkillTitle:                     "Update skill",
		KeyUpdateSkillSkillLabel:                "Skill: %s",
		KeyUpdateSkillURLLabel:                  "URL:   %s",
		KeyUpdateSkillPathLabel:                 "Path:  %s",
		KeyUpdateSkillCurrentCommitLabel:        "Current commit: %s",
		KeyUpdateSkillLatestCommitLabel:         "Latest commit:  %s",
		KeySkillInstalled:                       "Skill installed: %s",
		KeySkillRemoved:                         "Skill removed: %s",
		KeySkillInstallFailed:                   "Skill install failed: %v",
		KeySkillRemoveFailed:                    "Skill remove failed: %v",
		KeyUsageSkillRemove:                     "Usage: /config del-skill <skill_name>",
		KeySkillAlreadyExists:                   "Skill already exists. Remove it first or use another name, or use /config update-skill <name> to update it.",
		KeyOverlayFormFooter:                    "Up/Down to move · Enter to apply · Esc to cancel",
		KeyOverlayPicklistHint:                  "  Up/Down to move · Enter or Tab to apply",
		KeyOverlayUpdateSkillRefTitle:           "Ref · Up/Down to move · Enter to update · Esc to cancel:",
		KeyOverlayEscCancel:                     "Esc to cancel",
		KeyOverlayEnterContinueEsc:              "Enter to continue · Esc to cancel",
		KeyOverlay12SelectEsc:                   "1 or 2 to select · Esc to cancel",
		KeyOverlayChoiceSelectEsc:               "Up/Down to move · Enter to select · 1/2 also work · Esc to cancel",
		KeyOverlayEnterSubmitEsc:                "Enter to submit · Esc to cancel",
		KeyOverlayEnterUpdateEsc:                "Up/Down to move · Enter to update · Esc to cancel",
		KeyAutoApproveHLWriteRedirection:        "Output redirection to a file (> or >>) is not allowed for auto-approve.",
		KeyAutoApproveHLShellParseError:         "Could not parse the command as shell: %v",
		KeyAutoApproveHLUnsupportedConstruct:    "Cannot auto-approve: unsupported syntax, or the program name is not a fixed literal.",
		KeyAutoApproveHLExpansionNotAllowed:     "Cannot auto-approve: shell expansion in these arguments is not allowed for this command.",
		KeyAutoApproveHLEmptySegment:            "Empty command segment.",
		KeyAutoApproveHLAllowlistNotLoaded:      "Allowlist is not available.",
		KeyAutoApproveHLCommandNotInAllowlist:   "Command is not on the allowlist: %s",
		KeyAutoApproveHLArgsPolicyMismatch:      "Arguments do not match the allowlist policy for %s.",
		KeyAutoApproveHLOpaqueArgv0:             "The command name is not a fixed literal.",
		KeyAutoApproveHLSegmentParseOrExpansion: "This part could not be matched to the allowlist (parsing or expansion).",
		KeyAutoApproveHLAwkFromFileOrFlags:      "awk: program from a file (-f) or unsupported options.",
		KeyAutoApproveHLAwkSourceError:          "awk: %v",
		KeyAutoApproveHLAwkReadonlyFailed:       "awk: the script failed the read-only check (for example: system(), writing to a file, or getline from a shell pipe).",
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
