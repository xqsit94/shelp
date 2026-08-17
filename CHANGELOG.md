# Changelog

All notable changes to this project are documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.1.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [Unreleased]

### Added

- Long command lists now scroll instead of overflowing the terminal: only the
  window that fits is drawn, with `↑ N more` / `↓ N more` markers, and the
  focused command is always kept on screen.
- `?` toggles a full list of key bindings in the command list.
- Provider errors now suggest the command that fixes them (wrong key, wrong
  endpoint, rate limit, unreachable host).
- Shell integration: `shelp init zsh|bash|fish|powershell` prints a snippet that
  binds `ctrl+g` to a widget which sends the current command line to shelp and
  replaces it with the generated commands (joined with ` && `, or `; ` in
  PowerShell). An empty line does nothing and a failure leaves the line
  untouched with the error on stderr. The key is rebound by pasting the snippet
  into the startup file instead of eval-ing it.
- Experimental Windows build: `windows/amd64` and `windows/arm64` are released
  as `shelp-windows-<arch>.zip`, commands run through `pwsh`, `powershell` or
  `cmd`, the system prompt gained Windows hints, and the blocklist gained
  catastrophic Windows commands (recursive/forced `Remove-Item`/`del`/`rd` on a
  drive root, `format`, `Format-Volume`, `Clear-Disk`, `Initialize-Disk`,
  `diskpart`). It is cross-compiled and not yet tested at runtime.
- Query history: every answered query is appended to
  `~/.shelp/history.jsonl` (time, query, commands, whether they ran, exit code
  and profile, mode `0600`, newest 1000 entries kept). `shelp history` lists the
  most recent queries, `shelp history run <n>` runs one of them again through
  the normal flow, and `shelp history clear` deletes the file. Recording is
  skipped with `--no-history` or `SHELP_NO_HISTORY=1`.
- Named provider profiles: the config file now holds `active_profile` and a
  `profiles` map, managed with `shelp config profile list|use|add|remove|rename`
  and selected per run with `--profile <name>` or `SHELP_PROFILE`. Single-profile
  files from earlier versions are read as the `default` profile and rewritten in
  the new format on the next write.
- Each generated command now comes with a one-line explanation of what it does,
  shown next to the risk level in the command list, the single-command menu, the
  `--yes` plan and `shelp config test`. `--print`, `--copy` and non-terminal runs
  still print commands only, and editing a command drops its explanation.
- Optional sampling parameters: `shelp config set temperature <0-2>` and
  `shelp config set max-tokens <n>` (cleared again with `shelp config unset ...`),
  overridable with `SHELP_TEMPERATURE` and `SHELP_MAX_TOKENS`. Both are unset by
  default and are then omitted from the request, because some OpenAI-compatible
  reasoning models reject them.
- `-p`/`--print` prints the generated commands to stdout, one per line, without
  running them; plain text when stdout is redirected, syntax-highlighted on a
  terminal.
- `-y`/`--yes` runs the generated commands without the confirmation UI, after
  listing each one with its risk level and skipping blocked ones.
- `-c`/`--copy` prints the commands and copies them to the clipboard, warning on
  stderr (and still printing) when no clipboard tool is available.
- Runs without a terminal (piped input or output) now print the commands instead
  of failing, so `$(shelp -p …)` and `shelp … | pbcopy` work.
- Edit before execute: `e` in the multi-select list and `Edit` in the
  single-command menu open an inline editor; saving re-assesses the risk level
  and deselects the command if the edit turned it into a blocked one.
- The single-command menu can now take a refinement when regenerating, like the
  multi-command list.
- `SHELP_URL`, `SHELP_API_KEY` and `SHELP_MODEL` override the config file, and
  `SHELP_CONFIG_DIR` relocates the config directory. `shelp config show` marks
  values that came from the environment.
- `shelp config test` sends one harmless request to the provider and reports the
  endpoint, model, command count and round-trip time, or the error verbatim. The
  same check runs automatically after the first-run wizard.
- A warning when the API URL uses `http://` with a non-local host, because the
  API key is then sent in cleartext.
- Process exit codes: `0` success, `1` error or no commands, `130` cancelled,
  otherwise the exit code of the last failed command (`128 + signal` when it was
  killed).
- Retries with backoff and `Retry-After` support for `429`/`5xx`/transport
  failures, plus `--debug`/`SHELP_DEBUG=1` request and response tracing with the
  `Authorization` header redacted.
- The system prompt now includes the shell, OS/arch, working directory and
  BSD-vs-GNU hints, and states that each array entry runs in its own shell.
- Regenerating sends the rejected commands and the user's refinement as
  conversation history instead of repeating the original query.
- CI workflow (build, vet, gofmt, `test -race`, tidy check, staticcheck,
  govulncheck) and tests for the AI client, executor, config, safety rules,
  prompt models, paths and the root command.
- `CONTRIBUTING.md` and `SECURITY.md`.

### Changed

- Key hints are generated from the bindings themselves with `bubbles/help`
  instead of four hardcoded strings, so they adapt to the terminal width and
  cannot drift from what the keys actually do. Every screen sizes itself from
  `tea.WindowSizeMsg`, so resizing reflows the view.
- Selection checkboxes are `[✓]`/`[ ]`; `[●]` collided with the `●` used for
  the safe risk level on the line below.
- `shelp history` prints `(not run)` instead of an undocumented dash, and
  `shelp config profile list` names the profile in use rather than reporting
  none while `config show` reports `default`.

- GitHub Actions bumped to versions that run on Node 24 (`actions/checkout@v5`,
  `actions/setup-go@v6`, `actions/upload-artifact@v6`,
  `actions/download-artifact@v7`, `softprops/action-gh-release@v3`), and CI now
  cross-compiles for windows/amd64, windows/arm64, linux/amd64 and darwin/arm64.
- Commands stream straight to the terminal and inherit stdin, so interactive and
  long-running commands work; the alt-screen output viewer that erased output
  from the scrollback is gone.
- Command list, confirmation and summary views use a tree-style layout with risk
  icons instead of boxes, with distinct icons (`✓ ✕ ▲`) rather than colour alone.
- `ctrl+c` now cancels the in-flight AI request and the running command instead
  of only closing the TUI.
- The syntax highlighting theme is picked once from the terminal background.
- `shelp config set …` writes only to the config file, so environment overrides
  are never persisted.
- Version is injected at build time via ldflags instead of being hard-coded.

### Fixed

- The focused row in the command list is marked with a caret. It was previously
  distinguished only by the colour of its tree branch, so under `NO_COLOR` the
  view was identical before and after moving the cursor and there was no way to
  tell which command `space` or `e` would act on.
- Commands longer than the terminal are truncated with an ellipsis. They were
  the one thing clipped at the screen edge without a marker, while the
  explanation below them was truncated properly, so the text being approved
  could differ from the text that ran.
- Key hints no longer clip mid-word. At 80 columns `enter: execute` and
  `q: quit` were cut off entirely, leaving no on-screen way to run or quit.
- The first-run wizard no longer draws skewed input boxes: a prefix was
  concatenated onto a three-line bordered box and indented only its first line.
- The edit and refine screens no longer scatter their input across the screen.
  A newline inside a `lipgloss.Render` call pads the block and pushes the next
  write across by the width of the line above it.
- `shelp config test` no longer writes ANSI escapes into redirected output and
  now honours `NO_COLOR`, which the syntax highlighter ignored.
- A failing command reports its exit code as it happens, instead of asking
  whether to continue with no indication that anything went wrong.

- `install.sh` downloads the released `.tar.gz` and verifies its SHA-256
  checksum instead of fetching a raw binary URL that returned 404.
- Safety rules no longer flag every `sudo rm <path>`, and no longer miss
  `rm -rf /*`, `rm -rf $HOME`, chained commands, `dd` to block devices or
  `curl … | sudo bash`. Unfiltered `find / -delete` is blocked; filtered finds
  stay at caution level.
- A destructive target is now caught anywhere in the argument list, not only as
  the last one, so `rm -rf $HOME /tmp/cache` and `chmod -R 777 / /home` are
  blocked. Paths that merely start with the same characters, such as
  `rm -rf /tmp/cache`, are still allowed.
- Pipes into a privileged shell are recognised when `sudo`/`doas` carry their own
  options, e.g. `curl … | sudo -u root bash`.
- `--yes` no longer stops on an interactive "Continue with next command?"
  question when a command fails: the run ends there, as it already did without a
  terminal.
- Short API keys are fully masked in `shelp config show` instead of revealing
  most of the key.
- `Truncate` is rune- and ANSI-aware, so multibyte commands are no longer cut
  mid-character.
- Errors are printed once instead of twice, and usage is no longer dumped on
  runtime failures.
- Removed dead code and unused identifiers from the prompt package.
