# Contributing to envcheck

Thanks for your interest. Please open an issue before starting anything
substantial, so we can agree on the approach before you write code.

## The one rule

**envcheck reports variable names. It never reads, prints, logs, or
transmits a variable's value.**

This is the reason the tool is safe to run against production config, and no
feature is worth weakening it. A change that would put a secret value into
output, a log line, an error message, or a network request will be declined.
`internal/envfile` reads values because detecting empty ones requires it;
those values must not escape that boundary.

## Setup

You need Go (the version in `go.mod`) and `make`. Nothing else — the linter
and release tooling are fetched on demand by `go run`.

```bash
git clone https://github.com/geekreflex/envcheck
cd envcheck
make check
```

## Workflow

```bash
make test     # go test -race -shuffle=on ./...
make lint     # go vet + golangci-lint at the pinned version
make build    # ./dist/envcheck with version metadata injected
make check    # everything CI runs, in one command
make help     # all targets
```

Run `make check` before pushing. CI runs the same things, plus a test matrix
across Linux, macOS, and Windows and a `govulncheck` scan.

## Code conventions

They are documented in [CLAUDE.md](CLAUDE.md), which doubles as the
architecture guide — the package layout, the dependency rules, and the error
and testing conventions all live there. In short:

- Table-driven tests with named subtests, standard library only. No
  assertion framework.
- Wrap errors with `%w` and context about *where*, never *what*. Lowercase,
  no trailing punctuation.
- No package-level mutable state.
- Exported identifiers get doc comments.

## Adding a language scanner

This is the most common contribution. Implement `scanner.Scanner`, register
it in `All()`, and extend the test table. Be conservative: skip dynamic keys
like `process.env[name]` rather than guessing, because a false positive
teaches people to ignore the output.

`.claude/skills/add-scanner/SKILL.md` walks through it, including the syntax
forms per language and the false positives each scanner must reject.

## Adding a remote provider

Providers compare key *names* against a remote environment. They must request
names only; if an API returns values too, decode into a type with no value
field so the values cannot be carried onward. Credentials come from the
environment, never from a flag. See
`.claude/skills/add-provider/SKILL.md`.

## What counts as a breaking change

Treat these as breaking, because people automate against them:

- Exit codes: `0` clean, `1` problems found (with `--ci`), `2` envcheck failed
- The `--json` output shape, including `Kind` string values
- Flag names and defaults
- `.envcheck.yaml` keys

## Commits and PRs

Conventional prefixes, because release notes are generated from them:
`feat:`, `fix:`, `perf:`, `docs:`, `test:`, `refactor:`, `chore:`, `ci:`.
Only the first three appear in the changelog.

Keep PRs focused on one change. Fill in the template, especially the privacy
checkbox.

## Security

Report vulnerabilities privately. See [SECURITY.md](SECURITY.md).
