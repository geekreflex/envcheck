---
name: release
description: Use when cutting an envcheck release, tagging a version, publishing to Homebrew, or debugging a failed release build — e.g. "release v0.2.0", "cut a release", "the release workflow failed", "bump the version". Covers the tag-driven GoReleaser flow, required secrets, and verification.
---

# Cutting a release

Releases are tag-driven. Pushing a `v*` tag triggers
`.github/workflows/release.yml`, which runs GoReleaser to build the binaries,
publish the GitHub release, and update the Homebrew tap. Nothing is published
by hand.

## Versioning

Semantic versioning, `v`-prefixed: `v0.2.0`. While the major version is `0`,
breaking changes bump the minor.

**Treat these as breaking:**

- Exit-code meanings (`0` clean, `1` problems, `2` envcheck failed). CI
  pipelines branch on them.
- The `--json` output shape — field names, nesting, `Kind` string values.
  Consumers parse it.
- Flag names and their defaults.
- `.envcheck.yaml` keys.

## Before tagging

```bash
make check      # tidy + vet + lint + test + build
make snapshot   # full GoReleaser build locally, publishing nothing
```

Then confirm:

1. `make snapshot` produced archives for all six targets (linux/darwin/windows
   × amd64/arm64) under `dist/`.
2. `./dist/envcheck --version` reports the injected version, commit, and date
   — not `dev`. If it says `dev`, the `-ldflags` paths in `.goreleaser.yaml`
   have drifted from the variable names in `internal/buildinfo`.
3. The README's documented flags and output match reality.
4. `CHANGELOG`-worthy commits use conventional prefixes (`feat:`, `fix:`,
   `perf:`) — the release notes are grouped by them, and `docs:`, `test:`,
   `chore:`, `ci:`, `style:` are filtered out.

## Cutting it

```bash
git switch main && git pull
make check
git tag -a v0.2.0 -m "v0.2.0"
git push origin v0.2.0
```

Then watch it:

```bash
gh run watch
gh release view v0.2.0
```

## Required secrets

| Secret | Purpose |
|---|---|
| `GITHUB_TOKEN` | Provided automatically; publishes the release. |
| `HOMEBREW_TAP_TOKEN` | PAT with `contents:write` on `geekreflex/homebrew-tap`. The default token cannot push to another repository. |

If the tap repository does not exist yet, create `geekreflex/homebrew-tap`
first, or comment out the `brews` block — GoReleaser fails the whole release
when it cannot push the formula.

## When a release fails

The release is a single atomic run; a mid-way failure can leave a published
tag with no artifacts.

1. Read the failure: `gh run view --log-failed`.
2. Fix forward. Do not reuse a tag that has already been pushed — delete the
   draft release and the tag, or bump the patch version and tag again.
3. To delete a bad tag:
   ```bash
   gh release delete v0.2.0 --yes
   git push --delete origin v0.2.0
   git tag -d v0.2.0
   ```
4. Re-run `make snapshot` locally and reproduce before tagging again.

## Verify after publishing

```bash
go install github.com/geekreflex/envcheck@v0.2.0 && envcheck --version
brew install geekreflex/tap/envcheck && envcheck --version
```

Both must report the tagged version. Check the release page carries checksums
and an SBOM per archive.
