package ui_test

import (
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"

	"delve-shell/internal/config"
	"delve-shell/internal/host/cmd"
	"delve-shell/internal/i18n"
	"delve-shell/internal/remote"
	"delve-shell/internal/ui"
)

func TestBlackboxSlashHelpOpensOverlay(t *testing.T) {
	f := newBlackboxFixture(t)
	got := enterText(f.model, "/help")
	if !got.Overlay.Active {
		t.Fatalf("expected /help to open overlay")
	}
	if got.Overlay.Title == "" {
		t.Fatalf("expected /help overlay title to be non-empty")
	}
	if !strings.Contains(got.Overlay.Viewport.View(), "dev") {
		t.Fatalf("expected /help overlay content to include default version, got %q", got.Overlay.Viewport.View())
	}
	transcript := strings.Join(got.TranscriptLines(), "\n")
	if !strings.Contains(transcript, "/help") {
		t.Fatalf("expected user echo for /help, got %q", transcript)
	}
}

func TestBlackboxSlashBashSendsMessagesToShell(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("/bash is not available on Windows")
	}
	f := newBlackboxFixture(t)
	f.model.WithTranscriptLines([]string{"a", "b"})

	_ = enterText(f.model, "/bash")
	select {
	case snap := <-f.shellRequested:
		msgs := snap.Messages
		if len(msgs) < 2 || msgs[0] != "a" || msgs[1] != "b" {
			t.Fatalf("unexpected shell message snapshot prefix: %#v", msgs)
		}
		if snap.Mode != hostcmd.SubshellModeLocalBash {
			t.Fatalf("expected local bash subshell mode, got %v", snap.Mode)
		}
		joined := strings.Join(msgs, "\n")
		if !strings.Contains(joined, "> /bash") {
			t.Fatalf("expected user echo for /bash in snapshot, got %#v", msgs)
		}
	default:
		t.Fatalf("expected /bash to send message snapshot")
	}
}

func TestBlackboxSlashBashSnapshotIncludesInputHistory(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("/bash is not available on Windows")
	}
	f := newBlackboxFixture(t)
	m := enterText(f.model, "hello")
	m.Interaction.WaitingForAI = false
	_ = enterText(m, "/bash")
	select {
	case snap := <-f.shellRequested:
		if len(snap.InputHistory) != 2 || snap.InputHistory[0] != "hello" || snap.InputHistory[1] != "/bash" {
			t.Fatalf("want input history [hello /bash], got %#v", snap.InputHistory)
		}
	default:
		t.Fatalf("expected /bash to send shell snapshot")
	}
}

func TestBlackboxSlashBashRemoteModeWhenRemoteActive(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("/bash is not available on Windows")
	}
	f := newBlackboxFixture(t)
	next, _ := f.model.Update(remote.ExecutionChangedMsg{Active: true, Label: "r1"})
	m := next.(*ui.Model)
	_ = enterText(m, "/bash")
	select {
	case snap := <-f.shellRequested:
		if snap.Mode != hostcmd.SubshellModeRemoteSSH {
			t.Fatalf("expected remote SSH subshell mode, got %v", snap.Mode)
		}
	default:
		t.Fatalf("expected /bash snapshot")
	}
}

func TestBlackboxSlashRunExecutesDirectCommand(t *testing.T) {
	f := newBlackboxFixture(t)
	_ = enterText(f.model, "/exec echo")
	select {
	case cmd := <-f.execDirectChan:
		if cmd != "echo" {
			t.Fatalf("expected exec cmd 'echo', got %q", cmd)
		}
	default:
		t.Fatalf("expected /exec to send command to execDirectChan")
	}
}

func TestBlackboxSlashRunUsageFillsInput(t *testing.T) {
	f := newBlackboxFixture(t)
	got := enterText(f.model, "/exec")
	if got.Input.Value() != "/exec " {
		t.Fatalf("expected /exec to fill input to '/exec ', got %q", got.Input.Value())
	}
}

func TestBlackboxSlashRunDropdownUsesRemoteCachedSuggestionsWhenAvailable(t *testing.T) {
	f := newBlackboxFixture(t)

	next, _ := f.model.Update(remote.ExecutionChangedMsg{Active: true, Label: "r1"})
	m1 := next.(*ui.Model)
	next2, _ := m1.Update(remote.RunCompletionCacheMsg{RemoteLabel: "r1", Commands: []string{"busybox", "bzip2"}})
	m2 := next2.(*ui.Model)

	m2.Input.SetValue("/exec b")
	m2.Input.CursorEnd()
	view := m2.View()
	if !strings.Contains(view, "/exec busybox") || !strings.Contains(view, "/exec bzip2") {
		t.Fatalf("expected remote cached /exec suggestions in dropdown, got view:\n%s", view)
	}
}

func TestBlackboxSlashConfigRemoveRemoteNoHostsShowsHint(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("DELVE_SHELL_ROOT", dir)
	if err := config.EnsureRootDir(); err != nil {
		t.Fatal(err)
	}
	f := newBlackboxFixture(t)
	got := enterText(f.model, "/config remove-remote")
	if strings.TrimSpace(got.Input.Value()) != "" {
		t.Fatalf("expected input cleared after no-hosts remove-remote, got %q", got.Input.Value())
	}
	joined := strings.Join(got.TranscriptLines(), "\n")
	if !strings.Contains(joined, "No hosts") {
		t.Fatalf("expected no-hosts hint in transcript, got %q", joined)
	}
}

func TestBlackboxSlashConfigFillsToFirstSubcommandOnEnter(t *testing.T) {
	f := newBlackboxFixture(t)
	got := enterText(f.model, "/config")
	if got.Input.Value() != "/config remove-remote " {
		t.Fatalf("expected /config to fill to first subcommand with trailing space, got %q", got.Input.Value())
	}
}

func TestBlackboxSlashDropdownUpDownAndEnterFill(t *testing.T) {
	f := newBlackboxFixture(t)
	m := f.model
	m.Input.SetValue("/")
	m.Input.CursorEnd()

	next, _ := m.Update(tea.KeyMsg{Type: tea.KeyDown})
	m2 := next.(*ui.Model)
	if m2.Input.Value() != "/" {
		t.Fatalf("expected input to remain '/', got %q", m2.Input.Value())
	}

	next2, _ := m2.Update(tea.KeyMsg{Type: tea.KeyEnter})
	m3 := next2.(*ui.Model)
	if strings.TrimSpace(m3.Input.Value()) == "/" {
		t.Fatalf("expected enter to fill a concrete slash option, got %q", m3.Input.Value())
	}
	if v := strings.TrimSpace(m3.Input.Value()); v != "" && !strings.HasPrefix(v, "/") {
		t.Fatalf("expected filled value to start with '/', got %q", m3.Input.Value())
	}
}

func TestBlackboxSlashDropdownTabFillsLikeEnter(t *testing.T) {
	f := newBlackboxFixture(t)
	m := f.model
	m.Input.SetValue("/")
	m.Input.CursorEnd()

	next, _ := m.Update(tea.KeyMsg{Type: tea.KeyDown})
	m2 := next.(*ui.Model)
	next2, _ := m2.Update(tea.KeyMsg{Type: tea.KeyTab})
	m3 := next2.(*ui.Model)
	if strings.TrimSpace(m3.Input.Value()) == "/" {
		t.Fatalf("expected tab to fill a concrete slash option, got %q", m3.Input.Value())
	}
	if v := strings.TrimSpace(m3.Input.Value()); v != "" && !strings.HasPrefix(v, "/") {
		t.Fatalf("expected filled value to start with '/', got %q", m3.Input.Value())
	}
}

func TestBlackboxSlashLockedDuringProcessing(t *testing.T) {
	f := newBlackboxFixture(t)
	m := f.model
	m.Interaction.WaitingForAI = true
	m.Input.SetValue("/help")
	m.Input.CursorEnd()

	next, _ := m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	got := next.(*ui.Model)
	if got.Overlay.Active {
		t.Fatal("expected /help not to open overlay while processing")
	}
	if got.Input.Value() != "/help" {
		t.Fatalf("expected busy processing to keep slash input unchanged, got %q", got.Input.Value())
	}
	if strings.Contains(strings.Join(got.TranscriptLines(), "\n"), "/help") {
		t.Fatalf("expected busy processing to suppress slash submission, got transcript %q", strings.Join(got.TranscriptLines(), "\n"))
	}
	if strings.Contains(got.View(), "/help -") {
		t.Fatalf("expected busy processing to hide slash dropdown, got view:\n%s", got.View())
	}
}

func TestBlackboxSlashLockedDuringCommandExecution(t *testing.T) {
	f := newBlackboxFixture(t)
	m := f.model
	m.Interaction.CommandExecuting = true
	m.Input.SetValue("/help")
	m.Input.CursorEnd()

	next, _ := m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	got := next.(*ui.Model)
	if got.Overlay.Active {
		t.Fatal("expected /help not to open overlay while command is executing")
	}
	if got.Input.Value() != "/help" {
		t.Fatalf("expected executing state to keep slash input unchanged, got %q", got.Input.Value())
	}
	if strings.Contains(strings.Join(got.TranscriptLines(), "\n"), "/help") {
		t.Fatalf("expected executing state to suppress slash submission, got transcript %q", strings.Join(got.TranscriptLines(), "\n"))
	}
}

func TestBlackboxSlashTabDoesNotSubmitExactCommand(t *testing.T) {
	f := newBlackboxFixture(t)
	m := f.model
	m.Input.SetValue("/help")
	m.Input.CursorEnd()
	next, _ := m.Update(tea.KeyMsg{Type: tea.KeyTab})
	got := next.(*ui.Model)
	if got.Overlay.Active {
		t.Fatalf("expected tab not to submit /help (no overlay)")
	}
	if strings.TrimSpace(got.Input.Value()) != "/help" {
		t.Fatalf("expected input unchanged, got %q", got.Input.Value())
	}
}

func TestBlackboxSlashAccessOfflineSendsIntent(t *testing.T) {
	f := newBlackboxFixture(t)
	_ = enterText(f.model, "/access Offline")
	select {
	case <-f.accessOffline:
	default:
		t.Fatal("expected /access Offline to emit AccessOffline intent")
	}
}

func TestBlackboxSlashAccessLocalSendsIntent(t *testing.T) {
	f := newBlackboxFixture(t)
	got := enterText(f.model, "/access Local")
	select {
	case <-f.remoteOff:
	default:
		t.Fatal("expected /access Local to emit AccessLocal intent")
	}
	transcript := strings.Join(got.TranscriptLines(), "\n")
	if !strings.Contains(transcript, "/access Local") {
		t.Fatalf("expected user echo for /access Local, got %q", transcript)
	}
}

func TestBlackboxSlashAccessRemoteHostOpensOverlayAndSendsIntent(t *testing.T) {
	f := newBlackboxFixture(t)
	got := enterText(f.model, "/access root@example.com")
	if !got.Overlay.Active {
		t.Fatal("expected /access remote to open connecting overlay")
	}
	if got.Overlay.Title != "Connect Remote" {
		t.Fatalf("expected connect overlay title, got %q", got.Overlay.Title)
	}
	if strings.Contains(got.View(), "Add remote") {
		t.Fatalf("did not expect add-remote content in connect overlay, got %q", got.View())
	}
	select {
	case target := <-f.remoteOn:
		if target != "root@example.com" {
			t.Fatalf("expected /access target, got %q", target)
		}
	default:
		t.Fatal("expected /access remote to emit AccessRemote intent")
	}
	transcript := strings.Join(got.TranscriptLines(), "\n")
	if !strings.Contains(transcript, "/access root@example.com") {
		t.Fatalf("expected user echo for /access prod, got %q", transcript)
	}
}

func TestBlackboxSlashAccessRemoteWithoutUserOpensOverlayOnly(t *testing.T) {
	t.Setenv("DELVE_SHELL_ROOT", t.TempDir())
	if err := config.EnsureRootDir(); err != nil {
		t.Fatal(err)
	}
	f := newBlackboxFixture(t)
	got := enterText(f.model, "/access prod")
	if !got.Overlay.Active {
		t.Fatal("expected /access host to open overlay")
	}
	if got.Overlay.Title != "Connect Remote" {
		t.Fatalf("expected connect overlay title, got %q", got.Overlay.Title)
	}
	select {
	case target := <-f.remoteOn:
		if target != "prod" {
			t.Fatalf("expected AccessRemote target prod, got %q", target)
		}
	default:
		t.Fatal("expected /access host to emit AccessRemote intent")
	}
}

func TestBlackboxSlashConfigRemoveRemoteAppendsImmediateSuccess(t *testing.T) {
	i18n.SetLang("en")
	dir := t.TempDir()
	t.Setenv("DELVE_SHELL_ROOT", dir)
	if err := config.EnsureRootDir(); err != nil {
		t.Fatal(err)
	}
	if err := config.AddRemote("ops@prod", "Production", ""); err != nil {
		t.Fatal(err)
	}

	f := newBlackboxFixture(t)
	got := enterText(f.model, "/config remove-remote prod")
	transcript := strings.Join(got.TranscriptLines(), "\n")
	want := i18n.Tf(i18n.KeyConfigRemoteRemoved, "prod")
	if !strings.Contains(transcript, want) {
		t.Fatalf("expected remove-remote success in transcript, got %q", transcript)
	}
	remotes, err := config.LoadRemotes()
	if err != nil {
		t.Fatal(err)
	}
	if len(remotes) != 0 {
		t.Fatalf("expected remote removed, got %#v", remotes)
	}
}

func TestBlackboxSlashSkillRemoveAppendsImmediateSuccess(t *testing.T) {
	i18n.SetLang("en")
	dir := t.TempDir()
	t.Setenv("DELVE_SHELL_ROOT", dir)
	if err := config.EnsureRootDir(); err != nil {
		t.Fatal(err)
	}
	skillDir := filepath.Join(config.SkillsDir(), "demo")
	if err := os.MkdirAll(skillDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(skillDir, "SKILL.md"), []byte("# demo"), 0o644); err != nil {
		t.Fatal(err)
	}

	f := newBlackboxFixture(t)
	got := enterText(f.model, "/skill Remove demo")
	transcript := strings.Join(got.TranscriptLines(), "\n")
	want := i18n.Tf(i18n.KeySkillRemoved, "demo")
	if !strings.Contains(transcript, want) {
		t.Fatalf("expected remove-skill success in transcript, got %q", transcript)
	}
	if _, err := os.Stat(skillDir); !os.IsNotExist(err) {
		t.Fatalf("expected skill dir removed, stat err=%v", err)
	}
}

func TestBlackboxSlashBashOfflineAppendsErrorToTranscript(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("/bash is not available on Windows")
	}
	f := newBlackboxFixture(t)
	f.model = ui.NewModel(nil, testReadModel{openConfigModel: &f.openConfigModel, offline: true})
	f.model.CommandSender = testCommandSender{f: &f}
	got := enterText(f.model, "/bash")
	transcript := strings.Join(got.TranscriptLines(), "\n")
	if !strings.Contains(transcript, "/bash") {
		t.Fatalf("expected user echo for /bash, got %q", transcript)
	}
	if !strings.Contains(transcript, "Offline") {
		t.Fatalf("expected offline /bash error in transcript, got %q", transcript)
	}
}

func TestBlackboxSlashSkillOfflineShowsError(t *testing.T) {
	f := newBlackboxFixture(t)
	f.model = ui.NewModel(nil, testReadModel{openConfigModel: &f.openConfigModel, offline: true})
	f.model.CommandSender = testCommandSender{f: &f}
	got := enterText(f.model, "/skill someskill extra")
	transcript := strings.Join(got.TranscriptLines(), "\n")
	if !strings.Contains(transcript, "/skill") {
		t.Fatalf("expected user echo for /skill, got %q", transcript)
	}
	if !strings.Contains(transcript, "Offline") {
		t.Fatalf("expected offline /skill error in transcript, got %q", transcript)
	}
	select {
	case sub := <-f.submissions:
		t.Fatalf("did not expect chat submission in offline /skill, got %#v", sub)
	default:
	}
}

func TestBlackboxSlashSkillNewOpensAddSkillOverlay(t *testing.T) {
	f := newBlackboxFixture(t)
	got := enterText(f.model, "/skill New")
	if !got.Overlay.Active {
		t.Fatalf("expected /skill New to open add-skill overlay")
	}
	if got.Overlay.Key == "" {
		t.Fatalf("expected add-skill overlay key to be set")
	}
}
