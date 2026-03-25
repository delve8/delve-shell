// Package hostapp is the single process-wide host façade: bus input channels, allowlist/remote UI mirrors,
// and config-LLM startup one-shot. CLI startup wires these once; tests use ResetTestState.
package hostapp
