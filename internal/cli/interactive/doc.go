// Package interactive wires config, sessions, host bus, runner manager, and the Bubble Tea UI loop.
// It keeps the cmd/delve-shell entrypoint thin while centralizing startup sequencing.
//
// Startup splits into Run (preflight, wiring, host controller) and tui_loop.go (restartable TUI plus
// optional embedded subshell when the shell bridge delivers saved transcript lines).
package interactive
