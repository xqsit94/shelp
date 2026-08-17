# QA Report — shelp CLI/TUI

| | |
|---|---|
| **Date** | 2026-08-17 |
| **Target** | `shelp` @ `b45d2f8` (main, post-PR #2) |
| **Mode** | Full, report-only (no fixes applied) |
| **Harness** | pty + `pyte` terminal emulator, fake local provider on `127.0.0.1:18940` |
| **Duration** | ~35 min |
| **Surfaces visited** | 11 (command list, single-command menu, edit, refine, exec + summary, spinner, error, wizard, `config`, `history`, `init`) |
| **Frames captured** | 10 evidence files in `frames/` |
| **Health score** | **60 / 100** |

> `/qa-only` is browser-based and expects a URL. shelp is a terminal TUI, so the browser was replaced with a
> pty harness: the real binary runs under a pseudo-terminal, `pyte` renders the screen, and each "screenshot"
> is a captured frame plus a per-cell style dump (the only way to prove what is colour and what is text).
> No real AI provider was contacted and no real user config was touched (`SHELP_CONFIG_DIR` was redirected).

## Top 3 things to fix

1. **ISSUE-001** — the focused row is indicated by colour alone. Under `NO_COLOR` the list is unusable.
2. **ISSUE-002** — commands are silently cut off at the screen edge with no ellipsis. You approve a prefix.
3. **ISSUE-003** — at 80 columns the footer clips and `enter: execute` / `q: quit` are never shown.

## Severity summary

| Severity | Count |
|---|---|
| Critical | 1 |
| High | 4 |
| Medium | 5 |
| Low | 4 |
| **Total** | **14** |

## Category scores

| Category | Weight | Score |
|---|---:|---:|
| Output hygiene (console) | 15% | 70 |
| Reachable keybindings (links) | 10% | 70 |
| Visual | 10% | 51 |
| Functional | 20% | 29 |
| UX | 15% | 59 |
| Performance | 10% | 100 |
| Content | 5% | 94 |
| Accessibility | 15% | 52 |

---

## ISSUE-001 — Focused row is indicated by colour alone [Critical · Accessibility/Functional]

Evidence: `frames/F1-nocolor-cursor-invisible.txt`

In the multi-select list, the only difference between the focused row and every other row is the **colour of
the 2-character tree branch**: `├─` rendered `fg=7c3aed` (violet, bold) when focused, `fg=4b5563` (grey)
otherwise. The row text, the checkbox and the indentation are identical.

Style dump, cursor on item 1 vs item 2 (same run, one `↓` apart):

```
cursor on item 1:  L02: col0 '├─' fg=7c3aed bold ...   L04: col0 '├─' fg=4b5563 ...
cursor on item 2:  L02: col0 '├─' fg=4b5563 ...        L04: col0 '├─' fg=7c3aed bold ...
```

With `NO_COLOR=1` the two frames are **byte-for-byte identical**:

```
Generated Commands (3)
├─ [●] docker compose down
│     ● safe — Stops and removes the running containers
├─ [●] docker compose pull        <-- cursor may or may not be here. No way to tell.
│     ● safe — Fetches the newest images for every service
└─ [●] docker compose up -d
```

**Repro**
1. `NO_COLOR=1 shelp "restart the docker stack"`
2. Press `↓`.
3. Compare the screen before and after — nothing changes.
4. Press `space` or `e`. It acts on a row you cannot identify.

**Impact** Under `NO_COLOR`, a monochrome terminal, or for a user who cannot separate violet from grey on a
2-cell box-drawing glyph, the list cannot be operated. `space` toggles an unknown row and `e` edits an unknown
command. Selection state (`[●]`/`[○]`) survives; focus does not.

---

## ISSUE-002 — Commands are silently truncated with no ellipsis [High · Functional/Visual]

Evidence: `frames/F2-command-truncated-no-ellipsis.txt`

At 80 columns the command line is cut at the screen edge with **no truncation marker**, while the *explanation*
on the line below is correctly truncated with `…`:

```
├─ [●] find /var/log -type f -name '*.log' -mtime +30 -exec gzip -9 {} \; -print
│     ● safe — Compresses log files older than thirty days and counts them, whi…
└─ [●] awk 'BEGIN{FS=OFS=","} NR>1 {sum[$2]+=$5} END{for (k in sum) print k, sum
```

The first command actually continues `| tee /tmp/compressed-logs.txt | wc -l`; the second continues
`[k]}' data.csv`. Neither suffix is visible and nothing indicates text was removed.

**Repro** Ask for anything that produces a command longer than the terminal width, at 80 columns.

**Impact** Safety-relevant. The user confirms execution of a command whose visible text is a truncated prefix
that can read as harmless while the hidden tail changes what it does. The one element with no overflow
indicator is the one being executed; the decorative explanation gets the ellipsis.

---

## ISSUE-003 — Help footer clips; `enter: execute` and `q: quit` never shown at 80 cols [High · UX/Accessibility]

Evidence: `frames/F3-help-clipped-{60,80,100,120}cols.txt`

The footer is a fixed 106-character string that is never measured against the terminal, and it is pushed right
by a 19-column indent. What the user actually sees:

| Width | Visible footer |
|---|---|
| 60 | `↑/↓: navigate • space: toggle • a: all •` |
| 80 | `↑/↓: navigate • space: toggle • a: all • n: none • e: edit •` |
| 100 | `… • r: regenerate • ente` |
| 120+ | full string |

**Impact** At 80 columns — the default width of most terminals — the two most important bindings, `enter:
execute` and `q: quit`, are invisible. A first-time user has no on-screen way to learn how to run or leave the
list. At 100 columns the text breaks mid-word (`ente`), which reads as a rendering bug.

---

## ISSUE-004 — No viewport: long lists scroll the header and first items off-screen [High · Functional/UX]

Evidence: `frames/F4-no-viewport-12-commands.txt`

With 12 suggestions on a 24-row terminal the list renders all 26+ lines. The terminal scrolls, so the title
`Generated Commands (12)` and items 1–2 are gone from the live view and cannot be brought back while the TUI
is running:

```
├─ [●] echo step-03        <-- first visible row; items 01 and 02 and the title are above the viewport
...
└─ [●] echo step-12
  12 of 12 selected
```

**Impact** No pagination, no scroll indicator, no "N more above". Navigating up to item 1 moves the focus
(already invisible per ISSUE-001) off-screen entirely. The count in the header that would tell the user
something is missing is itself the thing that scrolled away.

---

## ISSUE-005 — First-run wizard renders skewed input boxes [High · Visual]

Evidence: `frames/F5-wizard-skewed-boxes.txt` (captured at 100x40, so this is not a clipping artifact)

Every one of the three input boxes has its top border indented 2 columns while the body and bottom border sit
at column 0:

```
  AI API URL:
  ┌───────────────────────────────────────────────────────┐   <- starts col 2
│ > https://openrouter.ai/api/v1/chat/completions       │     <- starts col 0
└───────────────────────────────────────────────────────┘     <- starts col 0
```

Confirmed in the cell dump: top border spans cols 2–58, bottom border spans cols 0–56.

**Impact** This is the first screen a new user ever sees, and all three boxes are visibly broken. At 24 rows
the welcome panel's top border also scrolls off before the form is usable.

---

## ISSUE-006 — Arbitrary indentation in edit and refine screens [Medium · Visual]

Evidence: `frames/F6-edit-refine-indentation.txt`

Edit mode places the input 10 columns in; refine mode scatters text at columns 40 and 50, while their own
titles sit at column 0 and their help at column 2:

```
Edit command
  1 of 3

          > docker compose down

Refine your request
  Original: "restart the docker stack"

                                        Add to your request (or press Enter to retry):
                                                  > add refinement here...
```

**Impact** Reads as broken layout. The indent is not a design choice — it tracks the length of the line above
it (`  1 of 3` is 8 chars → input at col 10), so it shifts as content changes.

---

## ISSUE-007 — `config test` emits raw ANSI when piped and ignores `NO_COLOR` [Medium · Functional/Output hygiene]

```
$ shelp config test | od -c | sed -n 1p
0000000    └  **  **  ─  **  **   033   [   3   8   ;   5   ;   2   3   1
$ NO_COLOR=1 shelp config test | od -c | sed -n 1p
0000000    └  **  **  ─  **  **   033   [   3   8   ;   5   ;   2   3   1   <- unchanged
$ shelp -p "restart" | od -c | sed -n 1p
0000000    d   o   c   k   e   r       c   o   m   p   o   s   e            <- correct
```

**Impact** `shelp config test > support.log` produces escape garbage. `-p` gets this right, so the behaviour is
inconsistent within the same binary. `NO_COLOR` is a widely honoured convention and is ignored on this path.

---

## ISSUE-008 — No `tea.WindowSizeMsg` handling anywhere [Medium · Functional]

Width is sampled once when a model is built. Resizing the terminal mid-session does not reflow anything, and
the truncation in ISSUE-002/003 is computed against a stale width.

---

## ISSUE-009 — A failed command reports nothing before asking whether to continue [Medium · UX]

Evidence: `frames/F9-failure-no-error-shown.txt`

```
├─ [2/3] false

Continue with next command?

[Yes]   No
```

No `✕`, no exit code, no error text. The user infers that something failed only because a question appeared.
The exit code does surface later in the summary, after the decision has already been made.

---

## ISSUE-010 — Execution progress never closes its tree and has no per-step status [Medium · Visual]

Every step renders as `├─ [n/N]`, including the last, so the tree has no terminator. No `✓`/`✕` is attached as
each command finishes; status only appears in the final summary block.

---

## ISSUE-011 — `●` means two different things on adjacent lines [Low · Visual]

`[●]` is the "selected" checkbox (green) and `●` is the "safe" risk glyph (green), one line apart, same colour:

```
├─ [●] docker compose down
│     ● safe — Stops and removes the running containers
```

---

## ISSUE-012 — `history` uses an undocumented marker [Low · Content]

```
1  just now  "restart the docker stack"
   ├─ docker compose down –
```

The trailing `–` (not executed) has no legend on screen or in `--help`.

---

## ISSUE-013 — `config profile list` contradicts `config show` [Low · Content]

`config show` prints `Configuration (profile: default)` while `config profile list` prints
`No profiles configured yet.` in the same state.

---

## ISSUE-014 — Provider errors offer no remediation [Low · UX]

```
✕ Failed to generate commands: API error (status 401): Invalid API key provided
```

A 401 is almost always a wrong or expired key; nothing points at `shelp config set key`.

---

## Verified working (no issue raised)

- **Spinner animates correctly.** An initial reading suggested a frozen spinner; sampling at 150 ms showed a
  clean 5-frame cycle (`▰▱▱▱▱ → ▰▰▰▱▱ → ▰▰▰▰▰ → ▱▱▰▰▰ → ▱▱▱▱▰`). The earlier samples were 800 ms apart, almost
  exactly one full period. **Not a bug — retracted before reporting.**
- Risk levels use distinct glyphs (`●` safe / `▲` caution / `✕` danger), not colour alone.
- Blocked commands render `[⊘]` + `danger (blocked)` and are deselected by default.
- API keys are masked in the wizard (`****…`) and in `config show` (`sk-f******************abcd`).
- `-p` output is plain text when piped and highlighted on a terminal, exactly as documented.
- Unicode (CJK, emoji, combining marks) renders at correct width.
- The single-command menu uses a `>` text cursor and its footer fits at 80 columns — proof the codebase
  already has the right pattern; the multi-select list just doesn't use it.
- Ctrl+C exits 130 from every screen tested.

## Test infrastructure

Test framework present (`go test`, 8 packages, `-race` clean). Regression tests for these findings do not
exist because they are all rendering-level; none of them would be caught by the current unit tests, which
assert on model state rather than on rendered output at a given width.

## Notes on method

All findings above are black-box: they come from driving the real binary through a pty and reading the
rendered screen. Root causes were investigated separately and are recorded in `docs/PRD-ui-revamp.md`, not
here.
