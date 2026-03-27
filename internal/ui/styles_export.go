package ui

import (
	"strings"

	"delve-shell/internal/i18n"
)

// RenderOverlayHintLine renders one dim italic hint line from a full-line i18n key (word order stays per locale).
func RenderOverlayHintLine(lang, key string) string {
	s := strings.TrimSpace(i18n.T(lang, key))
	if s == "" {
		return ""
	}
	return hintStyle.Render(s) + "\n"
}

// RenderOverlayFormFooterHint renders the standard multi-field form footer (same copy for add-remote, config LLM, add-skill).
func RenderOverlayFormFooterHint(lang string) string {
	return RenderOverlayHintLine(lang, i18n.KeyOverlayFormFooter)
}

// RenderOverlayPicklistHintLine is the line above inline pick lists (ref/path/path completion).
func RenderOverlayPicklistHintLine(lang string) string {
	return RenderOverlayHintLine(lang, i18n.KeyOverlayPicklistHint)
}

// RenderOverlayUpdateSkillRefTitleLine is the title line above the ref list in update-skill.
func RenderOverlayUpdateSkillRefTitleLine(lang string) string {
	return hintStyle.Render(i18n.T(lang, i18n.KeyOverlayUpdateSkillRefTitle)) + "\n"
}

// SuggestStyleRender renders text using suggestion style.
func SuggestStyleRender(s string) string {
	return suggestStyle.Render(s)
}

// SuggestHiRender renders text using highlighted suggestion style.
func SuggestHiRender(s string) string {
	return suggestHi.Render(s)
}

// ErrStyleRender renders text using error style.
func ErrStyleRender(s string) string {
	return errStyle.Render(s)
}
