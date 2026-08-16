# shelp

Your AI-powered shell assistant. Convert natural language to safe, executable shell commands.

## Features

- **Natural Language Input**: Describe what you want in plain English
- **Review Before You Run**: Pick, edit or regenerate the generated commands before anything executes
- **Risk Labels**: Every command is labelled safe/caution/danger, and catastrophic patterns are blocked
- **Pipe Friendly**: Without a terminal shelp prints the commands instead of running them
- **BYOK**: Bring Your Own Key - use any OpenAI-compatible API
- **Shell Detection**: Generates commands compatible with your shell (bash, zsh, fish)

## Installation

### Homebrew (Recommended)

```bash
brew tap xqsit94/shelp
brew install shelp
```

**Update:**
```bash
brew update && brew upgrade shelp
```

**Uninstall:**
```bash
brew uninstall shelp
brew untap xqsit94/shelp
```

### Quick Install

```bash
curl -fsSL https://raw.githubusercontent.com/xqsit94/shelp/main/install.sh | bash
```

### Go Install

```bash
go install github.com/xqsit94/shelp@latest
```

### From Source

```bash
git clone https://github.com/xqsit94/shelp.git
cd shelp
go build -o shelp
mv shelp ~/.local/bin/
```

## Usage

### First Time Setup

On first run, shelp asks for your AI provider configuration in a short wizard
(API URL, API key, model), saves it to `~/.shelp/config.json` and immediately
tests the connection. If the test fails the configuration is kept and shelp
tells you which `shelp config set ...` command to run.

The wizard needs a terminal. On a machine where you cannot run it interactively,
set `SHELP_URL`, `SHELP_API_KEY` and `SHELP_MODEL` instead.

### Basic Usage

```bash
shelp "find all pdf files larger than 10MB"
shelp "show disk usage for current directory"
shelp "list all running docker containers"
shelp "show last 5 commits with file changes"
```

### Choosing What Runs

When the AI returns more than one command you get a multi-select list:

```
$ shelp "find javascript files changed this week"

Generated Commands (2)
├─ [●] find . -name "*.js" -mtime -7
│      ● safe
└─ [●] find . -name "*.jsx" -mtime -7
       ● safe

  2 of 2 selected

  ↑/↓: navigate • space: toggle • a: all • n: none • e: edit • r: regenerate • enter: execute • q: quit
```

| Key | Action |
| --- | --- |
| `↑`/`↓` (or `k`/`j`) | Move the cursor |
| `space` | Toggle the command under the cursor |
| `a` / `n` | Select all / select none |
| `e` | Edit the command under the cursor; `enter` saves, `esc` cancels |
| `r` | Regenerate, optionally with a refinement ("use fd instead") |
| `enter` | Run the selected commands |
| `q` / `esc` / `ctrl+c` | Quit without running anything |

A single command gets a menu instead:

```
$ shelp "show disk usage for current directory"

Generated Command
└─ du -sh .
   ● safe

> Execute
  Edit
  Regenerate
  Cancel

↑/↓: navigate • enter: select • y: execute • e: edit • r: regenerate • q: cancel
```

`y` executes, `e` edits the command inline, `r` regenerates (optionally with a
refinement) and `n`/`q`/`esc` cancels. Editing re-assesses the risk level: if an
edit turns the command into a blocked one, `Execute` disappears from the menu
and the command is deselected in the list.

Selected commands run one at a time with live output. When one fails you are
asked whether to continue with the rest (without a terminal the run stops), and
a summary tree is printed at the end.

### Flags

| Flag | Description |
| --- | --- |
| `-p`, `--print` | Print the generated commands to stdout, one per line, and exit. Nothing runs. |
| `-y`, `--yes` | Skip the confirmation UI and run the commands (blocked ones are skipped). |
| `-c`, `--copy` | Like `--print`, and copy the commands (newline-joined) to the clipboard. |
| `--debug` | Print the AI request and response to stderr (the API key is redacted). |
| `-v`, `--version` | Print the version. |
| `-h`, `--help` | Print help. |

`--print` wins over `--yes`.

### Non-Interactive Use

When stdin or stdout is not a terminal, shelp behaves as if `--print` was given,
so it composes with pipes and command substitution:

```bash
shelp -p "compress this folder as backup.tar.gz" | pbcopy
cmd=$(shelp -p "count lines of go code")
echo "$cmd"
```

Output is plain text when stdout is redirected and syntax-highlighted when it is
a terminal. For unattended runs that should actually execute, use `--yes`:

```bash
shelp -y "restart the docker compose stack"
```

### Configuration

```bash
# Update AI URL
shelp config set url https://openrouter.ai/api/v1/chat/completions

# Update API key (hidden input)
shelp config set key

# Update model
shelp config set model anthropic/claude-3.5-sonnet

# Show current configuration (API key masked, env values marked)
shelp config show

# Send one harmless request to the provider and report the result
shelp config test

# Remove the config file
shelp config reset
```

Environment variables override the config file:

| Variable | Effect |
| --- | --- |
| `SHELP_URL` | AI API endpoint |
| `SHELP_API_KEY` | API key |
| `SHELP_MODEL` | Model name |
| `SHELP_CONFIG_DIR` | Config directory (default `~/.shelp`) |
| `SHELP_DEBUG=1` | Same as `--debug` |

Precedence is environment > config file. `shelp config set ...` always writes to
the file, never to the environment, and `shelp config show` marks the values
that came from the environment with `(from env)`.

shelp warns when the API URL uses `http://` with a non-local host, because the
API key is then sent in cleartext.

### Exit Codes

| Code | Meaning |
| --- | --- |
| `0` | Everything succeeded, or the commands were only printed |
| `1` | Configuration/API error, no commands generated, or all commands blocked |
| `130` | Cancelled (`q`, `esc`, `ctrl+c`) |
| other | The exit code of the last command that failed (`128 + signal` if it was killed) |

## Supported AI Providers

shelp works with any OpenAI-compatible chat completions API:

- [OpenRouter](https://openrouter.ai) - Access multiple AI models
- [OpenAI](https://openai.com) - GPT-4, GPT-3.5
- [Anthropic](https://anthropic.com) - Claude (via proxy)
- [Together AI](https://together.ai) - Various open models
- [Groq](https://groq.com) - Fast inference
- Local models via [Ollama](https://ollama.ai), [LM Studio](https://lmstudio.ai), etc.

## Safety

**The confirmation prompt is the real safety mechanism.** Generated commands run
in your shell, as you, with your privileges and your environment. Read them
before pressing enter. `--yes` removes that step entirely, so use it only for
requests whose outcome you can predict.

The blocklist is a speed bump for catastrophic mistakes, not a sandbox. It is a
set of regular expressions over the command text: a determined command (or a
creative AI) can slip past it with variable indirection, encoding or quoting
tricks. Do not treat "not blocked" as "safe".

### Risk Levels

- **safe** (green): everything that does not match a caution pattern
- **caution** (yellow): `sudo`, `rm -rf`, `chmod`/`chown`, `dd`, `mkfs`,
  `fdisk`/`parted`, `kill`/`pkill`, `systemctl stop|restart|disable`,
  `service ... stop|restart`, `reboot`/`shutdown`/`init N`, writes into `/etc/`,
  and package installs (`pip install`, `npm install -g`, `brew install`,
  `apt|yum|dnf install`)
- **danger** (red): blocked commands - they cannot be selected or executed

### Blocked Commands

Blocking is case-insensitive and applies per command segment: the line is split
on `;`, `|`, `&` and newlines, and `sudo`/`doas`/`env`/`VAR=value` prefixes are
stripped before matching, so `cd /tmp && sudo rm -rf /` is still caught.

- `rm` with `-r`/`-f`/`--recursive`/`--force`/`--no-preserve-root` targeting `/`,
  `//`, `/*`, `~`, `~/`, `~/*`, `~/.`, `$HOME` or `${HOME}` (quoted or not)
- any `rm` with `--no-preserve-root`
- fork bombs (`:(){ :|:& };:`)
- `dd of=` a whole-disk device, `> /dev/sdX`, `mkfs* /dev/…`, `wipefs`/`shred`
  on a device node
- `chmod`/`chown -R` on `/` or `~`
- `mv / …` and `mv ~ /dev/null`
- piping a download into a shell: `curl|wget … | sh`, `sh <(curl …)`,
  `sh -c "$(curl …)"`, `echo … | base64 -d | sh`
- `perl -e '… exec …'` and `python -c '… exec …'`
- `find /` or `find ~` with `-delete`/`-exec rm` and no narrowing predicate
  (`-name`, `-path`, `-regex`, `-mtime`, `-mmin`, `-newer`, `-size`, `-empty`)

## Known Limitations

- Every generated command runs in its own fresh non-interactive shell, so `cd`,
  shell variables, aliases and your rc files do not carry over between commands.
  Dependent steps have to be joined with `&&` inside a single command (the AI is
  instructed to do this).
- Commands inherit your terminal, so interactive ones work, but shelp cannot
  tell what a command changed once it exits.
- macOS and Linux only; there is no Windows build.
- `--copy` needs a clipboard tool: `pbcopy` on macOS, `xclip` or `xsel` on Linux.
  Without one it warns and still prints the commands.

## Configuration File

Configuration is stored in `~/.shelp/config.json` (or `$SHELP_CONFIG_DIR`):

```json
{
  "ai_url": "https://openrouter.ai/api/v1/chat/completions",
  "api_key": "sk-or-...",
  "model": "anthropic/claude-3.5-sonnet"
}
```

File permissions are set to `0600` (owner read/write only) for security.

## Contributing

Contributions are welcome - see [CONTRIBUTING.md](CONTRIBUTING.md). Security
issues are covered in [SECURITY.md](SECURITY.md).

## License

MIT License - see [LICENSE](LICENSE) for details.
