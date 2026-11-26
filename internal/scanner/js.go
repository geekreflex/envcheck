package scanner

import (
	"path/filepath"
	"regexp"
	"strings"
)

// jsScanner matches process.env.X, process.env["X"], import.meta.env.X, and
// destructuring such as const { X, Y: alias } = process.env.
type jsScanner struct{}

// envObject is one object that exposes environment variables.
type envObject struct {
	dot, bracket, destructure *regexp.Regexp
	// builtins are properties the runtime sets itself; they are not
	// variables a user declares, so they are never reported.
	builtins map[string]bool
}

func newEnvObject(object string, builtins ...string) envObject {
	o := envObject{
		dot:     regexp.MustCompile(object + `(?:\?\.|\.)(?P<key>` + keyPattern + `)\b`),
		bracket: regexp.MustCompile(object + `(?:\?\.)?\[\s*` + quotedKey(`"`, `'`, "`") + `\s*\]`),
		// The body of { ... } = object. Nested patterns ({ A: { B } }) make
		// no sense for an env object, so [^{}] simply declines them.
		destructure: regexp.MustCompile(`(?:const|let|var)\s*\{(?P<body>[^{}]*)\}\s*=\s*` + object + `\b`),
		builtins:    make(map[string]bool, len(builtins)),
	}
	for _, b := range builtins {
		o.builtins[b] = true
	}
	return o
}

var (
	jsObjects = []envObject{
		newEnvObject(`\bprocess\.env`),
		// Vite sets these on import.meta.env itself.
		newEnvObject(`\bimport\.meta\.env`, "MODE", "BASE_URL", "PROD", "DEV", "SSR"),
	}
	jsKey = regexp.MustCompile(`^` + keyPattern + `$`)
)

func (jsScanner) Name() string { return "js" }

func (jsScanner) Supports(path string) bool {
	switch filepath.Ext(path) {
	case ".js", ".jsx", ".mjs", ".cjs", ".ts", ".tsx", ".mts", ".cts", ".vue", ".svelte", ".astro":
		return true
	}
	return false
}

func (jsScanner) Scan(path string, src []byte) ([]Ref, error) {
	s := prepare(src, lexJS)
	var refs []Ref
	for _, o := range jsObjects {
		found := s.findKeys(path, o.dot, o.bracket)
		found = append(found, s.destructured(path, o.destructure)...)
		for _, r := range found {
			if !o.builtins[r.Key] {
				refs = append(refs, r)
			}
		}
	}
	return refs, nil
}

// destructured returns a Ref for each property in const { A, B: b, C = 1,
// ...rest } = <object>. Rest elements and computed keys carry no literal
// name and are skipped.
func (s source) destructured(path string, re *regexp.Regexp) []Ref {
	var refs []Ref
	body := re.SubexpIndex("body")
	for _, m := range s.matches(re) {
		start, end := m[2*body], m[2*body+1]
		for start < end {
			next := end
			if comma := strings.IndexByte(string(s.code[start:end]), ','); comma >= 0 {
				next = start + comma
			}
			item := string(s.code[start:next])
			// The key is the text before any ':' (rename) or '=' (default).
			if cut := strings.IndexAny(item, ":="); cut >= 0 {
				item = item[:cut]
			}
			from := start + len(item) - len(strings.TrimLeft(item, " \t\r\n"))
			name := strings.TrimSpace(item)
			if len(name) >= 2 && (name[0] == '"' || name[0] == '\'') && name[len(name)-1] == name[0] {
				name = name[1 : len(name)-1]
				from++
			}
			if jsKey.MatchString(name) {
				refs = append(refs, s.ref(path, from, from+len(name)))
			}
			start = next + 1
		}
	}
	return refs
}
