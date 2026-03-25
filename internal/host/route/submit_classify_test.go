package route

import (
	"strings"
	"testing"
)

func TestClassifyUserSubmit_NewSession(t *testing.T) {
	cases := []struct {
		name string
		in   string
	}{
		{"exact", "/new"},
		{"leading_space", " /new"},
		{"trailing_space", "/new "},
		{"both_spaces", "  /new  "},
		{"tab_prefix", "\t/new"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := ClassifyUserSubmit(tc.in)
			if got.Kind != UserSubmitNewSession {
				t.Fatalf("kind: want NewSession, got %v (full %#v)", got.Kind, got)
			}
			if got.SessionID != "" {
				t.Fatalf("SessionID should be empty, got %q", got.SessionID)
			}
		})
	}
}

func TestClassifyUserSubmit_NotNewSession(t *testing.T) {
	cases := []string{
		"/newx",
		"/NEW",
		"//new",
		"/new/",
		"new",
		"/ new",
		"/new-session",
	}
	for _, in := range cases {
		t.Run(in, func(t *testing.T) {
			got := ClassifyUserSubmit(in)
			if got.Kind == UserSubmitNewSession {
				t.Fatalf("should not classify as NewSession: %q -> %#v", in, got)
			}
		})
	}
}

func TestClassifyUserSubmit_SwitchSession(t *testing.T) {
	cases := []struct {
		name   string
		in     string
		wantID string
	}{
		{"simple_id", "/sessions abc", "abc"},
		{"uuid_like", "/sessions 550e8400-e29b-41d4-a716-446655440000", "550e8400-e29b-41d4-a716-446655440000"},
		{"spaces_around_id", "/sessions   xyz  ", "xyz"},
		{"prefix_with_leading_ws", "  /sessions foo", "foo"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := ClassifyUserSubmit(tc.in)
			if got.Kind != UserSubmitSwitchSession {
				t.Fatalf("kind: want SwitchSession, got %v for %q", got.Kind, tc.in)
			}
			if got.SessionID != tc.wantID {
				t.Fatalf("SessionID: want %q, got %q", tc.wantID, got.SessionID)
			}
		})
	}
}

func TestClassifyUserSubmit_SessionsNoIDBecomesLLM(t *testing.T) {
	// After TrimSpace these collapse to "/sessions" (no trailing space+payload), matching host prefix `/sessions `.
	for _, in := range []string{"/sessions ", "/sessions   ", "/sessions\n", "/sessions\t"} {
		got := ClassifyUserSubmit(in)
		if got.Kind != UserSubmitLLM {
			t.Fatalf("want LLM for %q, got %#v", in, got)
		}
	}
}

func TestClassifyUserSubmit_SessionsPrefixWithoutSpace(t *testing.T) {
	// "/sessionsfoo" is not a switch command (prefix requires " /sessions " pattern in host).
	got := ClassifyUserSubmit("/sessionsfoo")
	if got.Kind != UserSubmitLLM {
		t.Fatalf("want LLM, got %#v", got)
	}
}

func TestClassifyUserSubmit_LLMChat(t *testing.T) {
	cases := []string{
		"",
		"hello",
		"run ls",
		"/help",
		"/config",
		"/cancel",
		"/run echo hi",
		" /not-a-slash-command ",
		"你好",
		"multi\nline",
	}
	for _, in := range cases {
		t.Run(shortLabel(in), func(t *testing.T) {
			got := ClassifyUserSubmit(in)
			if got.Kind != UserSubmitLLM {
				t.Fatalf("want LLM for %q, got %#v", in, got)
			}
			if got.SessionID != "" {
				t.Fatalf("SessionID should be empty for LLM, got %q", got.SessionID)
			}
		})
	}
}

func shortLabel(s string) string {
	s = strings.ReplaceAll(s, "\n", "\\n")
	if len(s) > 48 {
		return s[:45] + "..."
	}
	if s == "" {
		return "empty"
	}
	return s
}

func TestClassifyUserSubmit_StabilityTable(t *testing.T) {
	type row struct {
		in       string
		wantKind UserSubmitKind
		wantID   string
	}
	rows := []row{
		{"/new", UserSubmitNewSession, ""},
		{"/sessions a", UserSubmitSwitchSession, "a"},
		{"/sessions", UserSubmitLLM, ""},
		{"/sessions\tid", UserSubmitLLM, ""},
		{"/sessions \tid\t", UserSubmitSwitchSession, "id"},
		{"/SESSIONS x", UserSubmitLLM, ""},
		{"/sessionsX", UserSubmitLLM, ""},
		{"/sessions /x", UserSubmitSwitchSession, "/x"},
		{"/sessions ../evil", UserSubmitSwitchSession, "../evil"},
		{"/sessions .", UserSubmitSwitchSession, "."},
		{"/sessions ..", UserSubmitSwitchSession, ".."},
		{"/sessions -", UserSubmitSwitchSession, "-"},
		{"/sessions _", UserSubmitSwitchSession, "_"},
		{"/sessions 0", UserSubmitSwitchSession, "0"},
		{"/sessions 00", UserSubmitSwitchSession, "00"},
		{"/sessions jsonl", UserSubmitSwitchSession, "jsonl"},
		{"/sessions file.jsonl", UserSubmitSwitchSession, "file.jsonl"},
		{"/sessions a b", UserSubmitSwitchSession, "a b"},
		{"/sessions  a  b  ", UserSubmitSwitchSession, "a  b"},
		{"/sessions \n", UserSubmitLLM, ""},
		{"/sessions \nfoo", UserSubmitSwitchSession, "foo"},
		{"\n/sessions x\n", UserSubmitSwitchSession, "x"},
		{"/sessions x\ny", UserSubmitSwitchSession, "x\ny"},
		{"/sessions x\ty", UserSubmitSwitchSession, "x\ty"},
		{"/sessions x\u00a0y", UserSubmitSwitchSession, "x\u00a0y"},
		{"/sessions x　y", UserSubmitSwitchSession, "x　y"},
		{"/sessions x\x00y", UserSubmitSwitchSession, "x\x00y"},
		{strings.Repeat("/sessions ", 1) + "id", UserSubmitSwitchSession, "id"},
		{"/new ", UserSubmitNewSession, ""},
		{" /new", UserSubmitNewSession, ""},
		{"/new\n", UserSubmitNewSession, ""},
		{"/new\t", UserSubmitNewSession, ""},
		{"/new\u00a0", UserSubmitNewSession, ""},
		{"/new　", UserSubmitNewSession, ""},
		{"x/new", UserSubmitLLM, ""},
		{"pre/new", UserSubmitLLM, ""},
		{"/newx", UserSubmitLLM, ""},
		{"/new/", UserSubmitLLM, ""},
		{"/new//", UserSubmitLLM, ""},
		{"/NeW", UserSubmitLLM, ""},
		{"/NEW", UserSubmitLLM, ""},
		{"/New", UserSubmitLLM, ""},
		{"/n", UserSubmitLLM, ""},
		{"/ne", UserSubmitLLM, ""},
		{"/news", UserSubmitLLM, ""},
		{"/session", UserSubmitLLM, ""},
		{"/sessionx", UserSubmitLLM, ""},
		{"/sessions", UserSubmitLLM, ""},
		{"/sessionsx", UserSubmitLLM, ""},
		{"/sessions/", UserSubmitLLM, ""},
		{"/sessions//", UserSubmitLLM, ""},
		{"/sessions /", UserSubmitSwitchSession, "/"},
		{"/sessions  /", UserSubmitSwitchSession, "/"},
		{"/sessions \t/", UserSubmitSwitchSession, "/"},
		{"/sessions \t / \t", UserSubmitSwitchSession, "/"},
		{"/sessions a/b", UserSubmitSwitchSession, "a/b"},
		{"/sessions :foo:", UserSubmitSwitchSession, ":foo:"},
		{"/sessions [bracket]", UserSubmitSwitchSession, "[bracket]"},
		{"/sessions (paren)", UserSubmitSwitchSession, "(paren)"},
		{"/sessions {brace}", UserSubmitSwitchSession, "{brace}"},
		{"/sessions <tag>", UserSubmitSwitchSession, "<tag>"},
		{"/sessions \"quoted\"", UserSubmitSwitchSession, "\"quoted\""},
		{"/sessions 'single'", UserSubmitSwitchSession, "'single'"},
		{"/sessions `backtick`", UserSubmitSwitchSession, "`backtick`"},
		{"/sessions $VAR", UserSubmitSwitchSession, "$VAR"},
		{"/sessions %PATH%", UserSubmitSwitchSession, "%PATH%"},
		{"/sessions #hash", UserSubmitSwitchSession, "#hash"},
		{"/sessions ;semi", UserSubmitSwitchSession, ";semi"},
		{"/sessions |pipe", UserSubmitSwitchSession, "|pipe"},
		{"/sessions &amp", UserSubmitSwitchSession, "&amp"},
		{"/sessions &&", UserSubmitSwitchSession, "&&"},
		{"/sessions ||", UserSubmitSwitchSession, "||"},
		{"/sessions **", UserSubmitSwitchSession, "**"},
		{"/sessions ??", UserSubmitSwitchSession, "??"},
		{"/sessions +=", UserSubmitSwitchSession, "+="},
		{"/sessions =>", UserSubmitSwitchSession, "=>"},
		{"/sessions ->", UserSubmitSwitchSession, "->"},
		{"/sessions <-", UserSubmitSwitchSession, "<-"},
		{"/sessions ::", UserSubmitSwitchSession, "::"},
		{"/sessions ..\\", UserSubmitSwitchSession, "..\\"},
		{"/sessions C:\\", UserSubmitSwitchSession, "C:\\"},
		{"/sessions ./rel", UserSubmitSwitchSession, "./rel"},
		{"/sessions ../rel", UserSubmitSwitchSession, "../rel"},
		{"/sessions ~", UserSubmitSwitchSession, "~"},
		{"/sessions ~user", UserSubmitSwitchSession, "~user"},
		{"/sessions @user", UserSubmitSwitchSession, "@user"},
		{"/sessions user@host", UserSubmitSwitchSession, "user@host"},
		{"/sessions host:22", UserSubmitSwitchSession, "host:22"},
		{"/sessions 127.0.0.1", UserSubmitSwitchSession, "127.0.0.1"},
		{"/sessions ::1", UserSubmitSwitchSession, "::1"},
		{"/sessions http://x", UserSubmitSwitchSession, "http://x"},
		{"/sessions https://x", UserSubmitSwitchSession, "https://x"},
		{"/sessions ftp://x", UserSubmitSwitchSession, "ftp://x"},
		{"/sessions ws://x", UserSubmitSwitchSession, "ws://x"},
		{"/sessions wss://x", UserSubmitSwitchSession, "wss://x"},
		{"/sessions data:,", UserSubmitSwitchSession, "data:,"},
		{"/sessions javascript:alert(1)", UserSubmitSwitchSession, "javascript:alert(1)"},
		{"/sessions file:///etc/passwd", UserSubmitSwitchSession, "file:///etc/passwd"},
		{"/sessions %2e%2e%2f", UserSubmitSwitchSession, "%2e%2e%2f"},
		{"/sessions %00", UserSubmitSwitchSession, "%00"},
		{"/sessions %20", UserSubmitSwitchSession, "%20"},
		{"/sessions %09", UserSubmitSwitchSession, "%09"},
		{"/sessions \u200b", UserSubmitSwitchSession, "\u200b"},
		{"/sessions \ufeff", UserSubmitSwitchSession, "\ufeff"},
		{"/sessions \u202e", UserSubmitSwitchSession, "\u202e"},
		{strings.Repeat("a", 200), UserSubmitLLM, ""},
		{"/sessions " + strings.Repeat("b", 200), UserSubmitSwitchSession, strings.Repeat("b", 200)},
		{"/sessions " + strings.Repeat("c", 1), UserSubmitSwitchSession, "c"},
		{"/sessions " + strings.Repeat("d ", 50), UserSubmitSwitchSession, strings.TrimSpace(strings.Repeat("d ", 50))},
	}
	for i, tc := range rows {
		t.Run(shortLabel(tc.in), func(t *testing.T) {
			got := ClassifyUserSubmit(tc.in)
			if got.Kind != tc.wantKind {
				t.Fatalf("row %d kind: in=%q want %v got %v", i, tc.in, tc.wantKind, got.Kind)
			}
			if got.SessionID != tc.wantID {
				t.Fatalf("row %d SessionID: in=%q want %q got %q", i, tc.in, tc.wantID, got.SessionID)
			}
		})
	}
}

func TestClassifyUserSubmit_UnicodeLLM(t *testing.T) {
	in := "请总结日志"
	got := ClassifyUserSubmit(in)
	if got.Kind != UserSubmitLLM {
		t.Fatalf("want LLM, got %#v", got)
	}
}

func TestClassifyUserSubmit_SwitchPreservesInnerSpaces(t *testing.T) {
	got := ClassifyUserSubmit("/sessions  a  b c ")
	if got.SessionID != "a  b c" {
		t.Fatalf("want inner spaces preserved, got %q", got.SessionID)
	}
}

func BenchmarkClassifyUserSubmit(b *testing.B) {
	samples := []string{
		"/new",
		"/sessions abc-123",
		"hello world this is a longer chat line for benchmarking purposes",
		"/help",
	}
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		for _, s := range samples {
			_ = ClassifyUserSubmit(s)
		}
	}
}
