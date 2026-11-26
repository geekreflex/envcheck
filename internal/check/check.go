// Package check compares env files against each other and against code usage.
package check

import (
	"path"
	"slices"
	"sort"

	"github.com/geekreflex/envcheck/internal/envfile"
	"github.com/geekreflex/envcheck/internal/scanner"
)

// Kind classifies a problem found during a check.
type Kind string

const (
	// MissingInEnv is declared in .env.example but absent from .env.
	MissingInEnv Kind = "missing_in_env"
	// UndeclaredInExample is used in code but absent from .env.example.
	UndeclaredInExample Kind = "undeclared_in_example"
	// Unused is declared but never referenced in code.
	Unused Kind = "unused"
	// EmptyValue is declared with an empty value.
	EmptyValue Kind = "empty_value"
)

// Problem is a single finding. It never carries a variable's value.
type Problem struct {
	Kind Kind   `json:"kind"`
	Key  string `json:"key"`
	File string `json:"file,omitempty"`
	Line int    `json:"line,omitempty"`
}

// Result is the outcome of a full check.
type Result struct {
	Problems []Problem `json:"problems"`
}

// OK reports whether the check found nothing to fix.
func (r Result) OK() bool { return len(r.Problems) == 0 }

// Input is everything a check needs.
type Input struct {
	// Example is the documented contract. It is required.
	Example *envfile.File
	// Env is the local env file. Nil means there is none (common in CI),
	// which skips the missing and empty checks.
	Env *envfile.File
	// Refs are the variable reads found in source code.
	Refs []scanner.Ref
	// ScannedFiles is how many source files the scanners read. When it is
	// zero there is no code evidence either way, so the unused and
	// undeclared checks are skipped rather than flagging every variable.
	ScannedFiles int
	// Ignore lists key names or path.Match globs (NEXT_PUBLIC_*) that are
	// never reported.
	Ignore []string
}

// kindOrder fixes the order problems are reported in, matching the README.
var kindOrder = map[Kind]int{
	MissingInEnv:        0,
	UndeclaredInExample: 1,
	Unused:              2,
	EmptyValue:          3,
}

// Run compares the example file, the local env file and the code references.
// Problems are returned in a stable order: by kind, then key, then location.
func Run(in Input) Result {
	ignored := func(key string) bool {
		for _, pat := range in.Ignore {
			if ok, err := path.Match(pat, key); ok && err == nil {
				return true
			}
		}
		return false
	}

	example := firstDecl(in.Example)
	problems := []Problem{}
	add := func(p Problem) {
		if !ignored(p.Key) {
			problems = append(problems, p)
		}
	}

	if in.Env != nil {
		env := firstDecl(in.Env)
		for _, v := range in.Example.Vars {
			if _, ok := env[v.Key]; !ok && example[v.Key] == v.Line {
				add(Problem{Kind: MissingInEnv, Key: v.Key, File: in.Example.Path, Line: v.Line})
			}
		}
		// Emptiness is judged on the last declaration, which is the one
		// dotenv loaders keep.
		for _, v := range lastDecl(in.Env) {
			if v.Value == "" {
				add(Problem{Kind: EmptyValue, Key: v.Key, File: in.Env.Path, Line: v.Line})
			}
		}
	}

	if in.ScannedFiles > 0 {
		used := make(map[string]bool, len(in.Refs))
		reported := make(map[string]bool)
		for _, r := range sortedRefs(in.Refs) {
			used[r.Key] = true
			if _, ok := example[r.Key]; !ok && !reported[r.Key] {
				reported[r.Key] = true
				add(Problem{Kind: UndeclaredInExample, Key: r.Key, File: r.File, Line: r.Line})
			}
		}
		for _, v := range in.Example.Vars {
			if !used[v.Key] && example[v.Key] == v.Line {
				add(Problem{Kind: Unused, Key: v.Key, File: in.Example.Path, Line: v.Line})
			}
		}
	}

	sort.SliceStable(problems, func(i, j int) bool {
		a, b := problems[i], problems[j]
		if a.Kind != b.Kind {
			return kindOrder[a.Kind] < kindOrder[b.Kind]
		}
		if a.Key != b.Key {
			return a.Key < b.Key
		}
		if a.File != b.File {
			return a.File < b.File
		}
		return a.Line < b.Line
	})
	return Result{Problems: problems}
}

// firstDecl maps each key to the line of its first declaration, so a
// duplicated key is reported once.
func firstDecl(f *envfile.File) map[string]int {
	m := make(map[string]int, len(f.Vars))
	for _, v := range f.Vars {
		if _, ok := m[v.Key]; !ok {
			m[v.Key] = v.Line
		}
	}
	return m
}

// lastDecl returns the final declaration of each key, in file order.
func lastDecl(f *envfile.File) []envfile.Var {
	last := make(map[string]int, len(f.Vars))
	for i, v := range f.Vars {
		last[v.Key] = i
	}
	out := make([]envfile.Var, 0, len(last))
	for i, v := range f.Vars {
		if last[v.Key] == i {
			out = append(out, v)
		}
	}
	return out
}

// sortedRefs orders refs by file then line, so "first occurrence" is
// deterministic regardless of scan order.
func sortedRefs(refs []scanner.Ref) []scanner.Ref {
	out := slices.Clone(refs)
	sort.SliceStable(out, func(i, j int) bool {
		if out[i].File != out[j].File {
			return out[i].File < out[j].File
		}
		return out[i].Line < out[j].Line
	})
	return out
}
