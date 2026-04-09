package skill

import (
	"os"
	"path/filepath"
	"strings"

	"delve-shell/internal/host/cmd"
	"delve-shell/internal/i18n"
	"delve-shell/internal/input/lifecycletype"
	"delve-shell/internal/skill/store"
	slashskill "delve-shell/internal/slash/skill"
	"delve-shell/internal/ui"
)

func registerSlashExecutionProvider() {
	ui.RegisterSlashExecutionProvider(func(req ui.SlashExecutionRequest) (inputlifecycletype.ProcessResult, bool, error) {
		text := strings.TrimSpace(req.RawText)
		switch {
		case text == slashskill.Command(slashskill.ReservedNew):
			return ui.SlashOverlayOpenResult(OverlayOpenKeyAdd, "", "", false, nil), true, nil
		case strings.HasPrefix(text, "/config add-skill"):
			return ui.SlashOverlayOpenResult(OverlayOpenKeyAdd, "", "", false, nil), true, nil
		case strings.HasPrefix(text, "/config update-skill"):
			name := strings.TrimSpace(strings.TrimPrefix(text, "/config update-skill"))
			if name == "" {
				return ui.SlashTranscriptErrorResult(i18n.T(i18n.KeyDescConfigUpdateSkill)), true, nil
			}
			return ui.SlashOverlayOpenResult(OverlayOpenKeyUpdate, "", "", false, map[string]string{"name": name}), true, nil
		case strings.HasPrefix(text, "/config del-skill "):
			name := strings.TrimSpace(strings.TrimPrefix(text, "/config del-skill "))
			return handleSlashConfigDelSkillPrefix(name), true, nil
		case strings.HasPrefix(text, slashskill.Prefix):
			return executeSkillInvocation(req, strings.TrimSpace(strings.TrimPrefix(text, slashskill.Prefix))), true, nil
		default:
			return inputlifecycletype.ProcessResult{}, false, nil
		}
	})
}

func executeSkillInvocation(req ui.SlashExecutionRequest, rest string) inputlifecycletype.ProcessResult {
	if req.OfflineExecutionMode {
		return ui.SlashTranscriptErrorResult(i18n.T(i18n.KeyOfflineSlashSkillDisabled))
	}
	fields := strings.Fields(rest)
	if len(fields) < 1 {
		return ui.SlashTranscriptErrorResult(i18n.T(i18n.KeyUsageSkill))
	}
	skillName := fields[0]
	naturalLanguage := strings.TrimSpace(strings.TrimPrefix(rest, skillName))
	if naturalLanguage == "" {
		naturalLanguage = "Follow SKILL.md (no extra detail from the user)."
	}
	skillDir := skillstore.SkillDir(skillName)
	if _, err := os.Stat(filepath.Join(skillDir, "SKILL.md")); err != nil {
		return ui.SlashTranscriptErrorResult(i18n.T(i18n.KeySkillNotFound))
	}
	skillContent, err := skillstore.ReadSKILLContent(skillDir)
	if err != nil {
		return ui.SlashTranscriptErrorResult(i18n.Tf(i18n.KeySkillInstallFailed, err))
	}
	payload := skillInvocationPrompt(skillName, skillContent, naturalLanguage)
	userLine := strings.TrimSpace(req.RawText)
	if req.CommandSender == nil || !req.CommandSender.Send(hostcmd.Submission{
		Submission: inputlifecycletype.InputSubmission{
			Kind:                     inputlifecycletype.SubmissionChat,
			Source:                   inputlifecycletype.SourceProgrammatic,
			RawText:                  payload,
			SessionDisplayText:       userLine,
			SkillInvocationSkillName: skillName,
		},
	}) {
		return inputlifecycletype.ProcessResult{}
	}
	return ui.SlashProcessingResult()
}
