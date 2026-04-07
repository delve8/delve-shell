package history

import (
	"regexp"
	"strconv"
)

// RedactText applies light-weight regex based redaction to hide common secrets
// like passwords, tokens, and private key blocks before storing into history
// or returning redacted tool output to the model.
// It is intentionally heuristic: missed patterns may still leak; extend [redactRules] as needed.
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

// RedactedToolResultMessage builds the same shape as execute_command / run_skill tool returns,
// with stdout, stderr, and error text passed through [RedactText]. Used when the user marks
// result_contains_secrets (or equivalent) so the model still receives structure without raw secrets.
func RedactedToolResultMessage(stdout, stderr string, exitCode int, execErr error) string {
	stdout = RedactText(stdout)
	stderr = RedactText(stderr)
	msg := "stdout:\n" + stdout
	if stderr != "" {
		msg += "\nstderr:\n" + stderr
	}
	msg += "\nexit_code: " + strconv.Itoa(exitCode)
	if execErr != nil && exitCode == 0 {
		msg += "\nerror: " + RedactText(execErr.Error())
	}
	return msg
}

type redactRule struct {
	re   *regexp.Regexp
	repl string
}

// Order: larger / structured secrets first, then labeled headers, then generic key=value.
var redactRules = []redactRule{
	// PEM / OpenSSH private key blocks.
	{
		re:   regexp.MustCompile(`(?s)-----BEGIN [A-Z ]*PRIVATE KEY-----.*?-----END [A-Z ]*PRIVATE KEY-----`),
		repl: `[REDACTED_PRIVATE_KEY_BLOCK]`,
	},
	// JWT (JWS-style three base64url segments; requires eyJ header start to limit false positives).
	{
		re:   regexp.MustCompile(`\beyJ[A-Za-z0-9_-]{10,}\.[A-Za-z0-9_-]{10,}\.[A-Za-z0-9_-]{10,}\b`),
		repl: `[REDACTED_JWT]`,
	},
	// HTTP Authorization: Bearer …
	{
		re:   regexp.MustCompile(`(?i)\bauthorization\s*:\s*Bearer\s+\S+`),
		repl: `authorization: Bearer [REDACTED]`,
	},
	// Standalone Bearer prefix (e.g. in curl -H copies).
	{
		re:   regexp.MustCompile(`(?i)\bBearer\s+[A-Za-z0-9._~+/=-]{20,}\b`),
		repl: `Bearer [REDACTED]`,
	},
	// HTTP Basic …
	{
		re:   regexp.MustCompile(`(?i)\bBasic\s+[A-Za-z0-9+/=]{24,}`),
		repl: `Basic [REDACTED]`,
	},
	// URL query parameters for common secret names.
	{
		re:   regexp.MustCompile(`(?i)([?&])(access_token|refresh_token|id_token|api_key|apikey|client_secret|password|token|secret|key)=([^&\s#]+)`),
		repl: `${1}${2}=[REDACTED]`,
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
	// GitHub PATs (classic and fine-grained style prefixes).
	{
		re:   regexp.MustCompile(`\bghp_[A-Za-z0-9]{36,}\b`),
		repl: `[REDACTED_GITHUB_PAT]`,
	},
	{
		re:   regexp.MustCompile(`\bgithub_pat_[A-Za-z0-9_]{20,}\b`),
		repl: `[REDACTED_GITHUB_PAT]`,
	},
	// Slack tokens.
	{
		re:   regexp.MustCompile(`\bxox[baprs]-[A-Za-z0-9-]{10,}\b`),
		repl: `[REDACTED_SLACK_TOKEN]`,
	},
	// Stripe secret keys.
	{
		re:   regexp.MustCompile(`\bsk_(?:live|test)_[0-9a-zA-Z]{20,}\b`),
		repl: `[REDACTED_STRIPE_KEY]`,
	},
	// Google API keys (browser / simple key style).
	{
		re:   regexp.MustCompile(`\bAIza[0-9A-Za-z_-]{35}\b`),
		repl: `[REDACTED_GOOGLE_API_KEY]`,
	},
	// Generic key=value style secrets.
	{
		re:   regexp.MustCompile(`(?i)\b(password|passwd|pwd|secret|token|apikey|api_key|access_key|secret_key|refresh_token|client_secret|auth_token|id_token|bearer_token)\s*[:=]\s*([^\s]+)`),
		repl: `${1}=[REDACTED]`,
	},
	// Env-style *_SECRET=... variables.
	{
		re:   regexp.MustCompile(`(?i)\b([A-Z0-9_]*SECRET[A-Z0-9_]*)\s*=\s*([^\s]+)`),
		repl: `${1}=[REDACTED]`,
	},
	// Env-style *_PASSWORD=... (e.g. DB_PASSWORD=...).
	{
		re:   regexp.MustCompile(`(?i)\b([A-Z0-9_]*PASSWORD)\s*=\s*([^\s]+)`),
		repl: `${1}=[REDACTED]`,
	},
}
