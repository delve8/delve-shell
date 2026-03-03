package agent

import (
	"context"

	"github.com/cloudwego/eino/schema"
)

// Agent wraps the eino agent; will be wired to ReAct + execute_command / view_context tools.
type Agent struct{}

// New creates an agent (placeholder; model, tools, rules to be injected later).
func New(ctx context.Context) (*Agent, error) {
	_ = schema.Message{}
	return &Agent{}, nil
}
