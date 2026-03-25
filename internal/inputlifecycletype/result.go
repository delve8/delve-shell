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
		cloned.Transcript = &payload
	}
	if event.Overlay != nil {
		payload := *event.Overlay
		cloned.Overlay = &payload
	}
	if event.Status != nil {
		payload := *event.Status
		cloned.Status = &payload
	}
	if event.Approval != nil {
		payload := *event.Approval
		cloned.Approval = &payload
	}
	if event.Error != nil {
		payload := *event.Error
		cloned.Error = &payload
	}
	if event.Message != nil {
		payload := *event.Message
		cloned.Message = &payload
	}
	return cloned
}
