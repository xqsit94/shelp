# Changelog

All notable changes to this project are documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.1.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [Unreleased]

### Added

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

- `install.sh` downloads the released `.tar.gz` and verifies its SHA-256
  checksum instead of fetching a raw binary URL that returned 404.
- Safety rules no longer flag every `sudo rm <path>`, and no longer miss
  `rm -rf /*`, `rm -rf $HOME`, chained commands, `dd` to block devices or
  `curl … | sudo bash`. Unfiltered `find / -delete` is blocked; filtered finds
  stay at caution level.
- Short API keys are fully masked in `shelp config show` instead of revealing
  most of the key.
- `Truncate` is rune- and ANSI-aware, so multibyte commands are no longer cut
  mid-character.
- Errors are printed once instead of twice, and usage is no longer dumped on
  runtime failures.
- Removed dead code and unused identifiers from the prompt package.
