package tools

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"

	"github.com/cloudwego/eino/components/tool"
	"github.com/cloudwego/eino/schema"

	"delve-shell/internal/skillstore"
)

// GetSkillTool returns one skill's full detail and SKILL.md content so the AI can learn how to call run_skill.
type GetSkillTool struct{}

var _ tool.InvokableTool = (*GetSkillTool)(nil)

func (t *GetSkillTool) Info(ctx context.Context) (*schema.ToolInfo, error) {
	return &schema.ToolInfo{
		Name: "get_skill",
		Desc: "Get one skill's full detail: description, script list, and complete SKILL.md content (usage, params, examples). Call this before run_skill so you know which script and args to use. Use list_skills first to see available skill names.",
		ParamsOneOf: schema.NewParamsOneOfByParams(map[string]*schema.ParameterInfo{
			"skill_name": {
				Type:     schema.String,
				Desc:     "Skill name (directory under ~/.delve-shell/skills/).",
				Required: true,
			},
		}),
	}, nil
}

func (t *GetSkillTool) InvokableRun(ctx context.Context, argumentsInJSON string, _ ...tool.Option) (string, error) {
	var input struct {
		SkillName string `json:"skill_name"`
	}
	if err := json.Unmarshal([]byte(argumentsInJSON), &input); err != nil {
		return "get_skill requires skill_name", nil
	}
	skillName := strings.TrimSpace(input.SkillName)
	if skillName == "" {
		return "get_skill requires skill_name. Use list_skills to see available skills.", nil
	}

	skillDir := skillstore.SkillDir(skillName)
	if _, err := os.Stat(filepath.Join(skillDir, "SKILL.md")); err != nil {
		return "Skill not found: " + skillName + ". Use list_skills to see available skills, then get_skill(skill_name) to read its SKILL.md.", nil
	}
	meta, err := skillstore.LoadSKILL(skillDir)
	if err != nil {
		return "Failed to load skill: " + err.Error(), nil
	}
	scriptNames, err := skillstore.ListScripts(skillDir)
	if err != nil {
		scriptNames = nil
	}
	var b strings.Builder
	b.WriteString("Skill: ")
	b.WriteString(meta.Name)
	b.WriteString("\nDescription: ")
	b.WriteString(meta.Description)
	if len(scriptNames) == 0 {
		b.WriteString("\nScripts: (none)")
	} else {
		b.WriteString("\nScripts: ")
		b.WriteString(strings.Join(scriptNames, ", "))
	}
	b.WriteString("\nUsage: run_skill(skill_name=\"" + meta.Name + "\", script_name=\"<script>\", args=[...])")
	content, err := skillstore.ReadSKILLContent(skillDir)
	if err == nil && content != "" {
		b.WriteString("\n\n--- SKILL.md (full) ---\n")
		b.WriteString(content)
	}
	return b.String(), nil
}
