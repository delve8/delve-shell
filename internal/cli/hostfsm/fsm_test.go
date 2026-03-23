package hostfsm

import "testing"

func TestMachine_LLMTransitions(t *testing.T) {
	m := NewMachine(StateIdle)
	if m.State() != StateIdle {
		t.Fatalf("initial: %s", m.State())
	}
	var ctx Context
	if !m.Apply(&ctx, EvtLLMRunStart) {
		t.Fatal("start from idle")
	}
	if m.State() != StateLLMRunning {
		t.Fatalf("after start: %s", m.State())
	}
	if !m.Apply(&ctx, EvtLLMRunEnd) {
		t.Fatal("end from llm")
	}
	if m.State() != StateIdle {
		t.Fatalf("after end: %s", m.State())
	}
}
