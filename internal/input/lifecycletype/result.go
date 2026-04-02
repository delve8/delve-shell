package inputlifecycletype

// ProcessResult is the normalized outcome from a submission processor.
type ProcessResult struct {
	Outputs      []OutputEvent
	WaitingForAI bool
	Consumed     bool
	Err          error
}

// ConsumedResult marks a submission as handled with optional output events.
func ConsumedResult(outputs ...OutputEvent) ProcessResult {
	return ProcessResult{
		Outputs:  cloneOutputEvents(outputs),
		Consumed: true,
	}
}

// ErrorResult marks a submission as handled with an error and optional output events.
func ErrorResult(err error, outputs ...OutputEvent) ProcessResult {
	return ProcessResult{
		Outputs:  cloneOutputEvents(outputs),
		Consumed: true,
		Err:      err,
	}
}

func cloneOutputEvents(events []OutputEvent) []OutputEvent {
	if len(events) == 0 {
		return nil
	}
	out := make([]OutputEvent, 0, len(events))
	for _, event := range events {
		out = append(out, cloneOutputEvent(event))
	}
	return out
}

func cloneOutputEvent(event OutputEvent) OutputEvent {
	cloned := event
	if event.Transcript != nil {
		payload := *event.Transcript
		if len(event.Transcript.Lines) > 0 {
			payload.Lines = make([]TranscriptLine, len(event.Transcript.Lines))
			copy(payload.Lines, event.Transcript.Lines)
		}
		cloned.Transcript = &payload
	}
	if event.Slash != nil {
		payload := *event.Slash
		cloned.Slash = &payload
	}
	if event.Overlay != nil {
		payload := *event.Overlay
		if len(event.Overlay.Params) > 0 {
			payload.Params = make(map[string]string, len(event.Overlay.Params))
			for k, v := range event.Overlay.Params {
				payload.Params[k] = v
			}
		}
		cloned.Overlay = &payload
	}
	if event.Status != nil {
		payload := *event.Status
		cloned.Status = &payload
	}
	if event.CommandExec != nil {
		payload := *event.CommandExec
		cloned.CommandExec = &payload
	}
	if event.Approval != nil {
		payload := *event.Approval
		cloned.Approval = &payload
	}
	if event.Error != nil {
		payload := *event.Error
		cloned.Error = &payload
	}
	return cloned
}
