# envcheck

Catch missing environment variables before they break production.

`envcheck` compares your `.env.example`, your local `.env`, and the variables your code actually uses, then tells you what's missing, unused, or empty. It runs as a single binary, needs no signup, and never reads or prints secret values.

> Status: early development. The commands below describe the planned v1.

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

## Install

```bash
# Homebrew
brew install jerrynwosu/tap/envcheck

# Go
go install github.com/jerrynwosu/envcheck@latest

# npm (wrapper around the binary)
npx envcheck
```

## Usage

```bash
# Check the current directory
envcheck

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

## GitHub Actions

```yaml
name: env check
on: [pull_request]

jobs:
  envcheck:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v4
      - uses: jerrynwosu/envcheck-action@v1
        with:
          args: --ci
```

## Configuration

Optional `.envcheck.yaml` in the project root:

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

## Privacy

`envcheck` only works with variable names. It never prints, logs, or sends secret values anywhere. Remote provider checks will request key names only.

## Roadmap

- [ ] `.env` parser and diff
- [ ] JS/TS and Go code scanners
- [ ] `--ci` and `--json` output
- [ ] Config file support
- [ ] GoReleaser + Homebrew tap
- [ ] GitHub Action
- [ ] Vercel provider
- [ ] GitHub Actions secrets provider
- [ ] AWS SSM / Secrets Manager provider

## Contributing

Issues and PRs are welcome. Please open an issue first to discuss larger changes.

## License

MIT
