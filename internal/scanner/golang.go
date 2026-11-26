package scanner

import (
	"path/filepath"
	"regexp"
)

// goScanner matches os.Getenv("X") and os.LookupEnv("X").
type goScanner struct{}

// goPattern matches a call with a literal key, in "double" or `raw` quotes.
// Calls with a variable or computed key are skipped: there is no name to
// report, and guessing would create false positives.
var goPattern = regexp.MustCompile(`\bos\.(?:Getenv|LookupEnv)\(\s*` + quotedKey(`"`, "`") + `\s*\)`)

func (goScanner) Name() string { return "go" }

func (goScanner) Supports(path string) bool {
	return filepath.Ext(path) == ".go"
}

func (goScanner) Scan(path string, src []byte) ([]Ref, error) {
	return prepare(src, lexGo).findKeys(path, goPattern), nil
}
