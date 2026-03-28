package slashview

import "testing"

func TestShouldFillOnly(t *testing.T) {
	if !ShouldFillOnly(Option{Cmd: "/exec <cmd>"}, "/e") {
		t.Fatalf("expected fill-only for prefix")
	}
	if ShouldFillOnly(Option{Cmd: "/exec <cmd>"}, "/exec <cmd>") {
		t.Fatalf("did not expect fill-only for exact match")
	}
	if !ShouldFillOnly(Option{Cmd: "/skill demo", FillValue: "/skill demo "}, "/skill demo") {
		t.Fatalf("expected fill-only for explicit fill-value option")
	}
}

func TestShouldResolveSelected(t *testing.T) {
	if ShouldResolveSelected(Option{Cmd: "/help"}, "/") {
		t.Fatalf("root slash should not resolve selected command")
	}
	if !ShouldResolveSelected(Option{Cmd: "/help"}, "/he") {
		t.Fatalf("prefix slash should resolve selected command")
	}
}
