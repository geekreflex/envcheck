# envcheck

[![CI](https://github.com/geekreflex/envcheck/actions/workflows/ci.yml/badge.svg)](https://github.com/geekreflex/envcheck/actions/workflows/ci.yml)
[![Go Report Card](https://goreportcard.com/badge/github.com/geekreflex/envcheck)](https://goreportcard.com/report/github.com/geekreflex/envcheck)
[![Go version](https://img.shields.io/github/go-mod/go-version/geekreflex/envcheck)](go.mod)
[![License: MIT](https://img.shields.io/badge/license-MIT-blue.svg)](LICENSE)

Catch missing environment variables before they break production.

`envcheck` compares your `.env.example`, your local `.env`, and the variables
your code actually uses, then tells you what's missing, unused, or empty. It
runs as a single binary, needs no signup, and never reads or prints secret
values.

> **Status: pre-release, not yet usable.** The package layout, CLI surface,
> and release pipeline are in place; the parsing, scanning, and reporting
> logic is still stubbed, so the binary currently exits `2` with
> "not implemented yet". Nothing is published to Homebrew or tagged yet.
> Everything below describes the v1 target. Follow the
> [roadmap](#roadmap) for progress.

---

## Why

Every team has hit at least one of these:

- A new teammate clones the repo and the app crashes because `.env.example` is out of date.
- Someone adds `process.env.NEW_KEY` in a PR but never documents it.
- A deploy fails because a key exists locally but was never added to Vercel, GitHub Actions, or AWS.

`envcheck` finds these problems in seconds, locally or in CI.

## Features

**v1**

- Diff `.env.example` against `.env`: missing keys, extra keys, empty values
- Scan source code for variables that are used but never declared
  - JS/TS: `process.env.X`, `process.env["X"]`, `import.meta.env.X`
  - Go: `os.Getenv("X")`, `os.LookupEnv("X")`
- Flag declared variables that no code uses
- `--ci` mode that exits non-zero on problems
- Human-readable and JSON output

**Planned**

- Remote providers that compare key names only, never values: Vercel, GitHub Actions, AWS SSM / Secrets Manager
- Monorepo support (per-app `.env.example` files)
- Python, Ruby, and Rust scanners
- An `npx envcheck` wrapper around the binary

## Install

Once v1 is tagged:

```bash
# Homebrew (macOS)
brew install geekreflex/tap/envcheck

# Go
go install github.com/geekreflex/envcheck@latest
```

Prebuilt binaries for Linux, macOS, and Windows (amd64 and arm64) will be
attached to each [release](https://github.com/geekreflex/envcheck/releases)
with checksums and an SBOM.

To build from source today:

```bash
git clone https://github.com/geekreflex/envcheck
cd envcheck
make build        # ./dist/envcheck
```

## Usage

```bash
# Check the current directory
envcheck

# Check a specific directory
envcheck ./services/api

# Point at specific files
envcheck --example .env.example --env .env.local

# Fail the build if anything is wrong
envcheck --ci

# Machine-readable output
envcheck --json
```

Example output:

```
envcheck v0.1.0

Missing in .env (declared in .env.example)
  ✗ STRIPE_WEBHOOK_SECRET

Used in code but not declared in .env.example
  ✗ REDIS_URL          src/lib/cache.ts:4
  ✗ SENTRY_DSN         cmd/api/main.go:22

Declared but never used
  ! LEGACY_API_KEY

Empty values
  ! NEXT_PUBLIC_API_URL

4 problems found
```

### Flags

| Flag | Description | Default |
| --- | --- | --- |
| `--example` | Path to the example env file | `.env.example` |
| `--env` | Path to the local env file | `.env` |
| `--ci` | Exit non-zero when problems are found | `false` |
| `--json` | Emit machine-readable JSON | `false` |
| `--config` | Path to `.envcheck.yaml` | discovered from the project root |
| `--version` | Print version, commit, and build date | |

### Exit codes

These are a stable contract — pipelines branch on them.

| Code | Meaning |
| --- | --- |
| `0` | Ran successfully, nothing to fix |
| `1` | Ran successfully, problems found. Only with `--ci`; without it a report is informational and still exits `0`. |
| `2` | `envcheck` itself failed: bad flags, unreadable files, malformed config |

### JSON output

`--json` emits one object with a `problems` array. Each problem carries a
`kind`, the variable `key`, and — where the problem came from source code —
the `file` and 1-indexed `line`. It never carries a value.

```json
{
  "problems": [
    { "kind": "missing_in_env", "key": "STRIPE_WEBHOOK_SECRET" },
    { "kind": "undeclared_in_example", "key": "REDIS_URL", "file": "src/lib/cache.ts", "line": 4 },
    { "kind": "unused", "key": "LEGACY_API_KEY" },
    { "kind": "empty_value", "key": "NEXT_PUBLIC_API_URL" }
  ]
}
```

`kind` is one of `missing_in_env`, `undeclared_in_example`, `unused`, or
`empty_value`.

## GitHub Actions

```yaml
name: env check
on: [pull_request]

jobs:
  envcheck:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v7
      - uses: geekreflex/envcheck-action@v1
        with:
          args: --ci
```

Until the action is published, run the binary directly:

```yaml
      - uses: actions/checkout@v7
      - uses: actions/setup-go@v7
        with:
          go-version: stable
      - run: go install github.com/geekreflex/envcheck@latest
      - run: envcheck --ci
```

## Configuration

Optional `.envcheck.yaml` in the project root. Flags override it.

```yaml
example: .env.example
env: .env
scan:
  include: ["src/**", "cmd/**", "internal/**"]
  exclude: ["**/*.test.ts", "node_modules/**"]
ignore:
  - NODE_ENV
  - CI
```

`scan.include` defaults to everything; `scan.exclude` defaults to
`node_modules/**`, `.git/**`, and `vendor/**`. Variables listed under
`ignore` are never reported.

## Privacy

`envcheck` works with variable *names*. It never prints, logs, or sends
secret values anywhere.

- Values are read only to detect whether a variable is empty, and never leave
  that boundary — not into reports, not into error messages.
- A bare `envcheck` run makes no network requests. Remote providers are
  opt-in.
- Remote provider checks will request key names only, and will never upload
  or modify your configuration.

Anything that breaks this is treated as a security vulnerability, not a bug.
See [SECURITY.md](SECURITY.md).

## Roadmap

- [x] CLI skeleton, exit-code contract, release pipeline
- [ ] `.env` parser and diff
- [ ] JS/TS and Go code scanners
- [ ] `--ci` and `--json` output
- [ ] Config file support
- [ ] GoReleaser + Homebrew tap
- [ ] GitHub Action
- [ ] Vercel provider
- [ ] GitHub Actions secrets provider
- [ ] AWS SSM / Secrets Manager provider

## Development

Requires Go (the version in [`go.mod`](go.mod)) and `make`. The linter and
release tooling are fetched on demand, so there is nothing else to install.

```bash
make check    # tidy + vet + lint + test + build — run before pushing
make test     # go test -race -shuffle=on ./...
make cover    # coverage report at coverage.html
make help     # all targets
```

Architecture, package boundaries, and code conventions are documented in
[CLAUDE.md](CLAUDE.md).

## Contributing

Issues and PRs are welcome. Please open an issue first to discuss larger
changes. See [CONTRIBUTING.md](CONTRIBUTING.md).

## License

[MIT](LICENSE) © Jerry Nwosu
