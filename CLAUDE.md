# envcheck

A single-binary Go CLI that reports environment variables which are missing,
unused, or empty across `.env.example`, `.env`, and the source code that reads
them. Built to run locally and in CI.

Module: `github.com/geekreflex/envcheck`

## The invariant

**envcheck reports variable names. It never reads, prints, logs, or transmits
a variable's value.**

This is the product's core promise, not a nice-to-have. It constrains code:

- `check.Problem` carries `Kind`, `Key`, `File`, `Line` — never a value. The
  field set is pinned by `TestProblemCarriesNoSecretValues`; if you add a
  field there, prove it cannot carry a secret.
- `envfile.Var` does hold `Value`, because detecting empty values requires it.
  That value must never escape the `internal/envfile` and `internal/check`
  boundary into a report, log line, or error message.
- Error messages must not interpolate values. `fmt.Errorf("parsing %s:%d", path, line)` — never the value.
- Future remote providers (Vercel, GitHub Actions, AWS) request key names
  only. No provider may fetch secret material.

When in doubt, leak nothing.

## Architecture

Data flows one way: parse and scan, then compare, then render.

```
main.go            process entry; exits with cmd.Execute()'s code
cmd/               cobra wiring, flag parsing, exit-code policy
internal/
  buildinfo/       version metadata (ldflags, with debug.BuildInfo fallback)
  config/          .envcheck.yaml loading and defaults
  envfile/         .env parsing -> []Var
  scanner/         source code -> []Ref (one implementation per language)
  check/           compares example + env + refs -> []Problem
  report/          []Problem -> text or JSON
```

Dependency rule: `check` depends on `envfile` and `scanner`; `report` depends
on `check`. Nothing in `internal/` imports `cmd`. Keep it acyclic.

Everything is under `internal/` deliberately — envcheck is a tool, not a
library, so there is no public API surface to keep stable yet.

## Commands

```bash
make check     # tidy + vet + lint + test + build. Run before pushing.
make test      # go test -race -shuffle=on ./...
make lint      # go vet + golangci-lint (version pinned in the Makefile)
make build     # ./dist/envcheck with version metadata injected
make cover     # coverage report at coverage.html
make snapshot  # local GoReleaser build, nothing published
```

`make lint` and `make snapshot` fetch their tools via `go run`, so there is
nothing to install first.

## Conventions

**Exit codes are a public contract.** `0` clean, `1` problems found (with
`--ci`), `2` envcheck itself failed. CI pipelines branch on these, so treat a
change as breaking. A completed check that found problems is not an error:
return `*problemsError` so `classify` maps it to `1` and does not print it as
a crash.

**Errors.** Wrap with `%w` and context about *where*, never *what*:
`fmt.Errorf("reading %s: %w", path, err)`. Lowercase, no trailing
punctuation. Return errors; do not log and continue.

**Testing.** Table-driven with named subtests, standard library only — no
assertion framework. Tests run with `-race -shuffle=on`, so no ordering
dependence between cases. Prefer testing behaviour through the exported
surface. `cmd.run(args, stdout, stderr)` takes its streams and args as
parameters precisely so the CLI is testable without touching `os.Args`.

**No global state.** Flags live in an `options` struct built per command
construction, not in package-level vars.

**Adding language support** is the most common extension. See
`.claude/skills/add-scanner/SKILL.md`; implement `scanner.Scanner`, register
in `All()`, and extend the table in `scanner_test.go`.

## Gotchas

- Scanners must report at most one claim per file path — `Supports` must be
  mutually exclusive across scanners, and a test enforces it.
- `.gitignore` ignores `.env` and `.env.*` but re-includes `.env.example`. Do
  not commit a real `.env`, including in test fixtures; put fixtures under
  `testdata/`.
- The default scan excludes `node_modules/**`, `.git/**`, and `vendor/**`.
  Scanning them is slow and produces false positives.
- `buildinfo` variables are set by `-ldflags`; they are intentionally
  unexported and lowercase. Do not rename without updating the Makefile and
  `.goreleaser.yaml` together.

## Current state

The CLI skeleton, exit-code policy, build metadata, and package boundaries
are in place. The parsing, scanning, comparison, and reporting bodies are
stubs marked `TODO`. `envcheck` currently exits `2` with "not implemented
yet". See the Roadmap in README.md for the intended order of work.
