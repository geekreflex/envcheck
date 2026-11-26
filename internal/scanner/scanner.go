// Package scanner finds environment variable references in source code.
package scanner

import (
	"fmt"
	"io/fs"
	"os"
	"path"
	"path/filepath"
	"strings"

	"github.com/geekreflex/envcheck/internal/config"
)

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

// Result is the outcome of walking a source tree.
type Result struct {
	// Refs is every variable reference found, in walk order.
	Refs []Ref
	// Files counts the source files a scanner read. Zero means no supported
	// source was found, which is different from code that reads no
	// variables.
	Files int
}

// maxFileSize skips files too large to be hand-written source: bundles,
// generated code, and vendored blobs that slipped past the exclude list.
const maxFileSize = 2 << 20

// Walk scans dir with every registered scanner, honouring the include and
// exclude patterns in cfg. Patterns are matched against slash-separated
// paths relative to dir; ** matches any number of path segments. Ref.File
// is dir joined with that relative path.
func Walk(dir string, cfg config.Config) (Result, error) {
	scanners := All()
	var res Result

	err := filepath.WalkDir(dir, func(p string, d fs.DirEntry, err error) error {
		if err != nil {
			return fmt.Errorf("scanning %s: %w", p, err)
		}
		rel, err := filepath.Rel(dir, p)
		if err != nil {
			return fmt.Errorf("scanning %s: %w", p, err)
		}
		rel = filepath.ToSlash(rel)

		if d.IsDir() {
			if rel != "." && excludesDir(cfg.Scan.Exclude, rel) {
				return filepath.SkipDir
			}
			return nil
		}
		if !d.Type().IsRegular() || !MatchAny(cfg.Scan.Include, rel) || MatchAny(cfg.Scan.Exclude, rel) {
			return nil
		}

		var sc Scanner
		for _, s := range scanners {
			if s.Supports(rel) {
				sc = s
				break
			}
		}
		if sc == nil {
			return nil
		}

		info, err := d.Info()
		if err != nil {
			return fmt.Errorf("scanning %s: %w", p, err)
		}
		if info.Size() > maxFileSize {
			return nil
		}

		src, err := os.ReadFile(p) //nolint:gosec // G304: walking the user's project is the purpose
		if err != nil {
			return fmt.Errorf("reading %s: %w", p, err)
		}
		refs, err := sc.Scan(filepath.ToSlash(p), src)
		if err != nil {
			return fmt.Errorf("scanning %s: %w", p, err)
		}
		res.Files++
		res.Refs = append(res.Refs, refs...)
		return nil
	})
	if err != nil {
		return Result{}, err
	}
	return res, nil
}

// excludesDir reports whether an exclude pattern ending in /** covers the
// whole directory rel, so the walk can skip it without visiting each file.
func excludesDir(patterns []string, rel string) bool {
	for _, p := range patterns {
		if prefix, ok := strings.CutSuffix(p, "/**"); ok && Match(prefix, rel) {
			return true
		}
	}
	return false
}

// MatchAny reports whether name matches any of patterns.
func MatchAny(patterns []string, name string) bool {
	for _, p := range patterns {
		if Match(p, name) {
			return true
		}
	}
	return false
}

// Match reports whether the slash-separated name matches pattern. Each
// segment is matched with path.Match, and a ** segment matches zero or more
// whole segments. A malformed pattern matches nothing; config.Load rejects
// those before a walk starts.
func Match(pattern, name string) bool {
	return matchSegments(strings.Split(pattern, "/"), strings.Split(name, "/"))
}

func matchSegments(pat, name []string) bool {
	if len(pat) == 0 {
		return len(name) == 0
	}
	if pat[0] == "**" {
		for i := 0; i <= len(name); i++ {
			if matchSegments(pat[1:], name[i:]) {
				return true
			}
		}
		return false
	}
	if len(name) == 0 {
		return false
	}
	ok, err := path.Match(pat[0], name[0])
	return ok && err == nil && matchSegments(pat[1:], name[1:])
}
