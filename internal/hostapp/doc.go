// Package hostapp defines the injectable Host façade (*Runtime): bus input channels, allowlist/remote UI mirrors,
// and config-LLM startup one-shot. The interactive CLI constructs a *Runtime, wires it via hostwiring.BindSendPorts,
// and passes it into ui.Model as Host.
package hostapp
