package slashview

import "testing"

func TestShouldFillOnly(t *testing.T) {
	if !ShouldFillOnly(Option{Cmd: "/exec <cmd>"}, "/e") {
		t.Fatalf("expected fill-only for prefix")
	}
	if ShouldFillOnly(Option{Cmd: "/exec <cmd>"}, "/exec <cmd>") {
		t.Fatalf("did not expect fill-only for exact match")
	}
	if !ShouldFillOnly(Option{Cmd: "/skill demo", FillValue: "/skill demo"}, "/skill demo") {
		t.Fatalf("expected fill-only for explicit fill-value option")
	}
	if !ShouldFillOnly(Option{Cmd: "/access Local"}, "/access l") {
		t.Fatalf("expected fill-only when prefix differs only by case after partial token")
	}
	if ShouldFillOnly(Option{Cmd: "/access Local"}, "/access Local") {
		t.Fatalf("exact command match should not be fill-only")
	}
	if ShouldFillOnly(Option{Cmd: "/access Local"}, "/access local") {
		t.Fatalf("case-insensitive exact match should not be fill-only")
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
