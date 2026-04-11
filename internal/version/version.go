package version

import "strings"

var (
	Version   = "dev"
	Commit    = "unknown"
	BuildDate = "unknown"
)

func String() string {
	parts := []string{strings.TrimSpace(Version)}
	if v := strings.TrimSpace(Commit); v != "" && v != "unknown" {
		parts = append(parts, "commit "+v)
	}
	if v := strings.TrimSpace(BuildDate); v != "" && v != "unknown" {
		parts = append(parts, "built "+v)
	}
	if len(parts) == 1 {
		return parts[0]
	}
	return parts[0] + " (" + strings.Join(parts[1:], ", ") + ")"
}
