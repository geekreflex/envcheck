package scanner

import (
	"os"
	"path/filepath"
	"reflect"
	"sort"
	"strconv"
	"testing"

	"github.com/geekreflex/envcheck/internal/config"
)

// keysAt flattens refs to "KEY:line" for compact comparison.
func keysAt(refs []Ref) []string {
	out := []string{}
	for _, r := range refs {
		out = append(out, r.Key+":"+strconv.Itoa(r.Line))
	}
	sort.Strings(out)
	return out
}

func TestGoScan(t *testing.T) {
	tests := []struct {
		name string
		src  string
		want []string
	}{
		{"Getenv", `x := os.Getenv("DATABASE_URL")`, []string{"DATABASE_URL:1"}},
		{"LookupEnv", `v, ok := os.LookupEnv("PORT")`, []string{"PORT:1"}},
		{"raw string key", "os.Getenv(`TOKEN`)", []string{"TOKEN:1"}},
		{"spaces inside call", `os.Getenv( "A" )`, []string{"A:1"}},
		{"line numbers", "package main\n\nfunc f() {\n\t_ = os.Getenv(\"A\")\n\t_ = os.Getenv(\"B\")\n}", []string{"A:4", "B:5"}},
		{"every occurrence", `os.Getenv("A"); os.Getenv("A")`, []string{"A:1", "A:1"}},
		{"dynamic key skipped", `os.Getenv(key)`, []string{}},
		{"concatenated key skipped", `os.Getenv("PREFIX_" + name)`, []string{}},
		{"line comment skipped", `// os.Getenv("OLD")`, []string{}},
		{"trailing comment skipped", `x := 1 // os.Getenv("OLD")`, []string{}},
		{"block comment skipped", "/*\nos.Getenv(\"OLD\")\n*/\nos.Getenv(\"NEW\")", []string{"NEW:4"}},
		{"inside string skipped", `msg := "call os.Getenv(\"X\") first"`, []string{}},
		{"inside raw string skipped", "doc := `os.Getenv(\"X\")`", []string{}},
		{"url is not a comment", `u := "http://x"; os.Getenv("A")`, []string{"A:1"}},
		{"raw string with backslash", "p := `C:\\`; os.Getenv(\"A\")", []string{"A:1"}},
		{"rune quote", `c := '"'; os.Getenv("A")`, []string{"A:1"}},
		{"other package skipped", `myos.Getenv("A")`, []string{}},
		{"invalid name skipped", `os.Getenv("MY-KEY")`, []string{}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			refs, err := goScanner{}.Scan("main.go", []byte(tt.src))
			if err != nil {
				t.Fatal(err)
			}
			if got := keysAt(refs); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("Scan() = %v, want %v", got, tt.want)
			}
			for _, r := range refs {
				if r.File != "main.go" {
					t.Errorf("File = %q, want main.go", r.File)
				}
			}
		})
	}
}

func TestJSScan(t *testing.T) {
	tests := []struct {
		name string
		src  string
		want []string
	}{
		{"dot", `const u = process.env.REDIS_URL`, []string{"REDIS_URL:1"}},
		{"optional chaining", `process.env?.A`, []string{"A:1"}},
		{"double bracket", `process.env["A"]`, []string{"A:1"}},
		{"single bracket", `process.env['A']`, []string{"A:1"}},
		{"template bracket", "process.env[`A`]", []string{"A:1"}},
		{"import.meta.env", `import.meta.env.VITE_API_URL`, []string{"VITE_API_URL:1"}},
		{"import.meta.env bracket", `import.meta.env["VITE_A"]`, []string{"VITE_A:1"}},
		{"vite builtins skipped", `import.meta.env.MODE; import.meta.env.DEV; import.meta.env.BASE_URL`, []string{}},
		{"builtin names on process.env kept", `process.env.MODE`, []string{"MODE:1"}},
		{"destructuring", "const { A, B: b, C = 'x', ...rest } = process.env", []string{"A:1", "B:1", "C:1"}},
		{"multi-line destructuring", "const {\n  A,\n  B,\n} = process.env", []string{"A:2", "B:3"}},
		{"quoted destructuring key", `const { "A": a } = process.env`, []string{"A:1"}},
		{"destructure import.meta builtin skipped", `const { MODE, VITE_X } = import.meta.env`, []string{"VITE_X:1"}},
		{"destructure other object skipped", `const { A } = config`, []string{}},
		{"dynamic key skipped", `process.env[name]`, []string{}},
		{"bracket with expression skipped", "process.env[`PREFIX_${n}`]", []string{}},
		{"line comment skipped", `// process.env.OLD`, []string{}},
		{"block comment skipped", "/** uses process.env.OLD */\nprocess.env.NEW", []string{"NEW:2"}},
		{"inside string skipped", `log("set process.env.X first")`, []string{}},
		{"inside template text skipped", "const s = `process.env.X`", []string{}},
		{"template expression scanned", "const u = `${process.env.API_URL}/users`", []string{"API_URL:1"}},
		{"nested braces in expression", "`${fn({a: 1})}${process.env.B}`", []string{"B:1"}},
		{"code after template", "const s = `a ${x} b`; process.env.A", []string{"A:1"}},
		{"url is not a comment", `fetch("https://x"); process.env.A`, []string{"A:1"}},
		{"prefix identifier skipped", `myprocess.env.A`, []string{}},
		{"line numbers", "\n\nif (process.env.A) {\n  use(process.env.B)\n}", []string{"A:3", "B:4"}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			refs, err := jsScanner{}.Scan("src/app.ts", []byte(tt.src))
			if err != nil {
				t.Fatal(err)
			}
			if got := keysAt(refs); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("Scan() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestMatch(t *testing.T) {
	tests := []struct {
		pattern, name string
		want          bool
	}{
		{"**", "a/b/c.go", true},
		{"**", "main.go", true},
		{"src/**", "src/a/b.ts", true},
		{"src/**", "lib/a.ts", false},
		{"**/node_modules/**", "node_modules/x/index.js", true},
		{"**/node_modules/**", "packages/web/node_modules/x/index.js", true},
		{"**/node_modules/**", "src/node_modules.ts", false},
		{"**/*.test.ts", "src/a/b.test.ts", true},
		{"**/*.test.ts", "b.test.ts", true},
		{"*.go", "main.go", true},
		{"*.go", "cmd/main.go", false},
		{"[bad", "x", false},
	}
	for _, tt := range tests {
		t.Run(tt.pattern+" "+tt.name, func(t *testing.T) {
			if got := Match(tt.pattern, tt.name); got != tt.want {
				t.Errorf("Match(%q, %q) = %v, want %v", tt.pattern, tt.name, got, tt.want)
			}
		})
	}
}

func TestWalk(t *testing.T) {
	dir := t.TempDir()
	files := map[string]string{
		"main.go":                           `package main; var _ = os.Getenv("GO_KEY")`,
		"src/app.ts":                        `process.env.TS_KEY`,
		"src/app.test.ts":                   `process.env.TEST_KEY`,
		"README.md":                         `process.env.DOC_KEY`,
		"node_modules/lib/index.js":         `process.env.DEP_KEY`,
		"packages/web/node_modules/x.js":    `process.env.NESTED_DEP_KEY`,
		"packages/web/.next/server/page.js": `process.env.BUILD_KEY`,
		"dist/bundle.js":                    `process.env.DIST_KEY`,
		"vendor/pkg/pkg.go":                 `os.Getenv("VENDOR_KEY")`,
	}
	for name, content := range files {
		p := filepath.Join(dir, filepath.FromSlash(name))
		if err := os.MkdirAll(filepath.Dir(p), 0o750); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(p, []byte(content), 0o600); err != nil {
			t.Fatal(err)
		}
	}

	cfg := config.Default()
	cfg.Scan.Exclude = append(cfg.Scan.Exclude, "**/*.test.ts")

	res, err := Walk(dir, cfg)
	if err != nil {
		t.Fatal(err)
	}
	if res.Files != 2 {
		t.Errorf("Files = %d, want 2", res.Files)
	}
	got := map[string]string{}
	for _, r := range res.Refs {
		rel, _ := filepath.Rel(dir, filepath.FromSlash(r.File))
		got[r.Key] = filepath.ToSlash(rel)
	}
	want := map[string]string{"GO_KEY": "main.go", "TS_KEY": "src/app.ts"}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("Walk() refs = %v, want %v", got, want)
	}
}

func TestWalkInclude(t *testing.T) {
	dir := t.TempDir()
	for name, content := range map[string]string{
		"src/a.ts":     `process.env.IN`,
		"scripts/b.js": `process.env.OUT`,
	} {
		p := filepath.Join(dir, filepath.FromSlash(name))
		if err := os.MkdirAll(filepath.Dir(p), 0o750); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(p, []byte(content), 0o600); err != nil {
			t.Fatal(err)
		}
	}

	cfg := config.Default()
	cfg.Scan.Include = []string{"src/**"}
	res, err := Walk(dir, cfg)
	if err != nil {
		t.Fatal(err)
	}
	if len(res.Refs) != 1 || res.Refs[0].Key != "IN" {
		t.Errorf("Walk() refs = %+v, want only IN", res.Refs)
	}
}

func TestWalkMissingDir(t *testing.T) {
	if _, err := Walk(filepath.Join(t.TempDir(), "nope"), config.Default()); err == nil {
		t.Fatal("Walk() on a missing directory returned nil error")
	}
}
