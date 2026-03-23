package agent

import (
	"context"

	"github.com/cloudwego/eino/schema"
)

// Agent is a placeholder; the live ReAct runner is Runner in runner.go. LLM tools: internal/agent/tools.
type Agent struct{}

// New creates an agent (placeholder; model, tools, rules to be injected later).
func New(ctx context.Context) (*Agent, error) {
	_ = schema.Message{}
	return &Agent{}, nil
}
