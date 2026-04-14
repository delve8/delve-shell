package history

import (
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"delve-shell/internal/config"
)

// Event is one history event for audit and LLM context.
type Event struct {
	Time    time.Time       `json:"time"`
	Type    string          `json:"type"` // EventType* constants
	Payload json.RawMessage `json:"payload"`
}

// Session is one session's history; only delve-shell writes; AI reads via read-only API.
type Session struct {
	id   string
	path string
	mu   sync.Mutex
	f    *os.File
	// afterAppend is invoked after a history event is successfully written.
	afterAppend func(Event)
}

type ExecutionContext struct {
	Execution   string
	Target      string
	OfflineMode bool
	AutoAllowed bool
}

const (
	ExecutionLocal         = "local"
	ExecutionRemote        = "remote"
	ExecutionOfflineManual = "offline_manual"
)

// NewSession creates a new session with a generated id (YYMMDD-HHMMSS + random hex suffix);
// file is created on first write to avoid empty files.
func NewSession() (*Session, error) {
	id := newSessionID()
	dir := config.HistoryDir()
	if err := os.MkdirAll(dir, 0700); err != nil {
		return nil, err
	}
	path := filepath.Join(dir, id+".jsonl")
	return &Session{id: id, path: path, f: nil}, nil
}

func newSessionID() string {
	return time.Now().Format("060102-150405") + "-" + randomHex2()
}

func randomHex2() string {
	b := make([]byte, 1)
	if _, err := rand.Read(b); err != nil {
		return hex.EncodeToString([]byte{byte(time.Now().UnixNano() % 256)})
	}
	return hex.EncodeToString(b)
}

// OpenSession opens an existing session file for appending (e.g. to continue a previous session).
// path must be an absolute path to a .jsonl file under HistoryDir; the session id is derived from the filename.
func OpenSession(path string) (*Session, error) {
	dir := config.HistoryDir()
	abs, err := filepath.Abs(path)
	if err != nil {
		return nil, err
	}
	// Ensure path is under HistoryDir
	rel, err := filepath.Rel(dir, abs)
	if err != nil || strings.HasPrefix(rel, "..") {
		return nil, errors.New("session path must be under history dir")
	}
	if !strings.HasSuffix(rel, ".jsonl") {
		return nil, errors.New("session path must be a .jsonl file")
	}
	id := strings.TrimSuffix(filepath.Base(abs), ".jsonl")
	return &Session{id: id, path: abs, f: nil}, nil
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
	if s.f == nil {
		f, err := os.OpenFile(s.path, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0600)
		if err != nil {
			s.mu.Unlock()
			return err
		}
		s.f = f
	}
	_, err = s.f.Write(append(line, '\n'))
	afterAppend := s.afterAppend
	s.mu.Unlock()
	if err == nil && afterAppend != nil {
		afterAppend(ev)
	}
	return err
}

// AppendUserInput records user input.
func (s *Session) AppendUserInput(text string) error {
	return s.append(EventTypeUserInput, map[string]string{"text": text})
}

// AppendLLMResponse records LLM response (caller passes serialized or structured content).
func (s *Session) AppendLLMResponse(payload interface{}) error {
	return s.append(EventTypeLLMResponse, payload)
}

// AppendCommand records a command about to run; reason and riskLevel are optional, for audit.
// kind is empty for shell (execute_command); use [CommandPayloadKindSkill] for run_skill. skillName is set when kind is skill.
func (s *Session) AppendCommand(command string, approved bool, reason, riskLevel, kind, skillName string) error {
	return s.AppendCommandWithContext(command, approved, reason, riskLevel, kind, skillName, ExecutionContext{})
}

func (s *Session) AppendCommandWithContext(command string, approved bool, reason, riskLevel, kind, skillName string, execCtx ExecutionContext) error {
	payload := map[string]interface{}{"command": command, "approved": approved}
	if reason != "" {
		payload["reason"] = reason
	}
	if riskLevel != "" {
		payload["risk_level"] = riskLevel
	}
	if kind != "" {
		payload["kind"] = kind
	}
	if skillName != "" {
		payload["skill_name"] = skillName
	}
	appendExecutionContext(payload, execCtx)
	return s.append(EventTypeCommand, payload)
}

// AppendSuggestedCommand records a command that was only suggested (not executed), e.g. in suggest mode.
func (s *Session) AppendSuggestedCommand(command, reason, riskLevel, kind, skillName string) error {
	return s.AppendSuggestedCommandWithContext(command, reason, riskLevel, kind, skillName, ExecutionContext{})
}

func (s *Session) AppendSuggestedCommandWithContext(command, reason, riskLevel, kind, skillName string, execCtx ExecutionContext) error {
	payload := map[string]interface{}{"command": command, "approved": false, "suggested": true}
	if reason != "" {
		payload["reason"] = reason
	}
	if riskLevel != "" {
		payload["risk_level"] = riskLevel
	}
	if kind != "" {
		payload["kind"] = kind
	}
	if skillName != "" {
		payload["skill_name"] = skillName
	}
	appendExecutionContext(payload, execCtx)
	return s.append(EventTypeCommand, payload)
}

// AppendCommandResult records command execution result.
func (s *Session) AppendCommandResult(command string, stdout, stderr string, exitCode int) error {
	return s.AppendCommandResultWithContext(command, stdout, stderr, exitCode, ExecutionContext{})
}

func (s *Session) AppendCommandResultWithContext(command string, stdout, stderr string, exitCode int, execCtx ExecutionContext) error {
	redactedStdout := RedactAndTruncateToolOutput(stdout)
	redactedStderr := RedactAndTruncateToolOutput(stderr)
	payload := map[string]interface{}{
		"command":   command,
		"stdout":    redactedStdout,
		"stderr":    redactedStderr,
		"exit_code": exitCode,
	}
	appendExecutionContext(payload, execCtx)
	return s.append(EventTypeCommandResult, payload)
}

const manualPasteNoteEN = "Pasted by user; may be edited or mistaken."

// AppendOfflineCommandProposal records a command proposed in offline mode (not executed in this tool).
func (s *Session) AppendOfflineCommandProposal(command, reason, riskLevel string) error {
	return s.AppendOfflineCommandProposalWithContext(command, reason, riskLevel, ExecutionContext{Execution: ExecutionOfflineManual, OfflineMode: true})
}

func (s *Session) AppendOfflineCommandProposalWithContext(command, reason, riskLevel string, execCtx ExecutionContext) error {
	payload := map[string]interface{}{
		"command":  command,
		"approved": true,
	}
	if reason != "" {
		payload["reason"] = reason
	}
	if riskLevel != "" {
		payload["risk_level"] = riskLevel
	}
	if strings.TrimSpace(execCtx.Execution) == "" {
		execCtx.Execution = ExecutionOfflineManual
	}
	execCtx.OfflineMode = true
	appendExecutionContext(payload, execCtx)
	return s.append(EventTypeCommand, payload)
}

// AppendOfflinePasteResult records user-pasted output for an offline command (no exit_code; not machine-verified).
func (s *Session) AppendOfflinePasteResult(command, pasted string) error {
	return s.AppendOfflinePasteResultWithContext(command, pasted, ExecutionContext{Execution: ExecutionOfflineManual, OfflineMode: true})
}

func (s *Session) AppendOfflinePasteResultWithContext(command, pasted string, execCtx ExecutionContext) error {
	payload := map[string]interface{}{
		"command":      command,
		"stdout":       RedactAndTruncateToolOutput(pasted),
		"manual_paste": true,
		"note":         manualPasteNoteEN,
	}
	if strings.TrimSpace(execCtx.Execution) == "" {
		execCtx.Execution = ExecutionOfflineManual
	}
	execCtx.OfflineMode = true
	appendExecutionContext(payload, execCtx)
	return s.append(EventTypeCommandResult, payload)
}

func appendExecutionContext(payload map[string]interface{}, execCtx ExecutionContext) {
	if payload == nil {
		return
	}
	if v := strings.TrimSpace(execCtx.Execution); v != "" {
		payload["execution"] = v
	}
	if v := strings.TrimSpace(execCtx.Target); v != "" {
		payload["execution_target"] = v
	}
	if execCtx.OfflineMode {
		payload["offline_mode"] = true
	}
	if execCtx.AutoAllowed {
		payload["auto_allowed"] = true
	}
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

// SetAfterAppendHook replaces the callback invoked after a history event is successfully written.
func (s *Session) SetAfterAppendHook(fn func(Event)) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.afterAppend = fn
}
