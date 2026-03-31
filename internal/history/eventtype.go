package history

// JSONL event type values (field Event.Type). Stable for audit and read paths.
const (
	EventTypeUserInput     = "user_input"
	EventTypeLLMResponse   = "llm_response"
	EventTypeToolCall      = "tool_call"
	EventTypeCommand       = "command"
	EventTypeCommandResult = "command_result"
)

// CommandPayloadKind is the optional "kind" field on command events (execute_command vs run_skill).
const CommandPayloadKindSkill = "skill"
