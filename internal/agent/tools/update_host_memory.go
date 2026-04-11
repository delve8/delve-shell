package tools

import (
	"context"
	"encoding/json"

	"github.com/cloudwego/eino/components/tool"
	"github.com/cloudwego/eino/schema"

	"delve-shell/internal/hostmem"
)

// UpdateHostMemoryTool lets the AI store stable host facts for the current execution environment.
type UpdateHostMemoryTool struct {
	CurrentContext func() hostmem.Context
}

var _ tool.InvokableTool = (*UpdateHostMemoryTool)(nil)

func (t *UpdateHostMemoryTool) Info(ctx context.Context) (*schema.ToolInfo, error) {
	return &schema.ToolInfo{
		Name: "update_host_memory",
		Desc: "Update host memory for the current execution environment. Use only for stable, reusable host facts. Memory is helpful but not guaranteed current, so store concise structured facts with evidence rather than long prose.",
		ParamsOneOf: schema.NewParamsOneOfByParams(map[string]*schema.ParameterInfo{
			"role": {
				Type: schema.String,
				Desc: "Stable machine role, for example k8s_control_plane, k8s_worker, bastion, build_agent.",
			},
			"role_confidence": {
				Type: schema.Number,
				Desc: "0 to 1 confidence for the role or semantic conclusion.",
			},
			"os_family": {
				Type: schema.String,
				Desc: "OS family when you have a better normalized value than existing memory.",
			},
			"tags_add": {
				Type: schema.Array,
				ElemInfo: &schema.ParameterInfo{
					Type: schema.String,
					Desc: "One tag to add.",
				},
				Desc: "Stable machine tags to add.",
			},
			"notes_add": {
				Type: schema.Array,
				ElemInfo: &schema.ParameterInfo{
					Type: schema.String,
					Desc: "One note to add.",
				},
				Desc: "Short stable notes to add.",
			},
			"evidence_add": {
				Type: schema.Array,
				ElemInfo: &schema.ParameterInfo{
					Type: schema.String,
					Desc: "One evidence item.",
				},
				Desc: "Short evidence strings behind semantic conclusions.",
			},
			"available_commands_add": {
				Type: schema.Array,
				ElemInfo: &schema.ParameterInfo{
					Type: schema.String,
					Desc: "One available command.",
				},
				Desc: "Commands believed available for the current user profile.",
			},
			"missing_commands_add": {
				Type: schema.Array,
				ElemInfo: &schema.ParameterInfo{
					Type: schema.String,
					Desc: "One missing command.",
				},
				Desc: "Commands believed missing for the current user profile.",
			},
			"package_managers_add": {
				Type: schema.Array,
				ElemInfo: &schema.ParameterInfo{
					Type: schema.String,
					Desc: "One package manager.",
				},
				Desc: "Package managers believed available.",
			},
		}),
	}, nil
}

func (t *UpdateHostMemoryTool) InvokableRun(ctx context.Context, argumentsInJSON string, _ ...tool.Option) (string, error) {
	_ = ctx
	if t.CurrentContext == nil {
		return "host memory is not available", nil
	}
	var patch hostmem.UpdatePatch
	if err := json.Unmarshal([]byte(argumentsInJSON), &patch); err != nil {
		return "update_host_memory requires structured JSON parameters", nil
	}
	return hostmem.Update(t.CurrentContext(), patch)
}
