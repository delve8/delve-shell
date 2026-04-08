package ui

import "github.com/charmbracelet/lipgloss"

// TUI styles. All use lipgloss; colors are ANSI 256 (e.g. 1=red, 2=green, 8=dim gray).
//
// Layout roles (pick from here; avoid one-off lipgloss in feature code):
//   - Footer/title bar: titleStyle, status*Style, pendingActionStyle (fixed bottom band; not transcript).
//   - Transcript — primary: execStyle, resultStyle; system: suggestStyle, errStyle, delveMsg.
//   - Approval card — T1 title: approvalHeaderStyle; T2 risk band: riskReadOnlyStyle / riskLowStyle / riskHighStyle;
//     T3 command: execStyle (+ execAuto* for auto-approve spans); T4 section labels: metaLabelStyle;
//     T5 section body: metaDetailStyle; T6 choice rows under input: suggestStyle / suggestHi.

var (
	// Footer/status and layout
	titleStyle         = lipgloss.NewStyle().Foreground(lipgloss.Color("246")) // footer line: mode
	separatorStyle     = lipgloss.NewStyle().Foreground(lipgloss.Color("8"))   // horizontal rule between viewport/footer/input
	statusIdleStyle    = lipgloss.NewStyle().Foreground(lipgloss.Color("246")) // [IDLE] / [空闲] — same tone as footer text
	statusRunningStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("11"))  // [PROCESSING] / [处理中] — softer yellow
)

var (
	// Commands and results
	execStyle   = lipgloss.NewStyle().Foreground(lipgloss.Color("6")).Italic(true)  // command text (pending, run line)
	resultStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("8")).MarginLeft(2) // command output
	// Auto-approve highlight on pending approval: safe segment vs risky segment vs separators (|, &&, spaces).
	execAutoSafeStyle    = lipgloss.NewStyle().Foreground(lipgloss.Color("10")).Italic(true)
	execAutoRiskStyle    = lipgloss.NewStyle().Foreground(lipgloss.Color("9")).Bold(true).Italic(true)
	execAutoNeutralStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("246")).Italic(true)
)

var (
	// General secondary text and lists
	suggestStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("7"))              // secondary: help, config success, approval reason, unselected list row
	suggestHi    = lipgloss.NewStyle().Foreground(lipgloss.Color("13")).Bold(true)  // highlighted list row (choice 1/2/3, slash options)
	hintStyle    = lipgloss.NewStyle().Foreground(lipgloss.Color("8")).Italic(true) // tertiary hint: copy hint, "Copied"
	// Input-history browse chrome: dimmer than suggest list rows (7) and input text so it reads as UI, not typed content.
	inputHistBrowsingHintStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("241")).Italic(true)
	sessionSwitchedStyle       = lipgloss.NewStyle().Foreground(lipgloss.Color("10")).Italic(true) // "Switched to session: xxx" hint at bottom
)

var (
	// Pending/choice: footer status + operation hint (e.g. [待确认] 1 or 2)
	pendingActionStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("11"))
)

var (
	// Approval/sensitive cards: card title and risk labels
	approvalHeaderStyle           = lipgloss.NewStyle().Foreground(lipgloss.Color("12")).Bold(true) // card title (cyan)
	riskReadOnlyStyle             = lipgloss.NewStyle().Foreground(lipgloss.Color("2")).Bold(true)  // [READ-ONLY]
	riskLowStyle                  = lipgloss.NewStyle().Foreground(lipgloss.Color("3")).Bold(true)  // [LOW-RISK]
	riskHighStyle                 = lipgloss.NewStyle().Foreground(lipgloss.Color("1")).Bold(true)  // [HIGH-RISK]
	approvalDecisionApprovedStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("2")).Bold(true)  // Decision: approved
	approvalDecisionRejectedStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("1")).Bold(true)  // Decision: rejected
	// Dismiss: distinct from approve/reject and from neutral card lines (Why/Summary).
	approvalDecisionDismissStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("245")).Bold(true).Italic(true)
	// Approval card: section labels (Risk Hint / Summary / Purpose) vs body lines (policy, summary, purpose text).
	metaLabelStyle  = lipgloss.NewStyle().Foreground(lipgloss.Color("7")).Bold(true)
	metaDetailStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("252"))
)

var (
	// Errors and config errors
	errStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("9")) // error messages
)

// Input field (textinput) styles; used in NewModel.
var (
	inputPromptStyle      = lipgloss.NewStyle().Foreground(lipgloss.Color("7"))
	inputTextStyle        = lipgloss.NewStyle()
	inputCursorStyle      = lipgloss.NewStyle().Foreground(lipgloss.Color("13"))
	inputPlaceholderStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("246"))
)

// Startup title (one line, transcript scrollback).
var startupTitleStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("6")).Bold(true)
