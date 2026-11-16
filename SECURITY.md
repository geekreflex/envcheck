# Security Policy

## Reporting a vulnerability

Please report security issues **privately**, not as a public issue:

- Open a [private advisory](https://github.com/geekreflex/envcheck/security/advisories/new) (preferred), or
- Email the maintainer listed on the GitHub profile for
  [@geekreflex](https://github.com/geekreflex).

Please include the envcheck version, the platform, and the smallest
reproduction you can manage. **Redact real secret values** — variable names
are enough to describe any envcheck bug.

You can expect an initial response within a few days. Since envcheck is
maintained outside of business hours, please allow reasonable time before
disclosing publicly. You will be credited in the advisory unless you prefer
otherwise.

## Supported versions

envcheck is pre-1.0. Fixes land on the latest minor release; there are no
backports yet.

## What counts as a vulnerability

envcheck's central promise is that it handles environment variable **names**,
never **values**. Anything that breaks that promise is a security issue, even
if it looks cosmetic:

- A secret value appearing in output, a log line, an error message, or a
  stack trace
- A secret value written to disk, a cache, or a temporary file
- Any network request carrying a secret value, or a remote provider
  requesting values rather than names
- A credential (`VERCEL_TOKEN`, `GITHUB_TOKEN`, AWS credentials) appearing in
  output or being sent to an unintended host
- Path traversal or arbitrary file reads driven by `.envcheck.yaml` or flags
- Code execution triggered by scanning a hostile source tree or config file

Please report any of these privately.

## What is not a vulnerability

- envcheck reading your `.env` file. That is its job; it runs locally with
  your own permissions.
- A variable reported in error (false positive) or missed (false negative).
  Those are ordinary bugs — please open an issue.
- Findings in a dependency with no demonstrated impact on envcheck. CI runs
  `govulncheck` on every change, so these are usually already visible.

## Design commitments

- Static binaries, built with `CGO_ENABLED=0` and `-trimpath`.
- A deliberately small dependency tree. Every dependency is a supply-chain
  liability for a tool that sits next to secrets, so new ones need
  justification.
- `govulncheck` and CodeQL run in CI on every change, plus weekly.
- Releases are built by GitHub Actions from a tag, never from a laptop, and
  ship checksums and an SBOM per archive.
- A bare `envcheck` run makes no network requests. Remote providers are
  opt-in by flag or config.
