---
name: add-provider
description: Use when adding a remote environment provider to envcheck — Vercel, GitHub Actions secrets, AWS SSM or Secrets Manager, Netlify, Fly, Doppler, or similar — or when changing how envcheck talks to one. Encodes the names-only privacy rule, auth handling, and failure semantics that every provider must follow.
---

# Adding a remote provider

A provider answers one question: **which variable names are set in this remote
environment?** It compares key names against the local `.env.example` so a
deploy does not fail on a key nobody added to the dashboard.

## The hard rule

**A provider requests key names only. Never values.**

This is non-negotiable and is the reason the feature is safe to ship. Concretely:

- Prefer an API endpoint that returns names without values. Most platforms
  have one (list/metadata endpoints).
- If the only available endpoint returns values too, discard them at the
  boundary — decode into a struct that has no value field at all, so a value
  cannot be carried onward even by mistake. Do not decode into a map and
  "remember not to use" the values.
- Never log, cache, or persist a response body that contains values.
- Never write a provider that *sets* or *uploads* variables. envcheck is a
  read-only diagnostic tool.

A provider that cannot list names without values should be declined and the
reason documented, rather than shipped with a caveat.

## Shape

Providers live in `internal/provider/<name>.go` behind one interface:

```go
// Provider reports the variable names configured in a remote environment.
type Provider interface {
    // Name identifies the provider, e.g. "vercel".
    Name() string
    // Keys returns the variable names set remotely. It never returns values.
    Keys(ctx context.Context, env string) ([]string, error)
}
```

Return `[]string` — not a map, not a struct with a value field. The signature
itself should make leaking impossible.

## Requirements

- **`context.Context` first**, honoured for cancellation and timeouts. Set a
  default timeout; CI must never hang.
- **Credentials come from the environment only** (`VERCEL_TOKEN`,
  `GITHUB_TOKEN`, standard AWS credential chain). Never a flag — flags land in
  shell history and CI logs. Never a config-file field.
- **Missing credentials is a clear, actionable error**, not a crash and not a
  silent skip: name the variable the user must set.
- **Redact on failure.** Error messages must not include tokens, URLs with
  embedded credentials, or response bodies. Wrap with the operation and the
  provider name.
- **`net/http` with an explicit client and timeout.** No SDK dependency unless
  the platform makes raw HTTP impractical; AWS is the likely exception. Every
  new dependency is a supply-chain liability for a tool that handles secrets.
- **Close response bodies** — `bodyclose` is enabled in the linter.
- **Offline by default.** Providers run only when explicitly requested by flag
  or config. A bare `envcheck` must never make a network call; that property is
  worth a test.

## Steps

1. Define the provider in `internal/provider/<name>.go`.
2. Add a `check.Kind` for remote mismatches (e.g. `MissingInRemote`) and
   handle it in `internal/report`. Keep `Problem` name-only.
3. Wire a flag in `cmd/root.go` and a config section in `internal/config`.
4. Test against a `httptest.Server` fixture — never the live API. Include a
   fixture whose payload *does* contain values, and assert the decoded result
   carries none.
5. Test the no-credentials path and the timeout path.
6. Document it in the README under Features, and note the token variable.
7. Run `make check`.

## Verify

```bash
make check
go test ./internal/provider/ -race -v
```
