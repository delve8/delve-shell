package ui

import (
	"strings"

	"github.com/charmbracelet/glamour"
	glamouransi "github.com/charmbracelet/glamour/ansi"
	"github.com/charmbracelet/glamour/styles"
	"github.com/charmbracelet/x/ansi"
	"github.com/muesli/termenv"

	"delve-shell/internal/textwrap"
)

// Inline code (`...`) foreground in glamour's dark theme defaults to 256-color 203 (reddish).
// Use a gold/yellow tone for better readability and consistency with the rest of the TUI palette.
const aiMarkdownInlineCodeColor256 = "220"

func delveAIMarkdownStyle() glamouransi.StyleConfig {
	cfg := styles.DarkStyleConfig
	z := uint(0)
	cfg.Document.Margin = &z
	cfg.Code = glamouransi.StyleBlock{
		StylePrimitive: glamouransi.StylePrimitive{
			Prefix:          " ",
			Suffix:          " ",
			Color:           ptrString(aiMarkdownInlineCodeColor256),
			BackgroundColor: ptrString("236"),
		},
	}
	// Default dark theme repeats ATX hashes (## / ###) as Prefix; hide them so only the
	// heading text is shown, still styled by the shared Heading block (color/bold).
	for _, patch := range []struct {
		dst *glamouransi.StyleBlock
		src glamouransi.StyleBlock
	}{
		{&cfg.H2, styles.DarkStyleConfig.H2},
		{&cfg.H3, styles.DarkStyleConfig.H3},
		{&cfg.H4, styles.DarkStyleConfig.H4},
		{&cfg.H5, styles.DarkStyleConfig.H5},
		{&cfg.H6, styles.DarkStyleConfig.H6},
	} {
		b := patch.src
		b.Prefix = ""
		*patch.dst = b
	}
	return cfg
}

func ptrString(s string) *string { return &s }

// minAIMarkdownInnerWidth is the minimum content width before we skip glamour and fall back to plain text.
const minAIMarkdownInnerWidth = 24

// renderAILineTranscript renders assistant markdown for the transcript using glamour (ANSI 256).
// If glamour fails or the terminal is too narrow, falls back to plain textwrap.
func renderAILineTranscript(markdown string, totalWidth int) []string {
	if totalWidth <= 0 {
		totalWidth = minContentWidthFallback
	}
	innerW := totalWidth
	if strings.TrimSpace(markdown) == "" {
		return nil
	}
	if innerW < minAIMarkdownInnerWidth {
		return []string{textwrap.WrapString(markdown, totalWidth)}
	}

	md := relaxMarkdownStrongAdjacentCJK(markdown)

	r, err := glamour.NewTermRenderer(
		glamour.WithWordWrap(innerW),
		glamour.WithColorProfile(termenv.ANSI256),
		glamour.WithStyles(delveAIMarkdownStyle()),
	)
	if err != nil {
		return []string{textwrap.WrapString(md, totalWidth)}
	}
	out, err := r.Render(md)
	if err != nil {
		return []string{textwrap.WrapString(md, totalWidth)}
	}
	out = strings.TrimRight(out, "\n")
	lines := strings.Split(out, "\n")
	if len(lines) == 0 {
		return nil
	}

	rendered := make([]string, 0, len(lines))
	for _, line := range lines {
		joined := line
		if totalWidth > 0 && ansi.StringWidth(joined) > totalWidth {
			joined = ansi.Truncate(joined, totalWidth, "")
		}
		rendered = append(rendered, joined)
	}
	return rendered
}
