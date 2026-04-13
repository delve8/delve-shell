package agent

import (
	"strings"
	"testing"
)

func TestAllowlistExecutionParagraphRequestsReadableMultilineScripts(t *testing.T) {
	p := allowlistExecutionParagraph()
	for _, want := range []string{
		"single execute_command",
		"one multi-line command string",
		"command text itself should contain newline characters",
		"do not treat this as output formatting",
		`trailing \ for`,
		"Only use a single-line command when it is genuinely short",
	} {
		if !strings.Contains(p, want) {
			t.Fatalf("allowlist execution paragraph missing %q:\n%s", want, p)
		}
	}
}
