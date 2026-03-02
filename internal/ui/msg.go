package ui

import "delve-shell/internal/agent"

// ApprovalRequestMsg 待用户审批的命令（由 agent 经 channel 转发到 TUI）
type ApprovalRequestMsg = *agent.ApprovalRequest

// AgentReplyMsg agent 对用户消息的回复
type AgentReplyMsg struct {
	Reply string
	Err   error
}

// CommandExecutedMsg 命令执行过程与结果（白名单/已批准/直接执行），用于在对话中展示
type CommandExecutedMsg struct {
	Command    string
	Allowed bool
	Direct     bool   // true 表示 /run 直接执行，不经 AI
	Result     string
	Sensitive  bool   // 为 true 时结果含隐私数据，未写入历史且返回给 LLM 的为 "done"
}

// ConfigReloadedMsg 通知 UI：配置/白名单已重载，下一条消息将使用新配置
type ConfigReloadedMsg struct{}
