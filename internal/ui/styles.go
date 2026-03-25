package ui

import "github.com/charmbracelet/lipgloss"

// TUI styles. All use lipgloss; colors are ANSI 256 (e.g. 1=red, 2=green, 8=dim gray).

var (
	// Header and layout
	titleStyle         = lipgloss.NewStyle().Bold(true)                                  // title line: mode
	separatorStyle     = lipgloss.NewStyle().Foreground(lipgloss.Color("8"))             // horizontal rule between header/content/input
	statusIdleStyle    = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("10")) // [IDLE] / [空闲] — green, stands out
	statusRunningStyle = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("11")) // [PROCESSING] / [处理中] — yellow, stands out
)

var (
	// Commands and results
	execStyle   = lipgloss.NewStyle().Foreground(lipgloss.Color("6")).Italic(true)  // command text (pending, run line)
	resultStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("8")).MarginLeft(2) // command output
)

var (
	// General secondary text and lists
	suggestStyle         = lipgloss.NewStyle().Foreground(lipgloss.Color("7"))               // secondary: help, config success, approval reason, unselected list row
	suggestHi            = lipgloss.NewStyle().Foreground(lipgloss.Color("13")).Bold(true)   // highlighted list row (choice 1/2/3, slash options)
	hintStyle            = lipgloss.NewStyle().Foreground(lipgloss.Color("8")).Italic(true)  // tertiary hint: copy hint, "Copied"
	sessionSwitchedStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("10")).Italic(true) // "Switched to session: xxx" hint at bottom
)

var (
	// Pending/choice: header status + operation hint (e.g. [待确认] 1 or 2)
	pendingActionStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("11")).Bold(true)
)

var (
	// Approval/sensitive cards: card title and risk labels
	approvalHeaderStyle           = lipgloss.NewStyle().Foreground(lipgloss.Color("12")).Bold(true) // card title (cyan)
	riskReadOnlyStyle             = lipgloss.NewStyle().Foreground(lipgloss.Color("2")).Bold(true)  // [READ-ONLY]
	riskLowStyle                  = lipgloss.NewStyle().Foreground(lipgloss.Color("3")).Bold(true)  // [LOW-RISK]
	riskHighStyle                 = lipgloss.NewStyle().Foreground(lipgloss.Color("1")).Bold(true)  // [HIGH-RISK]
	approvalDecisionApprovedStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("2")).Bold(true)  // Decision: approved
	approvalDecisionRejectedStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("1")).Bold(true)  // Decision: rejected
)

var (
	// Errors and config errors
	errStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("9")) // error messages
)

// Input field (textinput) styles; used in NewModel.
var (
	inputPromptStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("7"))
	inputTextStyle   = lipgloss.NewStyle()
	inputCursorStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("13"))
)
