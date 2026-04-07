package ui

import (
	"fmt"
	"strings"
	"testing"
)

func TestHandleExecStreamFlushMsg_TruncatesLongBuffer(t *testing.T) {
	m := NewModel(nil, nil)
	m.layout.Width = 120
	m.Interaction.execStreamWindowOpen = true
	for i := 0; i < 25; i++ {
		m.Interaction.execStreamBuffer = append(m.Interaction.execStreamBuffer, execStreamSeg{text: fmt.Sprintf("L%d", i)})
	}
	n, _ := m.Update(ExecStreamFlushMsg{})
	m = n.(*Model)
	transcript := strings.Join(m.messages, "\n")
	if strings.Contains(transcript, "L0") || strings.Contains(transcript, "L21") {
		t.Fatalf("expected early lines omitted, got transcript snippet: %q", transcript[:min(200, len(transcript))])
	}
	if !strings.Contains(transcript, "L24") {
		t.Fatalf("expected tail line preserved: %q", transcript)
	}
	if !strings.Contains(transcript, "22 earlier output line(s) omitted") {
		t.Fatalf("expected truncation hint: %q", transcript)
	}
}

func TestHandleExecStreamFlushMsg_SensitiveAlsoTruncates(t *testing.T) {
	m := NewModel(nil, nil)
	m.layout.Width = 120
	m.Interaction.execStreamWindowOpen = true
	for i := 0; i < 10; i++ {
		m.Interaction.execStreamBuffer = append(m.Interaction.execStreamBuffer, execStreamSeg{text: fmt.Sprintf("S%d", i)})
	}
	n, _ := m.Update(ExecStreamFlushMsg{Sensitive: true})
	m = n.(*Model)
	transcript := strings.Join(m.messages, "\n")
	if strings.Contains(transcript, "S0") || strings.Contains(transcript, "S6") {
		t.Fatalf("sensitive flush should truncate like non-sensitive: %q", transcript)
	}
	if !strings.Contains(transcript, "S9") {
		t.Fatalf("expected tail preserved: %q", transcript)
	}
	if !strings.Contains(transcript, "7 earlier output line(s) omitted") {
		t.Fatalf("expected truncation hint: %q", transcript)
	}
	if !strings.Contains(transcript, "Result contains sensitive data") {
		t.Fatalf("expected sensitive notice: %q", transcript)
	}
}

func TestHandleExecStreamFlushMsg_SensitiveRedactsBody(t *testing.T) {
	m := NewModel(nil, nil)
	m.layout.Width = 120
	m.Interaction.execStreamWindowOpen = true
	m.Interaction.execStreamBuffer = []execStreamSeg{
		{text: "ok"},
		{text: "password=supersecret"},
		{text: "done"},
	}
	n, _ := m.Update(ExecStreamFlushMsg{Sensitive: true})
	m = n.(*Model)
	transcript := strings.Join(m.messages, "\n")
	if strings.Contains(transcript, "supersecret") {
		t.Fatalf("expected password value redacted: %q", transcript)
	}
	if !strings.Contains(transcript, "[REDACTED]") && !strings.Contains(transcript, "password=") {
		t.Fatalf("expected redaction marker or key preserved: %q", transcript)
	}
}

func TestHandleExecStreamFlushMsg_NoTruncateBelowCap(t *testing.T) {
	m := NewModel(nil, nil)
	m.layout.Width = 120
	m.Interaction.execStreamWindowOpen = true
	for i := 0; i < 3; i++ {
		m.Interaction.execStreamBuffer = append(m.Interaction.execStreamBuffer, execStreamSeg{text: fmt.Sprintf("x%d", i)})
	}
	n, _ := m.Update(ExecStreamFlushMsg{})
	m = n.(*Model)
	transcript := strings.Join(m.messages, "\n")
	if strings.Contains(transcript, "omitted") {
		t.Fatalf("unexpected truncation hint for short output: %q", transcript)
	}
	if !strings.Contains(transcript, "x0") || !strings.Contains(transcript, "x2") {
		t.Fatalf("expected full short output: %q", transcript)
	}
}
