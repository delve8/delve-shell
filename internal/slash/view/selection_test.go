package slashview

import "testing"

func TestShouldFillOnly(t *testing.T) {
	if !ShouldFillOnly(Option{Cmd: "/skill demo"}, "/sk") {
		t.Fatalf("expected fill-only for prefix")
	}
	if ShouldFillOnly(Option{Cmd: "/skill demo"}, "/skill demo") {
		t.Fatalf("did not expect fill-only for exact match")
	}
	if ShouldFillOnly(Option{Cmd: "/skill demo", FillValue: "/skill demo"}, "/skill demo") {
		t.Fatalf("exact explicit fill-value match should not be fill-only")
	}
	if !ShouldFillOnly(Option{Cmd: "/skill demo", FillValue: "/skill demo"}, "/skill d") {
		t.Fatalf("expected fill-only for explicit fill-value prefix")
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
	if !ShouldFillOnly(Option{Cmd: "/access jump.example.com", FillValue: "/access jump.example.com"}, "/access jump.ex") {
		t.Fatalf("fill-only should consider FillValue prefixes")
	}
	if ShouldFillOnly(Option{Cmd: "/access jump.example.com", FillValue: "/access jump.example.com"}, "/access jump.example.com") {
		t.Fatalf("exact FillValue match should not be fill-only")
	}
	if !ShouldFillOnly(Option{Cmd: "/access prod", Desc: "Production"}, "/access pro") {
		t.Fatalf("fill-only should consider access descriptions")
	}
}

func TestShouldResolveSelected(t *testing.T) {
	if ShouldResolveSelected(Option{Cmd: "/help"}, "/") {
		t.Fatalf("root slash should not resolve selected command")
	}
	if !ShouldResolveSelected(Option{Cmd: "/help"}, "/he") {
		t.Fatalf("prefix slash should resolve selected command")
	}
	if !ShouldResolveSelected(Option{Cmd: "/access jump.example.com", FillValue: "/access jump"}, "/access jump") {
		t.Fatalf("resolve should consider FillValue prefixes")
	}
	if !ShouldResolveSelected(Option{Cmd: "/access prod", Desc: "Production"}, "/access pro") {
		t.Fatalf("resolve should consider access descriptions")
	}
}
