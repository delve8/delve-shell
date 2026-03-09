<img src="assets/logo.svg" width="64" height="56" alt="delve-shell" />

# delve-shell

AI-assisted ops CLI: chat with an AI in the terminal to run ops tasks. **Every command is shown and executed only after user confirmation (y/n).** Commands not on the allowlist always require approval. Suited for production and auditable workflows.

## Overview

- **Human-in-the-loop (HIL)**: Proposed commands are listed explicitly; execution happens only after the user approves or rejects. The tool does not rely on the AI’s in-chat “shall I run this” as a safety boundary.
- **Allowlist and auto-run**: By default, allowlisted commands (e.g. read-only `ls`, `cat`, `git status`) run without confirmation; others show an approval card (Run / Reject). With **Auto-run: None** (`/config allowlist_auto_run off`), every command shows a card (Run / Copy / Dismiss). The allowlist uses regexes and can be updated with `/config allowlist update` to merge in built-in defaults.
- **Config and i18n**: `config.yaml` sets the LLM (base_url, api_key, model), UI language (en/zh), etc. Environment variables are supported via `$VAR` / `${VAR}`.
- **Multi-platform**: Linux, macOS, Windows; amd64 and arm64.

## Config paths

On first run, the app creates `config.yaml`, `allowlist.yaml`, and related files under a config root directory.

| Platform | Default config root |
|----------|---------------------|
| Linux    | `~/.delve-shell`     |
| macOS    | `~/.delve-shell`     |
| Windows  | `%USERPROFILE%\.delve-shell` (e.g. `C:\Users\<username>\.delve-shell`) |

Override the root with an environment variable:

```bash
export DELVE_SHELL_ROOT=/path/to/my-dir   # Linux/macOS
set DELVE_SHELL_ROOT=D:\my-dir            # Windows
```

Main config: `<root>/config.yaml`  
Allowlist: `<root>/allowlist.yaml`  
Session data: `<root>/sessions`

## Configuration

### config.yaml

- **llm.base_url**: OpenAI-compatible API base URL. Empty means default OpenAI.
- **llm.api_key**: API key (required). Supports `$VAR` and `${VAR}` for environment variables.
- **llm.model**: Model name. Empty defaults to `gpt-4o-mini`.
- **language**: UI language: `en` or `zh`.
- **allowlist_auto_run**: When `true` (default), allowlisted commands run without confirmation; when `false`, every command shows an approval card (Run / Copy / Dismiss).

These can be changed from inside the app via slash commands, e.g.:

- `/config show`: Show config path and LLM summary.
- `/config auto-run list-only` or `disable`: Set whether listed commands auto-run (saved to config).
- `/config llm api_key <key>`: Set API key.
- `/config llm base_url <url>`: Set base_url (e.g. DashScope compatible endpoint).
- `/config llm model <name>`: Set model.
- `/config language en` or `zh`: Set UI language.

### Allowlist (allowlist.yaml)

Each line is one regex; matching commands run without confirmation. Built-in defaults include common read-only commands (e.g. `pwd`, `ls`, `git status`, `kubectl get`).  
Use **`/config allowlist update`** in the app to merge the current built-in defaults into your `allowlist.yaml` (only missing entries are added), then **`/reload`** to apply.

## Usage

1. Start: `./bin/delve-shell` (or `delve-shell` if on PATH).
2. Type a natural-language description of the task and press Enter; the AI replies and may propose commands.
3. If a proposed command is not on the allowlist, the UI shows the command and **Approve? (y/n)**; press `y` to run, `n` to reject.
4. Commands on the allowlist run directly and are tagged as allowlist; others are tagged as approved after confirmation.

### Slash commands

Type `/` to list and complete these commands (order: help → cancel → config → new → sessions → reload → run → sh → exit):

| Command        | Description |
|----------------|-------------|
| `/help`        | Show help and slash command list |
| `/cancel`      | Cancel the current AI request |
| `/config`      | Config (sub: show, auto-run list-only/disable, allowlist update, llm base_url/api_key/model, language) |
| `/config auto-run list-only`  | Listed commands run without confirmation (saved to config) |
| `/config auto-run disable`    | Every command shows Run / Copy / Dismiss (saved to config) |
| `/new`         | Start a new session |
| `/sessions`    | List and switch to another session (optional filter after space) |
| `/reload`      | Reload config and allowlist without restart |
| `/run <cmd>`   | Run a single command directly (no AI) |
| `/sh`          | Start the system shell; return to this session when it exits |
| `/exit`        | Quit (ctrl+c also works) |

After typing `/`, use **Up/Down** to select a suggestion and **Enter** to fill the input (no execution). Type the full command and press Enter again to run it.

### Keyboard and scrolling

- **Up / Down / PgUp / PgDown**: Scroll the conversation; when input starts with `/`, Up/Down cycle slash suggestions.
- **Enter**: Send input or confirm/reject a command.
- **ctrl+c**: Quit.
