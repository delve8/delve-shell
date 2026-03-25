package ui_test

import (
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"

	"delve-shell/internal/config"
	"delve-shell/internal/configllm"
	"delve-shell/internal/hostapp"
	_ "delve-shell/internal/remote"
	_ "delve-shell/internal/run"
	_ "delve-shell/internal/session"
	_ "delve-shell/internal/skill"
	"delve-shell/internal/ui"
)

type blackboxFixture struct {
	model          ui.Model
	submitChan     chan string
	execDirectChan chan string
	shellRequested chan []string
	cancelRequest  chan struct{}
	configUpdated  chan struct{}
	remoteOn       chan string
	remoteOff      chan struct{}
	remoteAuthResp chan ui.RemoteAuthResponse
}

func newBlackboxFixture(t *testing.T) blackboxFixture {
	t.Helper()
	f := blackboxFixture{
		submitChan:     make(chan string, 2),
		execDirectChan: make(chan string, 2),
		shellRequested: make(chan []string, 2),
		cancelRequest:  make(chan struct{}, 2),
		configUpdated:  make(chan struct{}, 2),
		remoteOn:       make(chan string, 2),
		remoteOff:      make(chan struct{}, 2),
		remoteAuthResp: make(chan ui.RemoteAuthResponse, 2),
	}
	hostapp.Wire(&hostapp.Send{
		Submit:         f.submitChan,
		ConfigUpdated:  f.configUpdated,
		CancelRequest:  f.cancelRequest,
		ExecDirect:     f.execDirectChan,
		RemoteOn:       f.remoteOn,
		RemoteOff:      f.remoteOff,
		RemoteAuthResp: f.remoteAuthResp,
		ShellSnapshot:  f.shellRequested,
	})
	t.Cleanup(hostapp.ResetTestState)
	hostapp.BindAllowlistAutoRun(func() bool { return true }, func(bool) {})
	hostapp.SetRemoteExecution(false, "")
	hostapp.SetOpenConfigLLMOnFirstLayout(false)
	f.model = ui.NewModel(nil)
	return f
}

func enterText(m ui.Model, text string) ui.Model {
	m.Input.SetValue(text)
	m.Input.CursorEnd()
	next, _ := m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	return next.(ui.Model)
}

func TestBlackboxSlashHelpOpensOverlay(t *testing.T) {
	f := newBlackboxFixture(t)
	got := enterText(f.model, "/help")
	if !got.Overlay.Active {
		t.Fatalf("expected /help to open overlay")
	}
	if got.Overlay.Title == "" {
		t.Fatalf("expected /help overlay title to be non-empty")
	}
}

func TestBlackboxSlashRemoteOnOpensOverlay(t *testing.T) {
	f := newBlackboxFixture(t)
	got := enterText(f.model, "/remote on")
	if !got.Overlay.Active {
		t.Fatalf("expected /remote on to open add-remote overlay")
	}
}

func TestBlackboxOverlayEscRunsFeatureResetHook(t *testing.T) {
	f := newBlackboxFixture(t)
	m := enterText(f.model, "/remote on")
	if !m.Overlay.Active {
		t.Fatalf("precondition failed: add-remote overlay should be active")
	}
	next, _ := m.Update(tea.KeyMsg{Type: tea.KeyEsc})
	got := next.(ui.Model)
	if got.Overlay.Active {
		t.Fatalf("expected esc to close overlay and reset feature state")
	}
}

func TestBlackboxSlashCancelSendsCancelRequest(t *testing.T) {
	f := newBlackboxFixture(t)
	f.model.Interaction.WaitingForAI = true

	got := enterText(f.model, "/cancel")
	if got.Interaction.WaitingForAI {
		t.Fatalf("expected waiting flag to be cleared after /cancel")
	}
	if strings.TrimSpace(got.Input.Value()) != "" {
		t.Fatalf("expected input cleared after /cancel, got %q", got.Input.Value())
	}
	select {
	case <-f.cancelRequest:
	default:
		t.Fatalf("expected cancel request to be sent")
	}
}

func TestBlackboxSlashCancelWhenIdleShowsHint(t *testing.T) {
	f := newBlackboxFixture(t)
	got := enterText(f.model, "/cancel")
	if strings.TrimSpace(got.Input.Value()) != "" {
		t.Fatalf("expected input cleared after idle /cancel, got %q", got.Input.Value())
	}
	if len(got.Messages) == 0 {
		t.Fatalf("expected feedback message when /cancel has no in-flight request")
	}
	last := strings.Join(got.Messages, "\n")
	if !strings.Contains(strings.ToLower(last), "no request") {
		t.Fatalf("expected no-request hint, got %q", last)
	}
}

func TestBlackboxSlashShSendsMessagesToShell(t *testing.T) {
	f := newBlackboxFixture(t)
	f.model.Messages = []string{"a", "b"}

	_ = enterText(f.model, "/sh")
	select {
	case msgs := <-f.shellRequested:
		if len(msgs) < 2 || msgs[0] != "a" || msgs[1] != "b" {
			t.Fatalf("unexpected shell message snapshot prefix: %#v", msgs)
		}
		joined := strings.Join(msgs, "\n")
		if !strings.Contains(joined, "User: /sh") {
			t.Fatalf("expected User echo for /sh in snapshot, got %#v", msgs)
		}
	default:
		t.Fatalf("expected /sh to send message snapshot")
	}
}

func TestBlackboxSlashRunExecutesDirectCommand(t *testing.T) {
	f := newBlackboxFixture(t)
	_ = enterText(f.model, "/run echo")
	select {
	case cmd := <-f.execDirectChan:
		if cmd != "echo" {
			t.Fatalf("expected exec cmd 'echo', got %q", cmd)
		}
	default:
		t.Fatalf("expected /run to send command to execDirectChan")
	}
}

func TestBlackboxSlashRunUsageFillsInput(t *testing.T) {
	f := newBlackboxFixture(t)
	got := enterText(f.model, "/run")
	if got.Input.Value() != "/run " {
		t.Fatalf("expected /run to fill input to '/run ', got %q", got.Input.Value())
	}
}

func TestBlackboxSlashConfigDelRemoteNoHostsShowsHint(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("DELVE_SHELL_ROOT", dir)
	if err := config.EnsureRootDir(); err != nil {
		t.Fatal(err)
	}
	f := newBlackboxFixture(t)
	got := enterText(f.model, "/config del-remote")
	if strings.TrimSpace(got.Input.Value()) != "" {
		t.Fatalf("expected input cleared after no-hosts del-remote, got %q", got.Input.Value())
	}
	joined := strings.Join(got.Messages, "\n")
	if !strings.Contains(joined, "No hosts") {
		t.Fatalf("expected no-hosts hint in transcript, got %q", joined)
	}
}

func TestBlackboxSlashConfigFillsToFirstSubcommandOnEnter(t *testing.T) {
	f := newBlackboxFixture(t)
	got := enterText(f.model, "/config")
	if got.Input.Value() != "/config add-remote" {
		t.Fatalf("expected /config to fill to first subcommand, got %q", got.Input.Value())
	}
}

func TestBlackboxSlashDropdownUpDownAndEnterFill(t *testing.T) {
	f := newBlackboxFixture(t)
	m := f.model
	m.Input.SetValue("/")
	m.Input.CursorEnd()

	next, _ := m.Update(tea.KeyMsg{Type: tea.KeyDown})
	m2 := next.(ui.Model)
	if m2.Input.Value() != "/" {
		t.Fatalf("expected input to remain '/', got %q", m2.Input.Value())
	}

	next2, _ := m2.Update(tea.KeyMsg{Type: tea.KeyEnter})
	m3 := next2.(ui.Model)
	if strings.TrimSpace(m3.Input.Value()) == "/" {
		t.Fatalf("expected enter to fill a concrete slash option, got %q", m3.Input.Value())
	}
	if v := strings.TrimSpace(m3.Input.Value()); v != "" && !strings.HasPrefix(v, "/") {
		t.Fatalf("expected filled value to start with '/', got %q", m3.Input.Value())
	}
}

func TestBlackboxSlashDropdownCancelFillThenExecute(t *testing.T) {
	f := newBlackboxFixture(t)
	f.model.Interaction.WaitingForAI = true

	m := f.model
	m.Input.SetValue("/c")
	m.Input.CursorEnd()

	// First Enter should fill to /cancel (not execute yet).
	next, _ := m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	m2 := next.(ui.Model)
	if strings.TrimSpace(m2.Input.Value()) != "/cancel" {
		t.Fatalf("expected first enter to fill /cancel, got %q", m2.Input.Value())
	}
	if !m2.Interaction.WaitingForAI {
		t.Fatalf("expected waiting flag to remain true after fill-only enter")
	}
	select {
	case <-f.cancelRequest:
		t.Fatalf("did not expect cancel signal on fill-only enter")
	default:
	}

	// Second Enter executes /cancel.
	next2, _ := m2.Update(tea.KeyMsg{Type: tea.KeyEnter})
	m3 := next2.(ui.Model)
	if m3.Interaction.WaitingForAI {
		t.Fatalf("expected waiting flag false after executing /cancel")
	}
	if strings.TrimSpace(m3.Input.Value()) != "" {
		t.Fatalf("expected input cleared after executing /cancel, got %q", m3.Input.Value())
	}
	select {
	case <-f.cancelRequest:
	default:
		t.Fatalf("expected cancel signal on second enter")
	}
}

func TestBlackboxSlashUpdateSkillEnterDoesNotSilentlyDrop(t *testing.T) {
	f := newBlackboxFixture(t)
	m := f.model
	m.Input.SetValue("/config update-skill x")
	m.Input.CursorEnd()

	next, _ := m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	got := next.(ui.Model)
	if got.Input.Value() == "" && !got.Overlay.Active {
		t.Fatalf("expected either overlay opened or non-empty input after enter")
	}
}

func TestBlackboxSlashNewSubmitsCommand(t *testing.T) {
	f := newBlackboxFixture(t)
	got := enterText(f.model, "/new")
	select {
	case v := <-f.submitChan:
		if v != "/new" {
			t.Fatalf("expected submit '/new', got %q", v)
		}
	default:
		t.Fatalf("expected /new to submit command")
	}
	if got.Input.Value() != "" {
		t.Fatalf("expected input to be cleared after /new, got %q", got.Input.Value())
	}
}

func TestBlackboxSlashSessionsPrefixSubmitsCommand(t *testing.T) {
	f := newBlackboxFixture(t)
	_ = enterText(f.model, "/sessions demo")
	select {
	case cmd := <-f.submitChan:
		if cmd != "/sessions demo" {
			t.Fatalf("expected /sessions command submit, got %q", cmd)
		}
	default:
		t.Fatalf("expected /sessions <id> to submit command")
	}
}

func TestBlackboxStartupOverlayProviderOpensConfigLLM(t *testing.T) {
	t.Cleanup(func() { hostapp.SetOpenConfigLLMOnFirstLayout(false) })
	hostapp.SetOpenConfigLLMOnFirstLayout(true)
	m := ui.NewModel(nil)
	next, _ := m.Update(tea.WindowSizeMsg{Width: 120, Height: 40})
	got := next.(ui.Model)
	if !got.Overlay.Active || !configllm.OverlayActive() {
		t.Fatalf("expected startup overlay provider to open config llm overlay")
	}
}
