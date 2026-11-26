<!--
Thanks for contributing! A check named `pr-policy` enforces this template:

- Title: Conventional Commits, e.g. `feat(scanner): detect Deno.env.get` or
  `fix(envfile): accept tabs before =`. Add `!` before the colon for a
  breaking change: `feat(cli)!: rename --env to --env-file`.
- Every checklist box must be checked. If an item does not apply, check it
  and add "N/A" with a short reason.
- A type label is applied from the title automatically.

Comments like this one are ignored by the check, so you can leave them.
-->

## What and why

<!-- What changes, and what problem it solves. Link the issue: Fixes #123 -->

## Checklist

- [ ] `make check` passes (tidy, vet, lint, test, build)
- [ ] Tests cover the change, including the negative cases
- [ ] No environment variable **value** is read, printed, logged, or
      transmitted anywhere in this change
- [ ] User-facing changes are reflected in the README
- [ ] Exit codes, `--json` output, flag names, and config keys are unchanged
      — or the change is called out below as breaking

## Breaking changes

<!-- "None", or describe what breaks and why it is worth it. -->
