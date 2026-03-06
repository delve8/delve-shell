package ui

import (
	"strings"

	"delve-shell/internal/i18n"
)

// slashOption is one row in the slash command list (command + description).
type slashOption struct {
	Cmd  string
	Desc string
}

// getSlashOptions returns top-level slash commands (shown when input starts with "/"); order: help, cancel, config, mode, reload, run, sh, exit.
func getSlashOptions(lang string) []slashOption {
	return []slashOption{
		{"/help", i18n.T(lang, i18n.KeyDescHelp)},
		{"/cancel", i18n.T(lang, i18n.KeyDescCancel)},
		{"/config", i18n.T(lang, i18n.KeyDescConfig)},
		{"/mode suggest", i18n.T(lang, i18n.KeyDescModeSuggest)},
		{"/mode run", i18n.T(lang, i18n.KeyDescModeRun)},
		{"/reload", i18n.T(lang, i18n.KeyDescReload)},
		{"/run <cmd>", i18n.T(lang, i18n.KeyDescRun)},
		{"/sh", i18n.T(lang, i18n.KeyDescSh)},
		{"/exit", i18n.T(lang, i18n.KeyDescExit)},
	}
}

// getConfigSubOptions returns /config sub-options (shown when input starts with "/config"), not /exit, /sh, etc.
func getConfigSubOptions(lang string) []slashOption {
	return []slashOption{
		{"/config show", i18n.T(lang, i18n.KeyDescConfigShow)},
		{"/config mode <suggest|run>", i18n.T(lang, i18n.KeyDescConfigMode)},
		{"/config allowlist update", i18n.T(lang, i18n.KeyDescConfigAllowlistUpdate)},
		{"/config llm base_url <url>", i18n.T(lang, i18n.KeyDescConfigLLMBaseURL)},
		{"/config llm api_key <key>", i18n.T(lang, i18n.KeyDescConfigLLMApiKey)},
		{"/config llm model <name>", i18n.T(lang, i18n.KeyDescConfigLLMModel)},
		{"/config language <en|zh>", i18n.T(lang, i18n.KeyDescConfigLanguage)},
	}
}

// getSlashOptionsForInput returns slash options to show: when input is "/config" or "/config xxx" returns only /config sub-options; when "/mode" or "/mode x" returns mode sub-options; else top-level commands.
func getSlashOptionsForInput(inputVal string, lang string) []slashOption {
	normalized := strings.TrimPrefix(inputVal, "/")
	normalized = strings.ToLower(strings.TrimSpace(normalized))
	if normalized == "config" || strings.HasPrefix(normalized, "config ") {
		return getConfigSubOptions(lang)
	}
	if normalized == "mode" || strings.HasPrefix(normalized, "mode ") {
		return getSlashOptions(lang) // /mode suggest, /mode run
	}
	return getSlashOptions(lang)
}

// visibleSlashOptions filters options by input prefix and returns matching indices; if none match, returns all.
func visibleSlashOptions(input string, opts []slashOption) []int {
	input = strings.TrimPrefix(input, "/")
	input = strings.ToLower(input)
	var out []int
	for i, opt := range opts {
		base := strings.Split(opt.Cmd, " ")[0]
		base = strings.TrimPrefix(base, "/")
		if input == "" || strings.HasPrefix(base, input) || strings.HasPrefix(opt.Cmd, "/"+input) {
			out = append(out, i)
		}
	}
	if len(out) == 0 {
		for i := range opts {
			out = append(out, i)
		}
	}
	return out
}

// slashChosenToInputValue converts the chosen slash command to the string to put in the input (strips <placeholder> and adds space).
func slashChosenToInputValue(chosen string) string {
	if strings.Contains(chosen, " <") {
		if i := strings.Index(chosen, " <"); i > 0 {
			return chosen[:i] + " "
		}
	}
	return chosen
}
