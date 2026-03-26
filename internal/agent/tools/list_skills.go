package tools

import (
	"context"
	"strings"

	"github.com/cloudwego/eino/components/tool"
	"github.com/cloudwego/eino/schema"

	"delve-shell/internal/skillstore"
)

// ListSkillsTool lists all installed skills (name, description only). Use get_skill to read one skill's full SKILL.md.
type ListSkillsTool struct{}

var _ tool.InvokableTool = (*ListSkillsTool)(nil)

func (t *ListSkillsTool) Info(ctx context.Context) (*schema.ToolInfo, error) {
	return &schema.ToolInfo{
		Name:        "list_skills",
		Desc:        "List all installed skills under ~/.delve-shell/skills/. Returns each skill's name and description. Use get_skill(skill_name) to read one skill's full SKILL.md (usage, params, examples) before calling run_skill.",
		ParamsOneOf: schema.NewParamsOneOfByParams(map[string]*schema.ParameterInfo{}),
	}, nil
}

func (t *ListSkillsTool) InvokableRun(ctx context.Context, argumentsInJSON string, _ ...tool.Option) (string, error) {
	list, err := skillstore.List()
	if err != nil {
		return "list_skills failed: " + err.Error(), nil
	}
	if len(list) == 0 {
		return "No skills installed. Skills live under ~/.delve-shell/skills/ (each subdir with SKILL.md and optional scripts/).", nil
	}
	var b strings.Builder
	b.WriteString("Installed skills:\n")
	for _, s := range list {
		b.WriteString("- ")
		b.WriteString(s.Name)
		b.WriteString(": ")
		b.WriteString(s.Description)
		b.WriteString("\n")
	}
	return b.String(), nil
}
