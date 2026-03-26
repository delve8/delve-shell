package skill

import (
	"os"
	"path/filepath"
	"strings"

	"delve-shell/internal/i18n"
	"delve-shell/internal/inputlifecycletype"
	"delve-shell/internal/skills"
	"delve-shell/internal/ui"
	"delve-shell/internal/uivm"
)

func registerSlashExecutionProvider() {
	ui.RegisterSlashExecutionProvider(func(req ui.SlashExecutionRequest) (inputlifecycletype.ProcessResult, bool, error) {
		text := strings.TrimSpace(req.RawText)
		switch {
		case text == "/config add-skill":
			return overlayOpenResult("skill_add", nil), true, nil
		case strings.HasPrefix(text, "/config add-skill"):
			url, ref, path := parseAddSkillArgs(strings.TrimSpace(strings.TrimPrefix(text, "/config add-skill")))
			return overlayOpenResult("skill_add", map[string]string{
				"url":  url,
				"ref":  ref,
				"path": path,
			}), true, nil
		case strings.HasPrefix(text, "/config update-skill"):
			name := strings.TrimSpace(strings.TrimPrefix(text, "/config update-skill"))
			if name == "" {
				return transcriptErrorResult(i18n.T("en", i18n.KeyDescConfigUpdateSkill)), true, nil
			}
			return overlayOpenResult("skill_update", map[string]string{"name": name}), true, nil
		case strings.HasPrefix(text, "/config del-skill "):
			name := strings.TrimSpace(strings.TrimPrefix(text, "/config del-skill "))
			return handleSlashConfigDelSkillPrefix(name), true, nil
		case strings.HasPrefix(text, "/skill "):
			return executeSkillInvocation(req, strings.TrimSpace(strings.TrimPrefix(text, "/skill "))), true, nil
		default:
			return inputlifecycletype.ProcessResult{}, false, nil
		}
	})
}

func parseAddSkillArgs(rest string) (url, ref, path string) {
	if rest == "" {
		return "", "", ""
	}
	fields := strings.Fields(rest)
	if len(fields) >= 1 {
		url = fields[0]
	}
	if len(fields) >= 2 {
		if strings.Contains(fields[1], "/") {
			path = fields[1]
		} else {
			ref = fields[1]
		}
	}
	if len(fields) >= 3 {
		ref = fields[1]
		path = fields[2]
	}
	return url, ref, path
}

func executeSkillInvocation(req ui.SlashExecutionRequest, rest string) inputlifecycletype.ProcessResult {
	fields := strings.Fields(rest)
	if len(fields) < 1 {
		return transcriptErrorResult(i18n.T("en", i18n.KeyUsageSkill))
	}
	skillName := fields[0]
	naturalLanguage := strings.TrimSpace(strings.TrimPrefix(rest, skillName))
	if naturalLanguage == "" {
		return transcriptErrorResult(i18n.T("en", i18n.KeyUsageSkill))
	}
	skillDir := skills.SkillDir(skillName)
	if _, err := os.Stat(filepath.Join(skillDir, "SKILL.md")); err != nil {
		return transcriptErrorResult(i18n.T("en", i18n.KeySkillNotFound))
	}
	skillContent, err := skills.ReadSKILLContent(skillDir)
	if err != nil {
		return transcriptErrorResult(i18n.Tf("en", i18n.KeySkillInstallFailed, err))
	}
	payload := skillInvocationPrompt(skillName, skillContent, naturalLanguage)
	if req.ActionSender == nil || !req.ActionSender.Send(uivm.UIAction{
		Kind: uivm.UIActionSubmission,
		Submission: inputlifecycletype.InputSubmission{
			Kind:    inputlifecycletype.SubmissionChat,
			Source:  inputlifecycletype.SourceProgrammatic,
			RawText: payload,
		},
	}) {
		return inputlifecycletype.ProcessResult{}
	}
	res := inputlifecycletype.ConsumedResult(inputlifecycletype.OutputEvent{
		Kind:   inputlifecycletype.OutputStatusChange,
		Status: &inputlifecycletype.StatusPayload{Key: "processing"},
	})
	res.WaitingForAI = true
	return res
}

func messageResult(msg any) inputlifecycletype.ProcessResult {
	return inputlifecycletype.ConsumedResult(inputlifecycletype.OutputEvent{
		Kind: inputlifecycletype.OutputMessage,
		Message: &inputlifecycletype.MessagePayload{
			Value: msg,
		},
	})
}

func overlayOpenResult(key string, params map[string]string) inputlifecycletype.ProcessResult {
	return inputlifecycletype.ConsumedResult(inputlifecycletype.OutputEvent{
		Kind: inputlifecycletype.OutputOverlayOpen,
		Overlay: &inputlifecycletype.OverlayPayload{
			Key:    key,
			Params: params,
		},
	})
}

func transcriptErrorResult(text string) inputlifecycletype.ProcessResult {
	return inputlifecycletype.ConsumedResult(inputlifecycletype.OutputEvent{
		Kind: inputlifecycletype.OutputMessage,
		Message: &inputlifecycletype.MessagePayload{
			Value: ui.TranscriptAppendMsg{Lines: []uivm.Line{
				{Kind: uivm.LineSystemError, Text: text},
			}},
		},
	})
}
