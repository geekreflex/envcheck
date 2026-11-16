---
name: add-scanner
description: Use when adding or changing support for a language's environment-variable syntax in envcheck — e.g. "add Python support", "scan Ruby files", "detect import.meta.env", "envcheck misses os.environ", "add a scanner". Covers the Scanner interface, registration, the false-positive rules, and the required test table.
---

# Adding a language scanner

A scanner turns one source file into `[]scanner.Ref` — every environment
variable the file reads, with the file and line where it reads it.

## Interface

Implement `scanner.Scanner` in `internal/scanner/<lang>.go`:

```go
type Scanner interface {
    Name() string                                  // stable id: "js", "go", "python"
    Supports(path string) bool                     // claim by extension
    Scan(path string, src []byte) ([]Ref, error)   // find references
}
```

Then register it in `All()` in `internal/scanner/scanner.go`. `All()` is the
single source of truth — `Walk` dispatches through it, and the tests iterate
it, so registration is the only wiring step.

## Steps

1. Create `internal/scanner/<lang>.go` with an unexported struct type
   (`pythonScanner`), matching the existing `jsScanner` / `goScanner` shape.
2. Implement the three methods. Keep `Supports` extension-based and cheap.
3. Add it to `All()`.
4. Extend the table in `internal/scanner/scanner_test.go`:
   - every extension you claim, mapped to your `Name()`
   - at least one path you must *not* claim
   - the existing test already asserts no two scanners claim the same path
     and that names are unique — do not weaken those.
5. Add a `Scan` test with a realistic snippet per supported syntax form, plus
   the negative cases below.
6. Document the syntax forms in the README Features list.
7. Run `make check`.

## Syntax to cover, by language

| Language | Forms |
|---|---|
| JS/TS | `process.env.X`, `process.env["X"]`, `process.env['X']`, `import.meta.env.X` |
| Go | `os.Getenv("X")`, `os.LookupEnv("X")` |
| Python | `os.environ["X"]`, `os.environ.get("X")`, `os.getenv("X")`, `environ.get("X")` |
| Ruby | `ENV["X"]`, `ENV.fetch("X")`, `ENV['X']` |
| Rust | `std::env::var("X")`, `env::var("X")`, `env!("X")`, `option_env!("X")` |

## False positives to reject

These are what make a scanner trustworthy. Each deserves a test case.

- **Dynamic keys.** `process.env[keyName]`, `os.Getenv(key)`, f-strings. The
  key is not a literal, so skip it — never guess. Reporting a variable the
  user cannot act on is worse than missing one.
- **Comments.** `// os.Getenv("OLD_KEY")` must not register.
- **Strings that merely look like access.** A key inside a string literal,
  or in documentation examples.
- **Destructuring and spreads.** `const {A, B} = process.env` reads two
  variables; decide whether you support it and test what you chose.
- **Test and generated files.** Not the scanner's job — the include/exclude
  patterns in `config.Scan` handle that. Do not special-case paths here.

## Rules

- **Never record a value.** A scanner returns key names, files, and lines.
  `Ref` has no value field and must not gain one. See the invariant in
  CLAUDE.md.
- **Be conservative.** A missed variable is a smaller failure than a false
  positive, which trains users to ignore the output.
- **Report every occurrence**, not the deduplicated set. `check` decides what
  to collapse; the scanner reports facts.
- **Lines are 1-indexed**, matching editors and the README's example output.
- **Never fail the whole run on one unparsable file.** Return what you found.
- **Regex is acceptable** for v1 and is what the JS and Go scanners use. If a
  language's ambiguity makes regex untenable, a real parser is fine, but it
  must not add a cgo dependency — releases build with `CGO_ENABLED=0`.

## Verify

```bash
make check
go test ./internal/scanner/ -run TestScannerSupports -v
```
