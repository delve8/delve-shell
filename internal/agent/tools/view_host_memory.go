package tools

import (
	"context"

	"github.com/cloudwego/eino/components/tool"
	"github.com/cloudwego/eino/schema"

	"delve-shell/internal/hostmem"
)

// ViewHostMemoryTool returns structured host memory for the current execution environment.
type ViewHostMemoryTool struct {
	CurrentContext func() hostmem.Context
}

var _ tool.InvokableTool = (*ViewHostMemoryTool)(nil)

func (t *ViewHostMemoryTool) Info(ctx context.Context) (*schema.ToolInfo, error) {
	return &schema.ToolInfo{
		Name:        "view_host_memory",
		Desc:        "View stored host memory for the current execution environment. Returns structured JSON-like content with machine facts, semantic role/tags/notes, and command availability for the current user profile. Use this when planning commands or checking prior host understanding.",
		ParamsOneOf: schema.NewParamsOneOfByParams(map[string]*schema.ParameterInfo{}),
	}, nil
}

func (t *ViewHostMemoryTool) InvokableRun(ctx context.Context, argumentsInJSON string, _ ...tool.Option) (string, error) {
	_ = ctx
	_ = argumentsInJSON
	if t.CurrentContext == nil {
		return "host memory is not available", nil
	}
	return hostmem.View(t.CurrentContext())
}
