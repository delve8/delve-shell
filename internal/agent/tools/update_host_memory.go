package tools

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/cloudwego/eino/components/tool"
	"github.com/cloudwego/eino/schema"

	"delve-shell/internal/hostmem"
)

// UpdateHostMemoryTool lets the AI store stable host facts for the current execution environment.
type UpdateHostMemoryTool struct {
	CurrentContext func() hostmem.Context
	UIEvents       chan<- any
}

var _ tool.InvokableTool = (*UpdateHostMemoryTool)(nil)

func (t *UpdateHostMemoryTool) Info(ctx context.Context) (*schema.ToolInfo, error) {
	return &schema.ToolInfo{
		Name: "update_host_memory",
		Desc: "Update host memory for the current execution environment. Use only for stable, reusable host facts. Memory is helpful but not guaranteed current, so store concise structured facts with evidence rather than long prose.",
		ParamsOneOf: schema.NewParamsOneOfByParams(map[string]*schema.ParameterInfo{
			"role": {
				Type: schema.String,
				Desc: "Stable machine role, for example k8s_control_plane, k8s_worker, bastion, build_agent, database_server, storage_node, monitoring_server.",
			},
			"role_confidence": {
				Type: schema.Number,
				Desc: "0 to 1 confidence for the role or semantic conclusion.",
			},
			"os_family": {
				Type: schema.String,
				Desc: "OS family when you have a better normalized value than existing memory.",
			},
			"capabilities_add": {
				Type: schema.Array,
				ElemInfo: &schema.ParameterInfo{
					Type: schema.String,
					Desc: "One durable capability or installed functional surface.",
				},
				Desc: "Stable capabilities this machine appears to provide or support, for example runs container workloads, manages cluster control plane, builds artifacts, hosts database clients, exposes storage tools.",
			},
			"responsibilities_add": {
				Type: schema.Array,
				ElemInfo: &schema.ParameterInfo{
					Type: schema.String,
					Desc: "One durable responsibility.",
				},
				Desc: "Stable responsibilities or purpose of this machine, for example cluster administration, workload execution, CI builds, monitoring, jump host access, database operations.",
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
	out, err := hostmem.Update(t.CurrentContext(), patch)
	if err != nil {
		return out, err
	}
	sendAgentNotify(t.UIEvents, summarizeHostMemoryPatch(patch))
	return out, nil
}

func summarizeHostMemoryPatch(patch hostmem.UpdatePatch) string {
	var parts []string
	if v := strings.TrimSpace(patch.OSFamily); v != "" {
		parts = append(parts, "os="+v)
	}
	if v := strings.TrimSpace(patch.Role); v != "" {
		part := "role=" + v
		if patch.RoleConfidence > 0 {
			confidence := patch.RoleConfidence
			if confidence > 1 {
				confidence = 1
			}
			part += fmt.Sprintf(" (confidence %.2f)", confidence)
		}
		parts = append(parts, part)
	}
	if v := compactList(patch.CapabilitiesAdd, 6); v != "" {
		parts = append(parts, "capabilities += "+v)
	}
	if v := compactList(patch.ResponsibilitiesAdd, 6); v != "" {
		parts = append(parts, "responsibilities += "+v)
	}
	if v := compactList(patch.TagsAdd, 6); v != "" {
		parts = append(parts, "tags += "+v)
	}
	if v := compactList(patch.AvailableAdd, 8); v != "" {
		parts = append(parts, "available commands += "+v)
	}
	if v := compactList(patch.MissingAdd, 8); v != "" {
		parts = append(parts, "missing commands += "+v)
	}
	if v := compactList(patch.PackageManagersAdd, 4); v != "" {
		parts = append(parts, "package managers += "+v)
	}
	if n := countNonEmpty(patch.NotesAdd); n > 0 {
		parts = append(parts, fmt.Sprintf("notes += %d", n))
	}
	if n := countNonEmpty(patch.EvidenceAdd); n > 0 {
		parts = append(parts, fmt.Sprintf("evidence += %d", n))
	}
	if len(parts) == 0 {
		return ""
	}
	return "Host memory updated: " + strings.Join(parts, "; ")
}

func compactList(items []string, limit int) string {
	var out []string
	seen := make(map[string]struct{})
	for _, item := range items {
		item = strings.TrimSpace(item)
		if item == "" {
			continue
		}
		if _, ok := seen[item]; ok {
			continue
		}
		seen[item] = struct{}{}
		out = append(out, item)
		if len(out) == limit {
			break
		}
	}
	if len(out) == 0 {
		return ""
	}
	if extra := countNonEmpty(items) - len(out); extra > 0 {
		out = append(out, fmt.Sprintf("+%d more", extra))
	}
	return strings.Join(out, ", ")
}

func countNonEmpty(items []string) int {
	n := 0
	for _, item := range items {
		if strings.TrimSpace(item) != "" {
			n++
		}
	}
	return n
}
