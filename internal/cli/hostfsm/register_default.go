package hostfsm

func init() {
	Register(Transition{
		From: StateIdle,
		On:   EvtLLMRunStart,
		To:   StateLLMRunning,
	})
	Register(Transition{
		From: StateLLMRunning,
		On:   EvtLLMRunEnd,
		To:   StateIdle,
	})
}
