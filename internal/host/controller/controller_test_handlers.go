package controller

import (
	"context"
	"errors"
	"io"
	"os"
	"path/filepath"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	tea "github.com/charmbracelet/bubbletea"

	"delve-shell/internal/config"
	"delve-shell/internal/hil/types"
	"delve-shell/internal/history"
	"delve-shell/internal/i18n"
	"delve-shell/internal/remote/execenv"
	"delve-shell/internal/runtime/executormgr"
	"delve-shell/internal/runtime/runnermgr"
	"delve-shell/internal/ui"
	"delve-shell/internal/ui/uivm"
)

func waitUntil(t *testing.T, pred func() bool, d time.Duration) {
	t.Helper()
	deadline := time.Now().Add(d)
	for time.Now().Before(deadline) {
		if pred() {
			return
		}
		time.Sleep(2 * time.Millisecond)
	}
	t.Fatal("timeout waiting for condition")
}

func writeControllerTestSSHConfig(t *testing.T, home string, content string) {
	t.Helper()
	sshDir := filepath.Join(home, ".ssh")
	if err := os.MkdirAll(sshDir, 0700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(sshDir, "config"), []byte(content), 0600); err != nil {
		t.Fatal(err)
	}
}

func TestHandleCancelRequest_NoRunning(t *testing.T) {
	s := &recordSender{}
	c := newTestControllerWithPresenter(s)
	c.handleCancelRequest()
	if len(s.msgs) != 0 {
		t.Fatalf("unexpected msgs: %+v", s.msgs)
	}
}

func TestHandleCancelRequest_RunningCancels(t *testing.T) {
	s := &recordSender{}
	c := newTestControllerWithPresenter(s)

	var called atomic.Bool
	c.llmRunning = true
	c.llmCancel = func() { called.Store(true) }
	c.handleCancelRequest()

	if !called.Load() {
		t.Fatal("cancel func was not called")
	}
}

func TestHandleExecDirect_StdoutOnly(t *testing.T) {
	s := &recordSender{}
	c := newTestControllerWithPresenter(s)
	fx := &fakeExec{
		stdout:   "ok",
		stderr:   "",
		exitCode: 0,
	}
	c.getExec = func() execenv.CommandExecutor { return fx }

	c.handleExecDirect("echo ok")
	waitUntil(t, func() bool { return fx.lastCmd != "" }, time.Second)
	if fx.lastCmd != "echo ok" {
		t.Fatalf("unexpected command: %q", fx.lastCmd)
	}
	var msg ui.TranscriptAppendMsg
	var ok bool
	for i := len(s.msgs) - 1; i >= 0; i-- {
		msg, ok = s.msgs[i].(ui.TranscriptAppendMsg)
		if ok {
			break
		}
	}
	if !ok {
		t.Fatalf("no TranscriptAppendMsg in %d msgs", len(s.msgs))
	}
	_ = msg
}

func TestHandleExecDirect_StdoutAndStderr(t *testing.T) {
	s := &recordSender{}
	c := newTestControllerWithPresenter(s)
	fx := &fakeExec{
		stdout:   "hello",
		stderr:   "warn",
		exitCode: 1,
		err:      errors.New("exit status 1"),
	}
	c.getExec = func() execenv.CommandExecutor { return fx }

	c.handleExecDirect("bad")
	waitUntil(t, func() bool { return fx.lastCmd != "" }, time.Second)
	found := false
	for i := len(s.msgs) - 1; i >= 0; i-- {
		if _, ok := s.msgs[i].(ui.TranscriptAppendMsg); ok {
			found = true
			break
		}
	}
	if !found {
		t.Fatalf("no TranscriptAppendMsg")
	}
}

func TestHandleExecDirect_RunErrWithoutExitCodeAddsErrorLine(t *testing.T) {
	s := &recordSender{}
	c := newTestControllerWithPresenter(s)
	fx := &fakeExec{
		stdout:   "",
		stderr:   "",
		exitCode: 0,
		err:      errors.New("network issue"),
	}
	c.getExec = func() execenv.CommandExecutor { return fx }

	c.handleExecDirect("x")
	waitUntil(t, func() bool { return fx.lastCmd != "" }, time.Second)
	found := false
	for i := len(s.msgs) - 1; i >= 0; i-- {
		if _, ok := s.msgs[i].(ui.TranscriptAppendMsg); ok {
			found = true
			break
		}
	}
	if !found {
		t.Fatalf("no TranscriptAppendMsg")
	}
}

// fakeStreamExec implements [execenv.StreamingRunner] for /exec streaming path tests.
type fakeStreamExec struct {
	lastCmd string
}

func (f *fakeStreamExec) Run(context.Context, string) (string, string, int, error) {
	return "", "", 0, errors.New("unexpected Run")
}

func (f *fakeStreamExec) RunStreaming(ctx context.Context, command string, stdout, stderr io.Writer) (exitCode int, err error) {
	_ = ctx
	f.lastCmd = command // same field as fakeExec for wait helpers
	_, _ = stdout.Write([]byte("out1\n"))
	_, _ = stderr.Write([]byte("err1\n"))
	return 0, nil
}

func TestHandleExecDirect_StreamingRunner(t *testing.T) {
	i18n.SetLang("en")
	s := &recordSender{}
	c := newTestControllerWithPresenter(s)
	fx := &fakeStreamExec{}
	c.getExec = func() execenv.CommandExecutor { return fx }

	c.handleExecDirect("echo hi")
	waitUntil(t, func() bool { return fx.lastCmd != "" }, time.Second)
	if fx.lastCmd != "echo hi" {
		t.Fatalf("cmd: %q", fx.lastCmd)
	}
	found := false
	for _, msg := range s.msgs {
		ta, ok := msg.(ui.TranscriptAppendMsg)
		if !ok || len(ta.Lines) != 1 {
			continue
		}
		if ta.Lines[0].Text == "Run (direct): echo hi" {
			found = true
			break
		}
	}
	if !found {
		t.Fatalf("no Run line in msgs: %d", len(s.msgs))
	}
	if len(s.msgs) < 4 {
		t.Fatalf("want several msgs (exec UI + stream), got %d", len(s.msgs))
	}
}

func TestHandleAccessLocal_AppendsSystemNotify(t *testing.T) {
	s := &recordSender{}
	c := newTestControllerWithPresenter(s)
	c.executors = executormgr.New()

	c.handleAccessLocal()

	msg := latestTranscriptAppendMsg(t, s.msgs)
	if len(msg.Lines) < 2 {
		t.Fatalf("want notify + blank, got %#v", msg.Lines)
	}
	if msg.Lines[0].Kind != uivm.LineSystemSuggest || msg.Lines[0].Text != "Switched back to local executor." {
		t.Fatalf("unexpected local notify: %#v", msg.Lines)
	}
	if msg.Lines[1].Kind != uivm.LineBlank {
		t.Fatalf("want trailing blank, got %#v", msg.Lines)
	}
}

func TestHandleAccessOffline_AppendsSystemNotify(t *testing.T) {
	s := &recordSender{}
	c := newTestControllerWithPresenter(s)
	c.executors = executormgr.New()

	c.handleAccessOffline()

	msg := latestTranscriptAppendMsg(t, s.msgs)
	if len(msg.Lines) < 2 {
		t.Fatalf("want notify + blank, got %#v", msg.Lines)
	}
	if msg.Lines[0].Kind != uivm.LineSystemSuggest {
		t.Fatalf("unexpected offline notify kind: %#v", msg.Lines)
	}
	if msg.Lines[0].Text != "Offline mode: commands are shown only, not executed here. Paste the results back and review them before running them elsewhere." {
		t.Fatalf("unexpected offline notify: %#v", msg.Lines)
	}
	if msg.Lines[1].Kind != uivm.LineBlank {
		t.Fatalf("want trailing blank, got %#v", msg.Lines)
	}
}

func TestResolveAccessRemoteTarget_PrefersSSHConfigOverSavedRemoteName(t *testing.T) {
	t.Setenv("DELVE_SHELL_ROOT", t.TempDir())
	home := t.TempDir()
	t.Setenv("HOME", home)
	t.Setenv("USER", "localuser")
	if err := config.EnsureRootDir(); err != nil {
		t.Fatal(err)
	}
	if err := config.AddRemote("ops@legacy.example.com", "prod", ""); err != nil {
		t.Fatal(err)
	}
	writeControllerTestSSHConfig(t, home, `
Host prod
  HostName current.example.com
  User deploy
  Port 2222
  IdentityFile ~/.ssh/prod_key
`)

	got := resolveAccessRemoteTarget("prod")
	if got.Target != "deploy@current.example.com:2222" {
		t.Fatalf("target=%q", got.Target)
	}
	if got.IdentityFile != filepath.Join(home, ".ssh", "prod_key") {
		t.Fatalf("identity=%q", got.IdentityFile)
	}
	if got.ConfigName != "prod" {
		t.Fatalf("config name=%q", got.ConfigName)
	}
}

func TestResolveAccessRemoteTarget_MatchesSSHConfigHostName(t *testing.T) {
	t.Setenv("DELVE_SHELL_ROOT", t.TempDir())
	home := t.TempDir()
	t.Setenv("HOME", home)
	t.Setenv("USER", "localuser")
	writeControllerTestSSHConfig(t, home, `
Host jump
  HostName jump.example.com
  User deploy
  Port 2201
`)

	got := resolveAccessRemoteTarget("jump.example.com")
	if got.Target != "deploy@jump.example.com:2201" {
		t.Fatalf("target=%q", got.Target)
	}
	if got.ConfigName != "jump" {
		t.Fatalf("config name=%q", got.ConfigName)
	}
}

func TestResolveAccessRemoteTarget_SSHConfigAliasBeatsSavedRemoteSameHost(t *testing.T) {
	t.Setenv("DELVE_SHELL_ROOT", t.TempDir())
	home := t.TempDir()
	t.Setenv("HOME", home)
	t.Setenv("USER", "localuser")
	if err := config.EnsureRootDir(); err != nil {
		t.Fatal(err)
	}
	if err := config.AddRemote("ops@192.168.140.200", "global master", "~/.ssh/remote_key"); err != nil {
		t.Fatal(err)
	}
	writeControllerTestSSHConfig(t, home, `
Host test
  HostName 192.168.140.200
  User ops
  IdentityFile ~/.ssh/test_key
`)

	got := resolveAccessRemoteTarget("test")
	if got.Target != "ops@192.168.140.200" {
		t.Fatalf("target=%q", got.Target)
	}
	if got.ConfigName != "test" {
		t.Fatalf("config name=%q want test", got.ConfigName)
	}
	if got.Label != "test (192.168.140.200)" {
		t.Fatalf("label=%q want test (192.168.140.200)", got.Label)
	}
	if got.IdentityFile != filepath.Join(home, ".ssh", "test_key") {
		t.Fatalf("identity=%q want ssh config identity", got.IdentityFile)
	}
}

func TestHandleSubmitNewSession_ReplacesTranscriptWithSessionBanner(t *testing.T) {
	i18n.SetLang("en")
	root := t.TempDir()
	t.Setenv("DELVE_SHELL_ROOT", root)
	if err := config.EnsureRootDir(); err != nil {
		t.Fatal(err)
	}
	s := &recordSender{}
	c := newTestControllerWithPresenter(s)
	c.runners = runnermgr.New(runnermgr.Options{})

	c.handleSubmitNewSession()

	msg := latestTranscriptReplaceMsg(t, s.msgs)
	if len(msg.Lines) < 2 {
		t.Fatalf("want session banner + blank, got %#v", msg.Lines)
	}
	if msg.Lines[len(msg.Lines)-2].Kind != uivm.LineSessionBanner {
		t.Fatalf("want session banner near tail, got %#v", msg.Lines)
	}
	if msg.Lines[len(msg.Lines)-1].Kind != uivm.LineBlank {
		t.Fatalf("want trailing blank, got %#v", msg.Lines)
	}
}

func TestHandleHistoryPreviewOpen_ReadsFullSession(t *testing.T) {
	i18n.SetLang("en")
	root := t.TempDir()
	t.Setenv("DELVE_SHELL_ROOT", root)
	if err := config.EnsureRootDir(); err != nil {
		t.Fatal(err)
	}

	sess, err := history.NewSession()
	if err != nil {
		t.Fatal(err)
	}
	if err := sess.AppendCommand("bash run.sh", true, "run demo skill", "low", history.CommandPayloadKindSkill, "demo", ""); err != nil {
		t.Fatal(err)
	}
	if err := sess.AppendCommandResult("bash run.sh", "needle-output", "", 0); err != nil {
		t.Fatal(err)
	}
	for i := 0; i < 205; i++ {
		if err := sess.AppendUserInput("filler"); err != nil {
			t.Fatal(err)
		}
	}
	if err := sess.Close(); err != nil {
		t.Fatal(err)
	}

	sessionID := strings.TrimSuffix(filepath.Base(sess.Path()), ".jsonl")
	sender := &recordSender{}
	c := newTestControllerWithPresenter(sender)

	c.handleHistoryPreviewOpen(sessionID)

	var msg ui.HistoryPreviewOverlayMsg
	found := false
	for _, raw := range sender.msgs {
		ov, ok := raw.(ui.HistoryPreviewOverlayMsg)
		if !ok {
			continue
		}
		msg = ov
		found = true
		break
	}
	if !found {
		t.Fatalf("no HistoryPreviewOverlayMsg in %#v", sender.msgs)
	}

	hasSkillLine := false
	hasResult := false
	for _, line := range msg.Lines {
		if line.Kind == uivm.LineSystemSuggest && line.Text == "Skill: demo" {
			hasSkillLine = true
		}
		if line.Kind == uivm.LineResult && strings.Contains(line.Text, "needle-output") {
			hasResult = true
		}
	}
	if !hasSkillLine {
		t.Fatalf("expected skill line in preview, got %#v", msg.Lines)
	}
	if !hasResult {
		t.Fatalf("expected early command result in preview, got %#v", msg.Lines)
	}
}

func TestHandleAgentUI_ApprovalRequest(t *testing.T) {
	s := &recordSender{}
	c := newTestControllerWithPresenter(s)
	req := &hiltypes.ApprovalRequest{Command: "ls", ResponseCh: make(chan hiltypes.ApprovalResponse, 1)}
	c.handleAgentUI(req)
	if len(s.msgs) != 1 {
		t.Fatalf("want 1 msg, got %d", len(s.msgs))
	}
	got, ok := s.msgs[0].(ui.ChoiceCardShowMsg)
	if !ok || got.PendingApproval == nil || got.PendingApproval.Command != "ls" {
		t.Fatalf("unexpected message: %T %#v", s.msgs[0], s.msgs[0])
	}
}

func latestTranscriptAppendMsg(t *testing.T, msgs []tea.Msg) ui.TranscriptAppendMsg {
	t.Helper()
	for i := len(msgs) - 1; i >= 0; i-- {
		if msg, ok := msgs[i].(ui.TranscriptAppendMsg); ok {
			return msg
		}
	}
	t.Fatalf("no TranscriptAppendMsg in %#v", msgs)
	return ui.TranscriptAppendMsg{}
}

func latestTranscriptReplaceMsg(t *testing.T, msgs []tea.Msg) ui.TranscriptReplaceMsg {
	t.Helper()
	for i := len(msgs) - 1; i >= 0; i-- {
		if msg, ok := msgs[i].(ui.TranscriptReplaceMsg); ok {
			return msg
		}
	}
	t.Fatalf("no TranscriptReplaceMsg in %#v", msgs)
	return ui.TranscriptReplaceMsg{}
}

func TestHandleAgentUI_SensitiveRequest(t *testing.T) {
	s := &recordSender{}
	c := newTestControllerWithPresenter(s)
	req := &hiltypes.SensitiveConfirmationRequest{Command: "cat /x", ResponseCh: make(chan hiltypes.SensitiveChoice, 1)}
	c.handleAgentUI(req)
	if len(s.msgs) != 1 {
		t.Fatalf("want 1 msg, got %d", len(s.msgs))
	}
	got, ok := s.msgs[0].(ui.ChoiceCardShowMsg)
	if !ok || got.PendingSensitive == nil || got.PendingSensitive.Command != "cat /x" {
		t.Fatalf("unexpected message: %T %#v", s.msgs[0], s.msgs[0])
	}
}

func TestHandleAgentUI_ExecEvent(t *testing.T) {
	s := &recordSender{}
	c := newTestControllerWithPresenter(s)
	c.handleAgentUI(hiltypes.ExecEvent{
		Command:   "ls",
		Allowed:   true,
		Result:    "ok",
		Sensitive: true,
		Suggested: false,
	})

	if len(s.msgs) != 1 {
		t.Fatalf("want 1 msg, got %d", len(s.msgs))
	}
	msg, ok := s.msgs[0].(ui.TranscriptAppendMsg)
	if !ok {
		t.Fatalf("wrong message type: %T", s.msgs[0])
	}
	_ = msg
}

func TestHandleAgentUI_UnknownPayloadIgnored(t *testing.T) {
	s := &recordSender{}
	c := newTestControllerWithPresenter(s)
	c.handleAgentUI(struct{ X int }{X: 1})
	if len(s.msgs) != 0 {
		t.Fatalf("unknown payload should be ignored, got %+v", s.msgs)
	}
}
