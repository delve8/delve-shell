package history

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
)

func TestSession_AppendCommand_SkillAuditPayload(t *testing.T) {
	dir := t.TempDir()
	s := &Session{id: "skill-audit", path: filepath.Join(dir, "skill-audit.jsonl")}
	defer s.Close()
	if err := s.AppendCommand("./run.sh", true, "why", "low", "skill", "my-skill"); err != nil {
		t.Fatal(err)
	}
	data, err := os.ReadFile(s.path)
	if err != nil {
		t.Fatal(err)
	}
	var ev Event
	if err := json.Unmarshal([]byte(firstLine(string(data))), &ev); err != nil {
		t.Fatal(err)
	}
	if ev.Type != "command" {
		t.Errorf("type: %q", ev.Type)
	}
	var payload map[string]interface{}
	if err := json.Unmarshal(ev.Payload, &payload); err != nil {
		t.Fatal(err)
	}
	if payload["kind"] != "skill" {
		t.Errorf("kind: %v", payload["kind"])
	}
	if payload["skill_name"] != "my-skill" {
		t.Errorf("skill_name: %v", payload["skill_name"])
	}
}

func TestSession_AppendCommandResult_RedactsBeforeWrite(t *testing.T) {
	dir := t.TempDir()
	// create session that writes under temp dir
	s := &Session{
		id:   "test",
		path: filepath.Join(dir, "test.jsonl"),
	}
	defer s.Close()

	stdout := "password=abc123\nsome output"
	stderr := "JWT_SECRET=xyz\nerror details"

	if err := s.AppendCommandResult("echo test", stdout, stderr, 0); err != nil {
		t.Fatalf("AppendCommandResult error: %v", err)
	}

	data, err := os.ReadFile(s.path)
	if err != nil {
		t.Fatalf("read session file: %v", err)
	}
	lines := string(data)
	if lines == "" {
		t.Fatal("expected non-empty session file")
	}

	var ev Event
	if err := json.Unmarshal([]byte(firstLine(lines)), &ev); err != nil {
		t.Fatalf("unmarshal event: %v", err)
	}
	var payload map[string]interface{}
	if err := json.Unmarshal(ev.Payload, &payload); err != nil {
		t.Fatalf("unmarshal payload: %v", err)
	}

	gotStdout, _ := payload["stdout"].(string)
	gotStderr, _ := payload["stderr"].(string)

	if gotStdout == stdout {
		t.Errorf("stdout not redacted, got %q", gotStdout)
	}
	if gotStderr == stderr {
		t.Errorf("stderr not redacted, got %q", gotStderr)
	}
	if contains(gotStdout, "abc123") || contains(gotStdout, "password=abc123") {
		t.Errorf("stdout still contains secret, got %q", gotStdout)
	}
	if contains(gotStderr, "xyz") || contains(gotStderr, "JWT_SECRET=xyz") {
		t.Errorf("stderr still contains secret, got %q", gotStderr)
	}
}

func firstLine(s string) string {
	for i := 0; i < len(s); i++ {
		if s[i] == '\n' {
			return s[:i]
		}
	}
	return s
}

