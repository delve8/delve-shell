package ui

import (
	"testing"

	"delve-shell/internal/i18n"
	"delve-shell/internal/ui/uivm"
)

func TestDropDuplicateRunTranscript_skipsSecondIdenticalRunLine(t *testing.T) {
	i18n.SetLang("en")
	m := NewModel(nil, nil)
	m.WithTranscriptLines(nil)
	m.layout.Width = 80
	run := "Run (approved): kubectl get pods"
	first := m.renderTranscriptLines([]uivm.Line{{Kind: uivm.LineExec, Text: run}})
	if len(first) != 1 {
		t.Fatalf("want 1 rendered row, got %d", len(first))
	}
	n, _ := m.Update(TranscriptAppendMsg{Lines: []uivm.Line{{Kind: uivm.LineExec, Text: run}}})
	m = n.(*Model)
	if len(m.messages) != 1 {
		t.Fatalf("want 1 message, got %d", len(m.messages))
	}
	dup := m.renderTranscriptLines([]uivm.Line{{Kind: uivm.LineExec, Text: run}})
	n2, _ := m.Update(TranscriptAppendMsg{Lines: []uivm.Line{{Kind: uivm.LineExec, Text: run}}})
	m2 := n2.(*Model)
	if len(m2.messages) != 1 {
		t.Fatalf("duplicate Run line should be dropped, got %d messages", len(m2.messages))
	}
	_ = dup
}

func TestDropDuplicateRunTranscript_stripsDuplicateRunFromMultiLineAppend(t *testing.T) {
	i18n.SetLang("en")
	m := NewModel(nil, nil)
	m.WithTranscriptLines(nil)
	m.layout.Width = 80
	run := "Run (approved): echo hi"
	n, _ := m.Update(TranscriptAppendMsg{Lines: []uivm.Line{{Kind: uivm.LineExec, Text: run}}})
	m = n.(*Model)
	if len(m.messages) != 1 {
		t.Fatalf("want 1 message after first Run, got %d", len(m.messages))
	}
	n2, _ := m.Update(TranscriptAppendMsg{Lines: []uivm.Line{
		{Kind: uivm.LineExec, Text: run},
		{Kind: uivm.LineResult, Text: "out"},
		{Kind: uivm.LineBlank},
	}})
	m2 := n2.(*Model)
	if len(m2.messages) != 3 {
		t.Fatalf("want startup cleared: Run + result + blank = 3 messages, got %d: %#v", len(m2.messages), m2.messages)
	}
}
