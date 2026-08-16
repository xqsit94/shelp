# Contributing

Thanks for helping out. Bug reports, small fixes and focused features are all
welcome.

## Prerequisites

- Go, at the version declared in [`go.mod`](go.mod) (or newer)
- A GitHub account for pull requests

No other tooling is required; everything below uses the Go toolchain.

## Build, test, lint

These are the same checks CI runs
([`.github/workflows/ci.yml`](.github/workflows/ci.yml)), so run them before
pushing:

```bash
go build ./...
go vet ./...
gofmt -l .                      # must print nothing
go test ./... -race -count=1
go mod tidy                     # must leave go.mod and go.sum unchanged
go run honnef.co/go/tools/cmd/staticcheck@latest ./...
```

The Windows build is cross-compiled only, so anything touching the executor,
signals or paths has to keep compiling for it:

```bash
GOOS=windows GOARCH=amd64 go build ./...
GOOS=windows GOARCH=amd64 go vet ./...
GOOS=linux GOARCH=amd64 go build ./...
GOOS=darwin GOARCH=arm64 go build ./...
```

`gofmt -l .` and staticcheck must both be clean; CI fails otherwise. Run
`gofmt -w .` to fix formatting.

To try your build without touching your own configuration, point it at a scratch
directory:

```bash
go build -o /tmp/shelp .
SHELP_CONFIG_DIR=/tmp/shelp-config /tmp/shelp -p "list files"
```

## Code style

- Idiomatic Go; keep changes small and focused.
- Comments only where the logic is genuinely non-obvious - no restating the code.
- Tests are table-driven and use the standard library (`testing`, `httptest`).
  Bubbletea models are tested by calling `Update` with `tea.KeyMsg` values.
- Never call a real AI provider from a test; use `httptest.Server`.

## Commits and pull requests

- [Conventional Commits](https://www.conventionalcommits.org/) for commit
  subjects: `feat:`, `fix:`, `docs:`, `refactor:`, `test:`, `chore:`,
  `build:`, `ci:`, with an optional scope (`fix(safety): …`).
- Open pull requests against `main`, describe what changed and why, and mention
  any user-visible behaviour change so it can go into `CHANGELOG.md` under
  `## [Unreleased]`.
