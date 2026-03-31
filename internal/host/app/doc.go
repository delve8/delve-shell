// Package app defines the injectable Host façade (*Runtime): bus input channels, remote UI mirrors,
// and config-LLM startup one-shot. The interactive CLI constructs a *Runtime, wires it directly,
// and adapts it to ui read-model/action channels in the interactive loop.
package app
