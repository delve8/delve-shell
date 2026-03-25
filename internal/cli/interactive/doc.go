// Package interactive wires config, sessions, host bus, runner manager, and the Bubble Tea UI loop.
// It keeps cmd/delve-shell and internal/cli entrypoints thin while centralizing startup sequencing.
//
// Startup splits into Run (preflight, wiring, host controller) and tui_loop.go (restartable TUI plus
// optional embedded subshell when the shell bridge delivers saved transcript lines).
package interactive
