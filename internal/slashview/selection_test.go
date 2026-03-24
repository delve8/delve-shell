package slashview

import "testing"

func TestShouldFillOnly(t *testing.T) {
	if !ShouldFillOnly("/run <cmd>", "/r") {
		t.Fatalf("expected fill-only for prefix")
	}
	if ShouldFillOnly("/run <cmd>", "/run <cmd>") {
		t.Fatalf("did not expect fill-only for exact match")
	}
}

func TestShouldResolveSelected(t *testing.T) {
	if ShouldResolveSelected("/help", "/") {
		t.Fatalf("root slash should not resolve selected command")
	}
	if !ShouldResolveSelected("/help", "/he") {
		t.Fatalf("prefix slash should resolve selected command")
	}
}
