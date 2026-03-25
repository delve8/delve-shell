package ui_test

import (
	"os"
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"

	"delve-shell/internal/bootstrap"
	"delve-shell/internal/config"
	"delve-shell/internal/configllm"
	"delve-shell/internal/remote"
	"delve-shell/internal/remoteauth"
	"delve-shell/internal/ui"
	"delve-shell/internal/uivm"
)

func TestMain(m *testing.M) {
	bootstrap.Install()
	os.Exit(m.Run())
}

type blackboxFixture struct {
	model          ui.Model
	submitChan     chan string
	execDirectChan chan string
	shellRequested chan []string
	cancelRequest  chan struct{}
	configUpdated  chan struct{}
	remoteOn       chan string
	remoteOff      chan struct{}
	remoteAuthResp chan remoteauth.Response
	openConfigLLM  bool
}

type testReadModel struct {
	openConfigLLM *bool
}

func (r testReadModel) AllowlistAutoRunEnabled() bool { return true }
func (r testReadModel) TakeOpenConfigLLMOnFirstLayout() bool {
	if r.openConfigLLM == nil {
		return false
	}
	v := *r.openConfigLLM
	*r.openConfigLLM = false
	return v
}

type testActionSender struct {
	f *blackboxFixture
}

func (s testActionSender) Send(a uivm.UIAction) bool {
	switch a.Kind {
	case uivm.UIActionSubmit:
		select {
		case s.f.submitChan <- a.Text:
			return true
		default:
			return false
		}
	case uivm.UIActionConfigUpdated:
		select {
		case s.f.configUpdated <- struct{}{}:
			return true
		default:
			return false
		}
	case uivm.UIActionExecDirect:
		select {
		case s.f.execDirectChan <- a.Text:
			return true
		default:
			return false
		}
	case uivm.UIActionCancelRequested:
		select {
		case s.f.cancelRequest <- struct{}{}:
			return true
		default:
			return false
		}
	case uivm.UIActionShellSnapshot:
		select {
		case s.f.shellRequested <- a.Messages:
			return true
		default:
			return false
		}
	case uivm.UIActionRemoteOnTarget:
		select {
		case s.f.remoteOn <- a.Text:
			return true
		default:
			return false
		}
	case uivm.UIActionRemoteOff:
		select {
		case s.f.remoteOff <- struct{}{}:
			return true
		default:
			return false
		}
	case uivm.UIActionRemoteAuthReply:
		select {
		case s.f.remoteAuthResp <- a.RemoteAuthReply:
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
		submitChan:     make(chan string, 2),
		execDirectChan: make(chan string, 2),
		shellRequested: make(chan []string, 2),
		cancelRequest:  make(chan struct{}, 2),
		configUpdated:  make(chan struct{}, 2),
		remoteOn:       make(chan string, 2),
		remoteOff:      make(chan struct{}, 2),
		remoteAuthResp: make(chan remoteauth.Response, 2),
	}
	f.model = ui.NewModel(nil, testReadModel{openConfigLLM: &f.openConfigLLM})
	f.model.ActionSender = testActionSender{f: &f}
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

func TestBlackboxEscSendsCancelRequest(t *testing.T) {
	f := newBlackboxFixture(t)
	f.model.Interaction.WaitingForAI = true

	next, _ := f.model.Update(tea.KeyMsg{Type: tea.KeyEsc})
	got := next.(ui.Model)
	if got.Interaction.WaitingForAI {
		t.Fatalf("expected waiting flag to be cleared after Esc")
	}
	select {
	case <-f.cancelRequest:
	default:
		t.Fatalf("expected cancel request to be sent")
	}
}

func TestBlackboxEscWhenIdleDoesNothing(t *testing.T) {
	f := newBlackboxFixture(t)
	next, _ := f.model.Update(tea.KeyMsg{Type: tea.KeyEsc})
	got := next.(ui.Model)
	select {
	case <-f.cancelRequest:
		t.Fatalf("did not expect cancel request while idle")
	default:
	}
	if got.Interaction.WaitingForAI {
		t.Fatalf("expected waiting flag to remain false while idle")
	}
}

func TestBlackboxSlashShSendsMessagesToShell(t *testing.T) {
	f := newBlackboxFixture(t)
	f.model = f.model.WithTranscriptLines([]string{"a", "b"})

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

func TestBlackboxSlashRunDropdownUsesRemoteCachedSuggestionsWhenAvailable(t *testing.T) {
	f := newBlackboxFixture(t)

	// Simulate remote on and a cached /run suggestion list from host.
	next, _ := f.model.Update(remote.ExecutionChangedMsg{Active: true, Label: "r1"})
	m1 := next.(ui.Model)
	next2, _ := m1.Update(remote.RunCompletionCacheMsg{RemoteLabel: "r1", Commands: []string{"busybox", "bzip2"}})
	m2 := next2.(ui.Model)

	m2.Input.SetValue("/run b")
	m2.Input.CursorEnd()
	view := m2.View()
	if !strings.Contains(view, "/run busybox") || !strings.Contains(view, "/run bzip2") {
		t.Fatalf("expected remote cached /run suggestions in dropdown, got view:\n%s", view)
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
	joined := strings.Join(got.TranscriptLines(), "\n")
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

func TestBlackboxEscClearsSlashInputBeforeCancelling(t *testing.T) {
	f := newBlackboxFixture(t)
	f.model.Interaction.WaitingForAI = true

	m := f.model
	m.Input.SetValue("/remote")
	m.Input.CursorEnd()

	next, _ := m.Update(tea.KeyMsg{Type: tea.KeyEsc})
	m2 := next.(ui.Model)
	if strings.TrimSpace(m2.Input.Value()) != "" {
		t.Fatalf("expected Esc to clear slash input first, got %q", m2.Input.Value())
	}
	if !m2.Interaction.WaitingForAI {
		t.Fatalf("expected waiting flag to remain true after clearing slash input")
	}
	select {
	case <-f.cancelRequest:
		t.Fatalf("did not expect cancel signal while clearing slash input")
	default:
	}

	next2, _ := m2.Update(tea.KeyMsg{Type: tea.KeyEsc})
	m3 := next2.(ui.Model)
	if m3.Interaction.WaitingForAI {
		t.Fatalf("expected waiting flag false after second Esc")
	}
	select {
	case <-f.cancelRequest:
	default:
		t.Fatalf("expected cancel signal on second Esc")
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
	got := enterText(f.model, "/sessions demo")
	select {
	case cmd := <-f.submitChan:
		if cmd != "/sessions demo" {
			t.Fatalf("expected /sessions command submit, got %q", cmd)
		}
	default:
		t.Fatalf("expected /sessions <id> to submit command")
	}
	if strings.TrimSpace(got.Input.Value()) != "" {
		t.Fatalf("expected input cleared after prefix slash execution, got %q", got.Input.Value())
	}
}

func TestBlackboxStartupOverlayProviderOpensConfigLLM(t *testing.T) {
	open := true
	m := ui.NewModel(nil, testReadModel{openConfigLLM: &open})
	next, _ := m.Update(tea.WindowSizeMsg{Width: 120, Height: 40})
	got := next.(ui.Model)
	if !got.Overlay.Active || !configllm.OverlayActive() {
		t.Fatalf("expected startup overlay provider to open config llm overlay")
	}
}
