# Security Policy

## Reporting a vulnerability

Please report security issues through GitHub's private vulnerability reporting:
open the [Security tab](https://github.com/xqsit94/shelp/security/advisories) of
this repository and choose "Report a vulnerability". If that is unavailable,
open a regular [issue](https://github.com/xqsit94/shelp/issues) that says a
security report is pending and asks for a private contact - do not include the
details in the public issue.

Please include the shelp version (`shelp --version`), your OS, and the steps to
reproduce.

## Where your API key lives

- `~/.shelp/config.json`, created with mode `0600` (owner read/write only). The
  directory itself is `0700`. `SHELP_CONFIG_DIR` moves both. Keys of every
  named profile live in that one file.
- The key is only sent to the API URL you configured, in the `Authorization`
  header. `--debug` output redacts it, and `shelp config show` masks it.
- If you would rather not store it on disk, set `SHELP_API_KEY` (together with
  `SHELP_URL` and `SHELP_MODEL`) in your environment or a secret manager;
  environment values override the file and are never written to it.
- shelp warns when the API URL uses `http://` with a non-local host, since the
  key then travels in cleartext.

## What shelp records

- `~/.shelp/history.jsonl`, created with mode `0600` in the `0700` config
  directory (`SHELP_CONFIG_DIR` moves both). It holds one JSON object per
  answered query: the time, the query text, the generated or selected commands,
  whether they ran, the exit code of the run and the profile name. Only the
  newest 1000 entries are kept.
- Nothing is sent anywhere: the file is local, and API keys are never written to
  it. It is still cleartext, so any secret you typed into a query or a command
  ends up on disk.
- `--no-history` skips one query and `SHELP_NO_HISTORY=1` disables recording
  altogether; `shelp history clear` deletes the file.

## What shelp executes

Generated commands run on your machine, in your shell, as you, with your
environment and privileges, after you confirm them. `--yes` skips that
confirmation, so only use it for requests whose outcome you can predict.

The blocklist (`internal/safety`) is best-effort: a set of regular expressions
that catches well-known catastrophic patterns such as `rm -rf /`, fork bombs,
writes to raw block devices and `curl | sh`. It is not a sandbox and can be
bypassed through variable indirection, encoding or quoting. Review commands
before running them; "not blocked" does not mean "safe".
