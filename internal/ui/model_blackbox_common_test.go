package ui_test

import (
	"os"
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"

	"delve-shell/internal/bootstrap"
	"delve-shell/internal/host/cmd"
	"delve-shell/internal/input/lifecycletype"
	"delve-shell/internal/remote/auth"
	"delve-shell/internal/ui"
	"delve-shell/internal/ui/uivm"
)

func TestMain(m *testing.M) {
	bootstrap.Install()
	os.Exit(m.Run())
}

type blackboxFixture struct {
	model          *ui.Model
	submissions    chan inputlifecycletype.InputSubmission
	sessionNew     chan struct{}
	historyPreview chan string
	sessionSwitch  chan string
	execDirectChan chan string
	shellRequested chan hostcmd.ShellSnapshot
	cancelRequest  chan struct{}
	configUpdated  chan struct{}
	remoteOn       chan string
	remoteOff      chan struct{}
	accessOffline  chan struct{}
	remoteAuthResp chan remoteauth.Response
	openConfigModel  bool
}

type testReadModel struct {
	openConfigModel *bool
	offline       bool
}

func (r testReadModel) TakeOpenConfigModelOnFirstLayout() bool {
	if r.openConfigModel == nil {
		return false
	}
	v := *r.openConfigModel
	*r.openConfigModel = false
	return v
}

func (r testReadModel) OfflineExecutionMode() bool { return r.offline }

func (r testReadModel) InitialRemoteFooter() (active bool, label string, offline bool) {
	return false, "", r.offline
}

type testCommandSender struct {
	f *blackboxFixture
}

func (s testCommandSender) Send(cmd hostcmd.Command) bool {
	switch c := cmd.(type) {
	case hostcmd.Submission:
		select {
		case s.f.submissions <- c.Submission:
			return true
		default:
			return false
		}
	case hostcmd.SessionNew:
		select {
		case s.f.sessionNew <- struct{}{}:
			return true
		default:
			return false
		}
	case hostcmd.HistoryPreviewOpen:
		select {
		case s.f.historyPreview <- c.SessionID:
			return true
		default:
			return false
		}
	case hostcmd.SessionSwitch:
		select {
		case s.f.sessionSwitch <- c.SessionID:
			return true
		default:
			return false
		}
	case hostcmd.ConfigUpdated:
		select {
		case s.f.configUpdated <- struct{}{}:
			return true
		default:
			return false
		}
	case hostcmd.ExecDirect:
		select {
		case s.f.execDirectChan <- c.Command:
			return true
		default:
			return false
		}
	case hostcmd.CancelRequested:
		select {
		case s.f.cancelRequest <- struct{}{}:
			return true
		default:
			return false
		}
	case hostcmd.ShellSnapshot:
		select {
		case s.f.shellRequested <- c:
			return true
		default:
			return false
		}
	case hostcmd.AccessRemote:
		select {
		case s.f.remoteOn <- c.Target:
			return true
		default:
			return false
		}
	case hostcmd.AccessLocal:
		select {
		case s.f.remoteOff <- struct{}{}:
			return true
		default:
			return false
		}
	case hostcmd.AccessOffline:
		select {
		case s.f.accessOffline <- struct{}{}:
			return true
		default:
			return false
		}
	case hostcmd.RemoteAuthReply:
		select {
		case s.f.remoteAuthResp <- c.Response:
			return true
		default:
			return false
		}
	default:
		return true
	}
}

func newBlackboxFixture(t *testing.T) blackboxFixture {
	t.Helper()
	f := blackboxFixture{
		submissions:    make(chan inputlifecycletype.InputSubmission, 2),
		sessionNew:     make(chan struct{}, 2),
		historyPreview: make(chan string, 2),
		sessionSwitch:  make(chan string, 2),
		execDirectChan: make(chan string, 2),
		shellRequested: make(chan hostcmd.ShellSnapshot, 2),
		cancelRequest:  make(chan struct{}, 2),
		configUpdated:  make(chan struct{}, 2),
		remoteOn:       make(chan string, 2),
		remoteOff:      make(chan struct{}, 2),
		accessOffline:  make(chan struct{}, 2),
		remoteAuthResp: make(chan remoteauth.Response, 2),
	}
	f.model = ui.NewModel(nil, testReadModel{openConfigModel: &f.openConfigModel})
	f.model.CommandSender = testCommandSender{f: &f}
	return f
}

func enterText(m *ui.Model, text string) *ui.Model {
	m.Input.SetValue(text)
	m.Input.CursorEnd()
	next, _ := m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	return next.(*ui.Model)
}

func TestBlackboxMainEnterSubmitsUserText(t *testing.T) {
	f := newBlackboxFixture(t)
	if got := f.model.Input.Height(); got != 1 {
		t.Fatalf("expected initial input height 1, got %d", got)
	}
	got := enterText(f.model, "hello world")
	if got.Input.Value() != "" {
		t.Fatalf("expected input cleared after enter, got %q", got.Input.Value())
	}
	transcript := strings.Join(got.TranscriptLines(), "\n")
	if !strings.Contains(transcript, "hello world") {
		t.Fatalf("expected user text to be appended to transcript, got %q", transcript)
	}
}

func TestBlackboxMainCtrlJInsertsNewline(t *testing.T) {
	f := newBlackboxFixture(t)
	m := f.model
	m.Input.SetValue("hello")
	m.Input.CursorEnd()
	next, _ := m.Update(tea.KeyMsg{Type: tea.KeyCtrlJ})
	got := next.(*ui.Model)
	if got.Input.Value() != "hello\n" {
		t.Fatalf("expected ctrl+j to insert newline, got %q", got.Input.Value())
	}
	if got.Input.Height() != 5 {
		t.Fatalf("expected input height to jump to 5 after newline, got %d", got.Input.Height())
	}
	if !strings.Contains(got.Input.View(), "hello") {
		t.Fatalf("first line must stay visible after ctrl+j (viewport scroll), view: %q", got.Input.View())
	}
}

func TestBlackboxAltEnterInsertsNewline(t *testing.T) {
	f := newBlackboxFixture(t)
	m := f.model
	m.Input.SetValue("hello")
	m.Input.CursorEnd()
	next, _ := m.Update(tea.KeyMsg{Type: tea.KeyEnter, Alt: true})
	got := next.(*ui.Model)
	if got.Input.Value() != "hello\n" {
		t.Fatalf("expected alt+enter to insert newline, got %q", got.Input.Value())
	}
	if got.Input.Height() != 5 {
		t.Fatalf("expected input height to jump to 5 after newline, got %d", got.Input.Height())
	}
}

func TestBlackboxSystemErrorClearsProcessingState(t *testing.T) {
	f := newBlackboxFixture(t)
	f.model.Interaction.WaitingForAI = true
	nextModel, _ := f.model.Update(ui.TranscriptAppendMsg{Lines: []uivm.Line{{Kind: uivm.LineSystemError, Text: "backend failed"}}})
	next := nextModel.(*ui.Model)
	if next.Interaction.WaitingForAI {
		t.Fatal("expected system error to clear waiting state")
	}
	if !strings.Contains(strings.Join(next.TranscriptLines(), "\n"), "Delve: Error: backend failed") {
		t.Fatalf("expected submit error to be appended to transcript, got %q", strings.Join(next.TranscriptLines(), "\n"))
	}
}
