package history

import (
	"regexp"
)

// RedactText applies light-weight regex based redaction to hide common secrets
// like passwords, tokens, and private key blocks before storing into history.
// It is intentionally conservative and focuses on high-signal patterns.
func RedactText(s string) string {
	if s == "" {
		return s
	}
	out := s
	for _, r := range redactRules {
		out = r.re.ReplaceAllString(out, r.repl)
	}
	return out
}

type redactRule struct {
	re   *regexp.Regexp
	repl string
}

var redactRules = []redactRule{
	// Generic key=value style secrets: password, token, secret, api key etc.
	{
		re:   regexp.MustCompile(`(?i)\b(password|passwd|pwd|secret|token|apikey|api_key|access_key|secret_key)\s*[:=]\s*([^\s]+)`),
		repl: `${1}=[REDACTED]`,
	},
	// Env-style *_SECRET=... variables.
	{
		re:   regexp.MustCompile(`(?i)\b([A-Z0-9_]*SECRET[A-Z0-9_]*)\s*=\s*([^\s]+)`),
		repl: `${1}=[REDACTED]`,
	},
	// AWS access key id (pattern only, no key name).
	{
		re:   regexp.MustCompile(`\bAKIA[0-9A-Z]{12,20}\b`),
		repl: `[AWS_ACCESS_KEY_ID]`,
	},
	// AWS secret access key when labeled.
	{
		re:   regexp.MustCompile(`(?i)\baws_secret_access_key\s*[:=]\s*([^\s]+)`),
		repl: `aws_secret_access_key=[REDACTED]`,
	},
	// PEM private key block: squash body into a placeholder.
	{
		re:   regexp.MustCompile(`(?s)-----BEGIN [A-Z ]*PRIVATE KEY-----.*?-----END [A-Z ]*PRIVATE KEY-----`),
		repl: `[REDACTED_PRIVATE_KEY_BLOCK]`,
	},
}

