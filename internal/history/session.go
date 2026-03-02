package history

import (
	"encoding/json"
	"os"
	"path/filepath"
	"sync"
	"time"

	"delve-shell/internal/config"
)

// Event 单条历史事件，用于审计与 LLM 上下文
type Event struct {
	Time    time.Time       `json:"time"`
	Type    string          `json:"type"` // "user_input" | "llm_response" | "tool_call" | "command" | "command_result"
	Payload json.RawMessage `json:"payload"`
}

// Session 单次会话的历史记录；仅由 delve-shell 写入，AI 通过只读接口读取
type Session struct {
	id   string
	path string
	mu   sync.Mutex
	f    *os.File
}

// NewSession 创建新会话；文件在首次写入时才创建，避免产生空文件
func NewSession(id string) (*Session, error) {
	dir := config.HistoryDir()
	if err := os.MkdirAll(dir, 0700); err != nil {
		return nil, err
	}
	path := filepath.Join(dir, id+".jsonl")
	return &Session{id: id, path: path, f: nil}, nil
}

func (s *Session) append(typ string, payload interface{}) error {
	data, err := json.Marshal(payload)
	if err != nil {
		return err
	}
	ev := Event{Time: time.Now().UTC(), Type: typ, Payload: data}
	line, err := json.Marshal(ev)
	if err != nil {
		return err
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.f == nil {
		f, err := os.OpenFile(s.path, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0600)
		if err != nil {
			return err
		}
		s.f = f
	}
	_, err = s.f.Write(append(line, '\n'))
	return err
}

// AppendUserInput 记录用户输入
func (s *Session) AppendUserInput(text string) error {
	return s.append("user_input", map[string]string{"text": text})
}

// AppendLLMResponse 记录 LLM 返回（思考、结论、工具调用等；调用方传入已序列化或结构化内容）
func (s *Session) AppendLLMResponse(payload interface{}) error {
	return s.append("llm_response", payload)
}

// AppendCommand 记录即将执行的命令/脚本
func (s *Session) AppendCommand(command string, approved bool) error {
	return s.append("command", map[string]interface{}{"command": command, "approved": approved})
}

// AppendCommandResult 记录命令执行结果
func (s *Session) AppendCommandResult(command string, stdout, stderr string, exitCode int) error {
	return s.append("command_result", map[string]interface{}{
		"command":  command,
		"stdout":   stdout,
		"stderr":   stderr,
		"exit_code": exitCode,
	})
}

// Close 关闭会话文件；若从未写入则无需关闭（未创建过文件）
func (s *Session) Close() error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.f == nil {
		return nil
	}
	err := s.f.Close()
	s.f = nil
	return err
}

// Path 返回当前会话文件路径（只读用途，如「查看上下文」tool）
func (s *Session) Path() string { return s.path }
