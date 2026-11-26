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

> **Status: early release.** The core checks work end to end on JS/TS and Go
> projects. Remote providers (Vercel, GitHub Actions, AWS) are not built yet.
> See the [roadmap](#roadmap).

---

## Why

Every team has hit at least one of these:

- A new teammate clones the repo and the app crashes because `.env.example` is out of date.
- Someone adds `process.env.NEW_KEY` in a PR but never documents it.
- A deploy fails because a key exists locally but was never added to Vercel, GitHub Actions, or AWS.

`envcheck` finds these problems in seconds, locally or in CI.

## Features

- Compare `.env.example` with `.env`: keys missing from `.env`, and keys with empty values
- Scan source code for variables that are used but never declared in `.env.example`
  - JS/TS (`.js .jsx .mjs .cjs .ts .tsx .mts .cts .vue .svelte .astro`):
    `process.env.X`, `process.env["X"]`, `import.meta.env.X`, and
    destructuring like `const { X, Y } = process.env`
  - Go: `os.Getenv("X")`, `os.LookupEnv("X")`
- Flag declared variables that no code uses
- Ignores comments, string literals, dependencies, and build output, so the
  report contains only reads your code actually does
- `--ci` mode that exits non-zero on problems
- Human-readable and JSON output
- Optional `.envcheck.yaml` config

**Planned**

- Remote providers that compare key names only, never values: Vercel, GitHub Actions, AWS SSM / Secrets Manager
- Monorepo support (per-app `.env.example` files)
- Python, Ruby, and Rust scanners
- An `npx envcheck` wrapper around the binary

## Install

envcheck is a single static binary with no runtime dependencies. Pick
whichever method suits you.

### Install script (macOS, Linux)

```bash
curl -sSfL https://raw.githubusercontent.com/geekreflex/envcheck/main/install.sh | sh
```

The script downloads the latest release for your OS and CPU and checks it
against the release's SHA-256 checksums. It installs to `/usr/local/bin` if
that's writable, otherwise to `~/.local/bin`. You can change both:

```bash
curl -sSfL https://raw.githubusercontent.com/geekreflex/envcheck/main/install.sh \
  | ENVCHECK_VERSION=v0.1.0 ENVCHECK_INSTALL_DIR="$HOME/bin" sh
```

### Homebrew (macOS)

```bash
brew install geekreflex/tap/envcheck
```

### Go

Requires Go 1.26 or newer:

```bash
go install github.com/geekreflex/envcheck@latest
```

The binary lands in `$(go env GOPATH)/bin`, so make sure that directory is on
your `PATH`.

### Prebuilt binaries (any OS, including Windows)

Download the archive for your platform from the
[releases page](https://github.com/geekreflex/envcheck/releases). Builds are
available for Linux, macOS, and Windows on amd64 and arm64, each with
`checksums.txt` and an SBOM. Extract it and put `envcheck` (or
`envcheck.exe`) somewhere on your `PATH`.

On Windows, using PowerShell:

```powershell
$v = "0.1.0"   # the release you want, without the leading v
Invoke-WebRequest "https://github.com/geekreflex/envcheck/releases/download/v$v/envcheck_${v}_windows_amd64.zip" -OutFile envcheck.zip
Expand-Archive envcheck.zip -DestinationPath "$env:USERPROFILE\bin\envcheck"
# then add $env:USERPROFILE\bin\envcheck to your PATH
```

To verify a download yourself:

```bash
sha256sum --ignore-missing -c checksums.txt     # Linux
shasum -a 256 --ignore-missing -c checksums.txt # macOS
```

### From source

```bash
git clone https://github.com/geekreflex/envcheck
cd envcheck
make build        # ./dist/envcheck
make install      # or install into $(go env GOPATH)/bin
```

Check the install with:

```bash
envcheck --version
```

## Quick start

Run it from the root of a project that has a `.env.example`:

```bash
cd my-app
envcheck
```

```
Missing in .env (declared in .env.example)
  ✗ STRIPE_WEBHOOK_SECRET  .env.example:2

Used in code but not declared in .env.example
  ✗ REDIS_URL   src/lib/cache.ts:4
  ✗ SENTRY_DSN  cmd/api/main.go:22

Declared in .env.example but never used in code
  ! LEGACY_API_KEY  .env.example:3

Empty values in .env
  ! NEXT_PUBLIC_API_URL  .env:2

5 problems found
```

Fix what it reports and run it again until you see `✓ No problems found`.
Then add `envcheck --ci` to your pipeline so the files stay in sync (see
[GitHub Actions](#github-actions)).

## Usage

```bash
# Check the current directory
envcheck

# Check another directory
envcheck ./services/api

# Point at specific files
envcheck --example .env.sample --env .env.local

# Fail the build if anything is wrong
envcheck --ci

# Machine-readable output
envcheck --json
```

### What it reports

| Problem | Meaning | Typical fix |
| --- | --- | --- |
| ✗ Missing in `.env` | Declared in `.env.example` but not set in your `.env` | Add it to `.env` |
| ✗ Used in code but not declared | Your code reads it but `.env.example` doesn't list it | Document it in `.env.example` |
| ! Declared but never used | Listed in `.env.example` but no scanned code reads it | Remove it, or add it to `ignore` if something outside the code reads it |
| ! Empty value | Set in `.env` with nothing after `=` | Fill it in |

Notes:

- `.env.example` is required. `.env` is optional. When it's missing, which is
  normal in CI, envcheck prints a note to stderr and skips the "missing" and
  "empty" checks.
- Keys in `.env` that aren't in `.env.example` are not reported, so your own
  local overrides are left alone.
- Code checks run only if envcheck found at least one supported source file.
  In a project written in an unsupported language, nothing is reported as
  unused.
- Variables set by the OS, shell, or CI (`NODE_ENV`, `CI`, `HOME`, `PATH`,
  `PWD`, `USER`, `SHELL`, `TMPDIR`, `TERM`, `TZ`, `LANG`) are ignored by default.
- A key is only detected when the code names it literally.
  `process.env[name]` and `os.Getenv(prefix + "_URL")` are skipped rather
  than guessed.

### Flags

| Flag | Description | Default |
| --- | --- | --- |
| `--example` | Path to the example env file | `.env.example` in the checked directory |
| `--env` | Path to the local env file. If given, the file must exist. | `.env` in the checked directory |
| `--ci` | Exit `1` when problems are found | `false` |
| `--json` | Emit machine-readable JSON | `false` |
| `--config` | Path to `.envcheck.yaml` | `.envcheck.yaml` in the checked directory, if present |
| `--version` | Print version, commit, and build date | |

`--example` and `--env` are relative to your current directory. The `example`
and `env` settings in the config file are relative to the checked directory.

### Exit codes

These are a stable contract. CI pipelines rely on them.

| Code | Meaning |
| --- | --- |
| `0` | Ran successfully, nothing to fix |
| `1` | Ran successfully, problems found. Only with `--ci`; without it a report is informational and still exits `0`. |
| `2` | `envcheck` itself failed: bad flags, unreadable or malformed files, malformed config |

### JSON output

`--json` prints a single object with a `problems` array. The array is empty
when everything is clean. Each problem has a `kind`, the variable's `key`, and
where it was found as a `file` and 1-indexed `line`. It never includes a value.

```json
{
  "problems": [
    { "kind": "missing_in_env", "key": "STRIPE_WEBHOOK_SECRET", "file": ".env.example", "line": 2 },
    { "kind": "undeclared_in_example", "key": "REDIS_URL", "file": "src/lib/cache.ts", "line": 4 },
    { "kind": "unused", "key": "LEGACY_API_KEY", "file": ".env.example", "line": 3 },
    { "kind": "empty_value", "key": "NEXT_PUBLIC_API_URL", "file": ".env", "line": 2 }
  ]
}
```

`kind` is one of `missing_in_env`, `undeclared_in_example`, `unused`, or
`empty_value`.

### Supported `.env` syntax

```bash
# comments and blank lines are ignored
PORT=8080
export API_URL=https://api.example.com   # export prefix and inline comments
GREETING="hello\nworld"                  # double quotes: \n \t \" \\ escapes
PATTERN='literal $value \n'              # single quotes: taken literally
CERT="-----BEGIN CERTIFICATE-----
...
-----END CERTIFICATE-----"               # quoted values may span lines
```

`${OTHER}` references are kept as written, not expanded. A line envcheck
can't parse fails the run with the file and line number. The line's
contents are never printed.

## GitHub Actions

The repository doesn't contain a `.env` file in CI, so `envcheck --ci` there
checks that `.env.example` covers every variable your code reads, and that
every variable it lists is still used:

```yaml
name: env check
on: [pull_request]

jobs:
  envcheck:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v7
      - name: Install envcheck
        run: |
          curl -sSfL https://raw.githubusercontent.com/geekreflex/envcheck/main/install.sh \
            | ENVCHECK_INSTALL_DIR="$HOME/.local/bin" sh
          echo "$HOME/.local/bin" >> "$GITHUB_PATH"
      - run: envcheck --ci
```

Or with Go:

```yaml
      - uses: actions/checkout@v7
      - uses: actions/setup-go@v7
        with:
          go-version: stable
      - run: go install github.com/geekreflex/envcheck@latest
      - run: envcheck --ci
```

The same works in any CI system: install the binary, then run
`envcheck --ci`.

## Configuration

Add an optional `.envcheck.yaml` (or `.envcheck.yml`) to the checked
directory, or pass `--config path`. Flags override it, and every key is
optional.

```yaml
example: .env.example   # relative to the checked directory
env: .env.local
scan:
  include: ["src/**", "cmd/**"]   # replaces the default ("**", everything)
  exclude: ["**/*.test.ts"]       # added to the default excludes
ignore:                           # added to the default ignore list
  - NEXT_PUBLIC_*
  - VERCEL_URL
```

- **Patterns** are matched against paths relative to the checked directory,
  using `/` as the separator. `*` matches within one path segment, and `**`
  matches any number of segments.
- **`scan.exclude` defaults:** dependencies and build output are always
  excluded: `**/node_modules/**`, `**/.git/**`, `**/vendor/**`, `**/dist/**`,
  `**/build/**`, `**/.next/**`, `**/.nuxt/**`, `**/.output/**`,
  `**/.svelte-kit/**`, `**/coverage/**`, `**/*.min.js`. Files over 2 MB are
  skipped.
- **`ignore`:** takes exact names or globs such as `NEXT_PUBLIC_*`. Ignored
  variables are never reported.
- **Mistakes fail fast:** unknown keys, wrong types, and malformed patterns
  fail the run with exit code `2`, so a typo can't be silently ignored.

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
- [x] `.env` parser and diff
- [x] JS/TS and Go code scanners
- [x] `--ci` and `--json` output
- [x] Config file support
- [x] Install script, GoReleaser archives, Homebrew tap
- [ ] First tagged release
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
