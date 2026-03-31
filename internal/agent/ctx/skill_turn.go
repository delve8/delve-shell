package agentctx

import (
	"context"
	"strings"
)

type skillSlashKey struct{}

// WithSkillSlashTurn marks the LLM request context as a /skill <name> turn. When non-empty,
// run_skill may auto-approve if the tool's skill_name matches (case-insensitive).
func WithSkillSlashTurn(ctx context.Context, skillName string) context.Context {
	name := strings.TrimSpace(skillName)
	if name == "" {
		return ctx
	}
	return context.WithValue(ctx, skillSlashKey{}, name)
}

// SkillSlashSkillName returns the skill directory name from a /skill invocation, if any.
func SkillSlashSkillName(ctx context.Context) (name string, ok bool) {
	v, ok := ctx.Value(skillSlashKey{}).(string)
	if !ok {
		return "", false
	}
	v = strings.TrimSpace(v)
	if v == "" {
		return "", false
	}
	return v, true
}
