// Package scanner finds environment variable references in source code.
package scanner

import "github.com/geekreflex/envcheck/internal/config"

// Ref is one use of an environment variable in source code.
type Ref struct {
	Key  string
	File string
	Line int
}

// Scanner extracts variable references from a single source file.
type Scanner interface {
	// Name identifies the scanner, e.g. "js" or "go".
	Name() string
	// Supports reports whether this scanner handles the given file path.
	Supports(path string) bool
	// Scan returns every variable reference found in src.
	Scan(path string, src []byte) ([]Ref, error)
}

// All returns the scanners enabled in v1.
func All() []Scanner {
	return []Scanner{jsScanner{}, goScanner{}}
}

// Walk scans dir with every registered scanner, honouring the include and
// exclude patterns in cfg.
func Walk(dir string, cfg config.Config) ([]Ref, error) {
	// TODO: walk the tree, filter by cfg.Scan, dispatch to All().
	return nil, nil
}
