package history

import "testing"

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

