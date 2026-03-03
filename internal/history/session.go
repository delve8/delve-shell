package history

import (
	"encoding/json"
	"os"
	"path/filepath"
	"sync"
	"time"

	"delve-shell/internal/config"
)

// Event is one history event for audit and LLM context.
type Event struct {
	Time    time.Time       `json:"time"`
	Type    string          `json:"type"` // "user_input" | "llm_response" | "tool_call" | "command" | "command_result"
	Payload json.RawMessage `json:"payload"`
}

// Session is one session's history; only delve-shell writes; AI reads via read-only API.
type Session struct {
	id   string
	path string
	mu   sync.Mutex
	f    *os.File
}

// NewSession creates a new session; file is created on first write to avoid empty files.
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

// AppendUserInput records user input.
func (s *Session) AppendUserInput(text string) error {
	return s.append("user_input", map[string]string{"text": text})
}

// AppendLLMResponse records LLM response (caller passes serialized or structured content).
func (s *Session) AppendLLMResponse(payload interface{}) error {
	return s.append("llm_response", payload)
}

// AppendCommand records a command about to run; reason and riskLevel are optional, for audit.
func (s *Session) AppendCommand(command string, approved bool, reason, riskLevel string) error {
	payload := map[string]interface{}{"command": command, "approved": approved}
	if reason != "" {
		payload["reason"] = reason
	}
	if riskLevel != "" {
		payload["risk_level"] = riskLevel
	}
	return s.append("command", payload)
}

// AppendCommandResult records command execution result.
func (s *Session) AppendCommandResult(command string, stdout, stderr string, exitCode int) error {
	return s.append("command_result", map[string]interface{}{
		"command":  command,
		"stdout":   stdout,
		"stderr":   stderr,
		"exit_code": exitCode,
	})
}

// Close closes the session file; no-op if never written.
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

// Path returns the session file path (read-only use, e.g. view_context tool).
func (s *Session) Path() string { return s.path }
