package agent

import (
	"context"

	"github.com/cloudwego/eino/schema"
)

// Agent 封装 eino agent，后续接入 ReAct + 执行命令 / 查看上下文 tool
type Agent struct{}

// New 创建 agent（占位，后续注入 model、tools、rules）
func New(ctx context.Context) (*Agent, error) {
	_ = schema.Message{}
	return &Agent{}, nil
}
