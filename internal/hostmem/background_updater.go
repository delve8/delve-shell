package hostmem

import (
	"context"
	"encoding/json"
	"log/slog"
	"strings"
	"time"

	openaimodel "github.com/cloudwego/eino-ext/components/model/openai"
	"github.com/cloudwego/eino/schema"

	"delve-shell/internal/config"
	configllm "delve-shell/internal/config/llm"
	"delve-shell/internal/history"
)

const (
	defaultBackgroundUpdaterDebounce  = 1200 * time.Millisecond
	defaultBackgroundUpdaterMaxEvents = 48
)

type BackgroundAnalyzeInput struct {
	Context       Context
	MemorySummary string
	Events        []history.Event
}

type BackgroundAnalyzer interface {
	Analyze(context.Context, BackgroundAnalyzeInput) (UpdatePatch, error)
}

type BackgroundUpdater struct {
	analyzer  BackgroundAnalyzer
	logger    *slog.Logger
	stop      <-chan struct{}
	debounce  time.Duration
	maxEvents int

	requests chan backgroundUpdateRequest
}

type BackgroundUpdaterOptions struct {
	Analyzer  BackgroundAnalyzer
	Logger    *slog.Logger
	Stop      <-chan struct{}
	Debounce  time.Duration
	MaxEvents int
}

type backgroundUpdateRequest struct {
	sessionPath string
	ctx         Context
}

func NewBackgroundUpdater(opts BackgroundUpdaterOptions) *BackgroundUpdater {
	if opts.Analyzer == nil {
		opts.Analyzer = LLMBackgroundAnalyzer{}
	}
	u := &BackgroundUpdater{
		analyzer:  opts.Analyzer,
		logger:    opts.Logger,
		stop:      opts.Stop,
		debounce:  opts.Debounce,
		maxEvents: opts.MaxEvents,
		requests:  make(chan backgroundUpdateRequest, 32),
	}
	if u.debounce <= 0 {
		u.debounce = defaultBackgroundUpdaterDebounce
	}
	if u.maxEvents <= 0 {
		u.maxEvents = defaultBackgroundUpdaterMaxEvents
	}
	go u.run()
	return u
}

func (u *BackgroundUpdater) Enqueue(sessionPath string, ctx Context, event history.Event) {
	if u == nil || !ctx.Valid() || strings.TrimSpace(sessionPath) == "" {
		return
	}
	if !backgroundRelevantEvent(event.Type) {
		return
	}
	req := backgroundUpdateRequest{
		sessionPath: strings.TrimSpace(sessionPath),
		ctx:         ctx,
	}
	select {
	case u.requests <- req:
	default:
		select {
		case <-u.requests:
		default:
		}
		select {
		case u.requests <- req:
		default:
		}
	}
}

func backgroundRelevantEvent(eventType string) bool {
	switch eventType {
	case history.EventTypeCommandResult, history.EventTypeLLMResponse:
		return true
	default:
		return false
	}
}

func (u *BackgroundUpdater) run() {
	var (
		timer   *time.Timer
		timerCh <-chan time.Time
		pending *backgroundUpdateRequest
	)
	for {
		select {
		case <-u.stop:
			if timer != nil {
				timer.Stop()
			}
			return
		case req := <-u.requests:
			reqCopy := req
			pending = &reqCopy
			if timer == nil {
				timer = time.NewTimer(u.debounce)
				timerCh = timer.C
				continue
			}
			if !timer.Stop() {
				select {
				case <-timer.C:
				default:
				}
			}
			timer.Reset(u.debounce)
		case <-timerCh:
			if pending != nil {
				u.process(*pending)
				pending = nil
			}
			timerCh = nil
			timer = nil
		}
	}
}

func (u *BackgroundUpdater) process(req backgroundUpdateRequest) {
	if u == nil || u.analyzer == nil || !req.ctx.Valid() || strings.TrimSpace(req.sessionPath) == "" {
		return
	}
	events, err := history.ReadRecent(req.sessionPath, u.maxEvents)
	if err != nil || len(events) == 0 {
		u.debug("background host memory read skipped", "error", err)
		return
	}
	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	defer cancel()
	patch, err := u.analyzer.Analyze(ctx, BackgroundAnalyzeInput{
		Context:       req.ctx,
		MemorySummary: SummaryForContext(req.ctx),
		Events:        events,
	})
	if err != nil {
		u.debug("background host memory analyze skipped", "error", err)
		return
	}
	patch = normalizeUpdatePatch(patch)
	if isEmptyUpdatePatch(patch) {
		return
	}
	if _, err := Update(req.ctx, patch); err != nil {
		u.debug("background host memory update failed", "error", err)
		return
	}
	u.debug("background host memory updated", "host_id", req.ctx.HostID, "profile_key", req.ctx.ProfileKey)
}

func (u *BackgroundUpdater) debug(msg string, args ...any) {
	if u == nil || u.logger == nil {
		return
	}
	u.logger.Debug(msg, args...)
}

type LLMBackgroundAnalyzer struct{}

func (LLMBackgroundAnalyzer) Analyze(ctx context.Context, input BackgroundAnalyzeInput) (UpdatePatch, error) {
	cfg, err := config.LoadEnsured()
	if err != nil {
		return UpdatePatch{}, err
	}
	baseURL, apiKey, model := cfg.LLMResolved()
	chatModel, err := openaimodel.NewChatModel(ctx, &openaimodel.ChatModelConfig{
		APIKey:     apiKey,
		BaseURL:    baseURL,
		Model:      model,
		HTTPClient: configllm.NewLLMHTTPClient(20 * time.Second),
	})
	if err != nil {
		return UpdatePatch{}, err
	}
	msg, err := chatModel.Generate(ctx, []*schema.Message{
		schema.SystemMessage(backgroundAnalyzerSystemPrompt),
		schema.UserMessage(backgroundAnalyzerUserPrompt(input)),
	})
	if err != nil {
		return UpdatePatch{}, err
	}
	return parseBackgroundPatch(msg)
}

const backgroundAnalyzerSystemPrompt = `You maintain persistent host memory for an ops shell.
Read recent session events and return ONLY a single JSON object shaped like this optional patch schema:
{
  "role": "string",
  "role_confidence": 0.0,
  "os_family": "string",
  "capabilities_add": ["string"],
  "responsibilities_add": ["string"],
  "tags_add": ["string"],
  "notes_add": ["string"],
  "evidence_add": ["string"],
  "available_commands_add": ["string"],
  "missing_commands_add": ["string"],
  "package_managers_add": ["string"]
}

Rules:
- Return {} when there is no durable host-memory update.
- Only extract stable, reusable host facts. Ignore one-off incidents, ephemeral failures, and long summaries.
- Prefer command availability facts only when a command clearly succeeded or is explicitly shown present.
- Prefer missing command facts only when evidence clearly shows command not found, missing from PATH, or equivalent stable absence.
- Prefer concise semantic labels such as k8s_control_plane, k8s_worker, bastion, build_agent, monitoring_server, cluster_administration.
- Do not restate facts that are already in current host memory unless the new evidence materially adds something.
- Keep evidence_add and notes_add short. No markdown. No prose outside JSON.`

func backgroundAnalyzerUserPrompt(input BackgroundAnalyzeInput) string {
	var b strings.Builder
	b.WriteString("Current execution target:\n")
	b.WriteString("host_id=" + input.Context.HostID + "\n")
	b.WriteString("profile_key=" + input.Context.ProfileKey + "\n")
	if alias := strings.TrimSpace(input.Context.Alias); alias != "" {
		b.WriteString("alias=" + alias + "\n")
	}
	if input.Context.WeakIdentity {
		b.WriteString("weak_identity=true\n")
	}
	b.WriteString("\nCurrent host memory summary:\n")
	if s := strings.TrimSpace(input.MemorySummary); s != "" {
		b.WriteString(s)
	} else {
		b.WriteString("(empty)")
	}
	b.WriteString("\n\nRecent session events:\n")
	for _, ev := range input.Events {
		b.WriteString(ev.Time.UTC().Format(time.RFC3339))
		b.WriteString(" [")
		b.WriteString(ev.Type)
		b.WriteString("] ")
		b.WriteString(strings.TrimSpace(string(ev.Payload)))
		b.WriteString("\n")
	}
	return b.String()
}

func parseBackgroundPatch(msg *schema.Message) (UpdatePatch, error) {
	if msg == nil {
		return UpdatePatch{}, nil
	}
	raw := strings.TrimSpace(msg.Content)
	raw = strings.TrimPrefix(raw, "```json")
	raw = strings.TrimPrefix(raw, "```")
	raw = strings.TrimSuffix(raw, "```")
	raw = strings.TrimSpace(raw)
	if start := strings.IndexByte(raw, '{'); start >= 0 {
		if end := strings.LastIndexByte(raw, '}'); end >= start {
			raw = raw[start : end+1]
		}
	}
	if raw == "" {
		return UpdatePatch{}, nil
	}
	var patch UpdatePatch
	if err := json.Unmarshal([]byte(raw), &patch); err != nil {
		return UpdatePatch{}, err
	}
	return patch, nil
}

func normalizeUpdatePatch(patch UpdatePatch) UpdatePatch {
	patch.Role = strings.TrimSpace(patch.Role)
	patch.OSFamily = strings.TrimSpace(patch.OSFamily)
	if patch.RoleConfidence < 0 {
		patch.RoleConfidence = 0
	}
	if patch.RoleConfidence > 1 {
		patch.RoleConfidence = 1
	}
	patch.CapabilitiesAdd = trimList(patch.CapabilitiesAdd, 16)
	patch.ResponsibilitiesAdd = trimList(patch.ResponsibilitiesAdd, 16)
	patch.TagsAdd = trimList(patch.TagsAdd, 16)
	patch.NotesAdd = trimList(patch.NotesAdd, 8)
	patch.EvidenceAdd = trimList(patch.EvidenceAdd, 8)
	patch.AvailableAdd = normalizeCommands(strings.Join(patch.AvailableAdd, "\n"))
	patch.MissingAdd = normalizeCommands(strings.Join(patch.MissingAdd, "\n"))
	patch.PackageManagersAdd = normalizeCommands(strings.Join(patch.PackageManagersAdd, "\n"))
	return patch
}

func trimList(items []string, max int) []string {
	return mergeUnique(nil, items, max)
}

func isEmptyUpdatePatch(patch UpdatePatch) bool {
	return patch.Role == "" &&
		patch.RoleConfidence == 0 &&
		patch.OSFamily == "" &&
		len(patch.CapabilitiesAdd) == 0 &&
		len(patch.ResponsibilitiesAdd) == 0 &&
		len(patch.TagsAdd) == 0 &&
		len(patch.NotesAdd) == 0 &&
		len(patch.EvidenceAdd) == 0 &&
		len(patch.AvailableAdd) == 0 &&
		len(patch.MissingAdd) == 0 &&
		len(patch.PackageManagersAdd) == 0
}
