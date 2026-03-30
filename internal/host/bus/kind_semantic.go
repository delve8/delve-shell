package bus

import (
	"fmt"
	"strings"
	"unicode/utf8"
)

// SemanticLabel names the event shape in architecture terms for human-readable tracing.
// Wire Kind string values stay stable for logs and tests; this label is for human traceability only.
func (k Kind) SemanticLabel() string {
	switch k {
	case KindSessionNewRequested:
		return "SessionNewRequested"
	case KindSessionSwitchRequested:
		return "SessionSwitchRequested"
	case KindHistoryPreviewRequested:
		return "HistoryPreviewRequested"
	case KindUserChatSubmitted:
		return "AIRequested"
	case KindConfigUpdated:
		return "ConfigReloaded"
	case KindCancelRequested:
		return "CancelRequested"
	case KindExecDirectRequested:
		return "ExecDirectRequested"
	case KindRemoteOnRequested:
		return "RemoteConnectRequested"
	case KindRemoteOffRequested:
		return "RemoteDisconnectRequested"
	case KindRemoteAuthResponseSubmitted:
		return "RemoteAuthAnswerSubmitted"
	case KindApprovalRequested:
		return "ApprovalRequested"
	case KindSensitiveConfirmationRequested:
		return "SensitiveConfirmationRequested"
	case KindAgentExecEvent:
		return "CommandExecuted"
	case KindAgentUnknown:
		return "AgentUIPassthrough"
	case KindLLMRunCompleted:
		return "LLMRunCompleted"
	default:
		return "UnknownKind"
	}
}

// RedactedSummary returns a single-line, non-sensitive description for tracing (no passwords or API keys).
func (e Event) RedactedSummary() string {
	k := e.Kind
	label := k.SemanticLabel()
	switch k {
	case KindSessionNewRequested, KindRemoteOffRequested, KindCancelRequested, KindConfigUpdated:
		return label
	case KindSessionSwitchRequested:
		return fmt.Sprintf("%s session_id=%s", label, clipOneLine(e.SessionID, 64))
	case KindHistoryPreviewRequested:
		return fmt.Sprintf("%s session_id=%s", label, clipOneLine(e.SessionID, 64))
	case KindUserChatSubmitted:
		return fmt.Sprintf("%s text=%q", label, clipOneLine(e.UserText, 80))
	case KindExecDirectRequested:
		return fmt.Sprintf("%s command=%q", label, clipOneLine(e.Command, 120))
	case KindRemoteOnRequested:
		return fmt.Sprintf("%s target=%q", label, clipOneLine(e.RemoteTarget, 120))
	case KindRemoteAuthResponseSubmitted:
		r := e.RemoteAuthResponse
		return fmt.Sprintf("%s target=%q kind=%s", label, clipOneLine(r.Target, 120), clipOneLine(r.Kind, 32))
	case KindApprovalRequested:
		if e.Approval == nil {
			return label + " approval=nil"
		}
		return fmt.Sprintf("%s command=%q", label, clipOneLine(e.Approval.Command, 120))
	case KindSensitiveConfirmationRequested:
		if e.Sensitive == nil {
			return label + " sensitive=nil"
		}
		return fmt.Sprintf("%s command=%q", label, clipOneLine(e.Sensitive.Command, 120))
	case KindAgentExecEvent:
		v := e.AgentExec
		return fmt.Sprintf("%s command=%q allowed=%v sensitive=%v", label, clipOneLine(v.Command, 120), v.Allowed, v.Sensitive)
	case KindAgentUnknown:
		return fmt.Sprintf("%s type=%T", label, e.AgentUI)
	case KindLLMRunCompleted:
		if e.Err != nil {
			return fmt.Sprintf("%s err=%q reply_len=%d", label, clipOneLine(e.Err.Error(), 80), utf8.RuneCountInString(e.Reply))
		}
		return fmt.Sprintf("%s reply_len=%d", label, utf8.RuneCountInString(e.Reply))
	default:
		return string(k)
	}
}

func clipOneLine(s string, maxRunes int) string {
	s = strings.ReplaceAll(s, "\n", " ")
	s = strings.ReplaceAll(s, "\r", "")
	if maxRunes <= 0 {
		return ""
	}
	if utf8.RuneCountInString(s) <= maxRunes {
		return s
	}
	var b strings.Builder
	n := 0
	for _, r := range s {
		if n >= maxRunes-3 {
			b.WriteString("...")
			break
		}
		b.WriteRune(r)
		n++
	}
	return b.String()
}
