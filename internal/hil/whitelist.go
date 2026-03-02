package hil

import (
	"regexp"
	"strings"

	"delve-shell/internal/config"
)

// Whitelist 基于配置的白名单匹配器
type Whitelist struct {
	patterns []compiledEntry
}

type compiledEntry struct {
	literal string
	regex   *regexp.Regexp
}

// NewWhitelist 从白名单条目构建匹配器；无效正则会被忽略
func NewWhitelist(entries []config.WhitelistEntry) *Whitelist {
	w := &Whitelist{}
	for _, e := range entries {
		if e.IsRegex {
			if re, err := regexp.Compile(e.Pattern); err == nil {
				w.patterns = append(w.patterns, compiledEntry{regex: re})
			}
		} else {
			w.patterns = append(w.patterns, compiledEntry{literal: e.Pattern})
		}
	}
	return w
}

// Allow 判断整条命令（或脚本）是否命中白名单，命中则无需用户审批
func (w *Whitelist) Allow(command string) bool {
	for _, p := range w.patterns {
		if p.regex != nil {
			if p.regex.MatchString(command) {
				return true
			}
		} else if p.literal == command {
			return true
		}
	}
	return false
}

// splitPipeline 按管道符 | 拆分命令，忽略引号内的 |
func splitPipeline(command string) []string {
	var parts []string
	var b strings.Builder
	inSingle := false
	inDouble := false
	for i := 0; i < len(command); i++ {
		c := command[i]
		switch {
		case c == '\'' && !inDouble:
			inSingle = !inSingle
			b.WriteByte(c)
		case c == '"' && !inSingle:
			inDouble = !inDouble
			b.WriteByte(c)
		case c == '|' && !inSingle && !inDouble:
			parts = append(parts, strings.TrimSpace(b.String()))
			b.Reset()
		default:
			b.WriteByte(c)
		}
	}
	if b.Len() > 0 {
		parts = append(parts, strings.TrimSpace(b.String()))
	}
	return parts
}

// splitShellChain 将一段按 ; && || 拆成多条子命令（用于严格校验，防止 cat x; rm -rf / 因含 cat 被放行）
func splitShellChain(segment string) []string {
	segment = strings.TrimSpace(segment)
	if segment == "" {
		return nil
	}
	// 简单按 ; 与 && || 拆分（不处理引号内），每段 trim
	var out []string
	for _, s := range strings.FieldsFunc(segment, func(r rune) bool {
		return r == ';' || r == '&' || r == '|'
	}) {
		s = strings.TrimSpace(s)
		if s != "" {
			out = append(out, s)
		}
	}
	if len(out) == 0 {
		return []string{segment}
	}
	return out
}

// AllowPipeline 若命令含管道，拆成子命令；仅当每个子命令都命中白名单时返回 true，整条管道可自动获批。
// 每一段还会按 ; && || 再拆，避免 "cat x; rm -rf /" 因含 cat 被整段放行。
func (w *Whitelist) AllowPipeline(command string) bool {
	parts := splitPipeline(command)
	if len(parts) <= 1 {
		return false
	}
	for _, part := range parts {
		for _, sub := range splitShellChain(part) {
			if sub == "" || !w.Allow(sub) {
				return false
			}
		}
	}
	return true
}
