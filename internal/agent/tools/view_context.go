package tools

import (
	"context"
	"encoding/json"
	"strings"

	"github.com/cloudwego/eino/components/tool"
	"github.com/cloudwego/eino/schema"

	"delve-shell/internal/history"
)

// ViewContextTool lets the AI fetch session context on demand (read-only).
type ViewContextTool struct {
	SessionPath string
	MaxEvents   int
}

var _ tool.InvokableTool = (*ViewContextTool)(nil)

func (t *ViewContextTool) Info(ctx context.Context) (*schema.ToolInfo, error) {
	return &schema.ToolInfo{
		Name: "view_context",
		Desc: "View current session history (user input, LLM replies, executed commands and results). Read-only. Use when you need to recall the conversation or command results.",
		ParamsOneOf: schema.NewParamsOneOfByParams(map[string]*schema.ParameterInfo{
			"max_events": {
				Type: schema.Integer,
				Desc: "Max number of recent events to return; default 50; 0 means use default",
			},
		}),
	}, nil
}

func (t *ViewContextTool) InvokableRun(ctx context.Context, argumentsInJSON string, _ ...tool.Option) (string, error) {
	max := t.MaxEvents
	if max <= 0 {
		max = 50
	}
	if argumentsInJSON != "" {
		var input struct {
			MaxEvents int `json:"max_events"`
		}
		_ = json.Unmarshal([]byte(argumentsInJSON), &input)
		if input.MaxEvents > 0 {
			max = input.MaxEvents
		}
	}
	if t.SessionPath == "" {
		return "no session history", nil
	}
	events, err := history.ReadRecent(t.SessionPath, max)
	if err != nil {
		return "read context failed: " + err.Error(), nil
	}
	var b strings.Builder
	for _, ev := range events {
		b.WriteString(ev.Time.Format("2006-01-02 15:04:05"))
		b.WriteString(" [")
		b.WriteString(ev.Type)
		b.WriteString("] ")
		b.WriteString(string(ev.Payload))
		b.WriteString("\n")
	}
	return b.String(), nil
}
