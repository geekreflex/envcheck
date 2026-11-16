package scanner

import "testing"

func TestScannerSupports(t *testing.T) {
	tests := []struct {
		path string
		want string // scanner name expected to claim the path, "" for none
	}{
		{"src/lib/cache.ts", "js"},
		{"src/app/page.tsx", "js"},
		{"server.js", "js"},
		{"config.mjs", "js"},
		{"legacy.cjs", "js"},
		{"component.jsx", "js"},
		{"cmd/api/main.go", "go"},
		{"README.md", ""},
		{"env.d.ts.bak", ""},
		{".env.example", ""},
		{"Makefile", ""},
	}

	for _, tt := range tests {
		t.Run(tt.path, func(t *testing.T) {
			var claimed []string
			for _, s := range All() {
				if s.Supports(tt.path) {
					claimed = append(claimed, s.Name())
				}
			}

			if len(claimed) > 1 {
				t.Fatalf("%q claimed by multiple scanners %v; each path must map to at most one", tt.path, claimed)
			}
			got := ""
			if len(claimed) == 1 {
				got = claimed[0]
			}
			if got != tt.want {
				t.Errorf("scanner for %q = %q, want %q", tt.path, got, tt.want)
			}
		})
	}
}

func TestAllScannerNamesAreUniqueAndNonEmpty(t *testing.T) {
	seen := make(map[string]bool)
	for _, s := range All() {
		name := s.Name()
		if name == "" {
			t.Errorf("%T has an empty Name()", s)
		}
		if seen[name] {
			t.Errorf("duplicate scanner name %q", name)
		}
		seen[name] = true
	}
	if len(seen) == 0 {
		t.Fatal("All() returned no scanners")
	}
}
