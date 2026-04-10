package history

import (
	"strings"
	"testing"
)

func TestRedactText_Empty(t *testing.T) {
	if got := RedactText(""); got != "" {
		t.Fatalf("RedactText(\"\") = %q, want empty", got)
	}
}

func TestRedactText_GenericSecrets(t *testing.T) {
	tests := []struct {
		in   string
		want string
	}{
		{
			in:   "password=abc123",
			want: "password=[REDACTED]",
		},
		{
			in:   "PWD: abc123",
			want: "PWD=[REDACTED]",
		},
		{
			in:   "token = xyz",
			want: "token=[REDACTED]",
		},
		{
			in:   "no secret here",
			want: "no secret here",
		},
	}

	for _, tt := range tests {
		got := RedactText(tt.in)
		if got != tt.want {
			t.Errorf("RedactText(%q) = %q, want %q", tt.in, got, tt.want)
		}
	}
}

func TestRedactText_EnvSecretVars(t *testing.T) {
	in := "DB_PASSWORD=abc JWT_SECRET=xyz"
	got := RedactText(in)
	if got == in {
		t.Fatal("expected env-style secrets to be redacted")
	}
	if wantSub := "JWT_SECRET=[REDACTED]"; !contains(got, wantSub) {
		t.Errorf("expected %q to contain %q", got, wantSub)
	}
}

func TestRedactText_AWSKeys(t *testing.T) {
	in := "aws_secret_access_key = foo AKIA1234567890ABCD"
	got := RedactText(in)
	if contains(got, "foo") {
		t.Errorf("aws secret access key value should be redacted, got %q", got)
	}
	if contains(got, "AKIA1234567890ABCD") {
		t.Errorf("AWS access key id should be redacted, got %q", got)
	}
}

func TestRedactText_JWT(t *testing.T) {
	in := "tok eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJzdWIiOiIxMjM0NTY3ODkwIn0.dozjgNryP4J3jVmNHl0w5N_XwpLbqQu0isrO5H2NcVc"
	got := RedactText(in)
	if contains(got, "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9") {
		t.Errorf("JWT should be redacted, got %q", got)
	}
	if !contains(got, "[REDACTED_JWT]") {
		t.Errorf("expected JWT placeholder, got %q", got)
	}
}

func TestRedactText_AuthorizationBearer(t *testing.T) {
	in := "curl -H \"Authorization: Bearer somelongtokenthinghere\" https://x"
	got := RedactText(in)
	if contains(got, "somelongtokenthinghere") {
		t.Errorf("bearer value redacted, got %q", got)
	}
}

func TestRedactText_QueryToken(t *testing.T) {
	in := "https://x?access_token=sekret&foo=1"
	got := RedactText(in)
	if contains(got, "sekret") {
		t.Errorf("query token redacted, got %q", got)
	}
}

func TestRedactedToolResultMessage(t *testing.T) {
	got := RedactedToolResultMessage("password=abc", "err", 0, nil)
	if contains(got, "abc") {
		t.Fatalf("stdout secret leaked: %q", got)
	}
	if !contains(got, "exit_code: 0") || !contains(got, "stderr:") {
		t.Fatalf("expected shape: %q", got)
	}
}

func TestRedactedToolResultMessage_TruncatesLargeOutput(t *testing.T) {
	stdout := "password=abc\n" + strings.Repeat("x", ToolOutputMaxBytes+2048)
	got := RedactedToolResultMessage(stdout, "", 0, nil)
	if contains(got, "abc") {
		t.Fatalf("stdout secret leaked: %q", got)
	}
	if !contains(got, "[truncated, omitted ") {
		t.Fatalf("expected truncation marker: %q", got)
	}
}

func TestRedactText_PrivateKeyBlock(t *testing.T) {
	in := "header\n-----BEGIN PRIVATE KEY-----\nABCDEF\n-----END PRIVATE KEY-----\nfooter"
	got := RedactText(in)
	if contains(got, "BEGIN PRIVATE KEY") || contains(got, "ABCDEF") {
		t.Errorf("private key block should be redacted, got %q", got)
	}
	if !contains(got, "[REDACTED_PRIVATE_KEY_BLOCK]") {
		t.Errorf("expected private key placeholder, got %q", got)
	}
}

func contains(s, sub string) bool {
	return len(sub) == 0 || (len(s) >= len(sub) && (index(s, sub) >= 0))
}

// index is a minimal substring search to avoid pulling in strings just for tests.
func index(s, sub string) int {
	n := len(s)
	m := len(sub)
	if m == 0 {
		return 0
	}
	if m > n {
		return -1
	}
	for i := 0; i <= n-m; i++ {
		if s[i:i+m] == sub {
			return i
		}
	}
	return -1
}
