// Package report renders check results for humans and for machines.
//
// Reports are built only from check.Problem, which carries names and
// locations and never a value, so nothing rendered here can leak a secret.
package report

import (
	"encoding/json"
	"fmt"
	"io"
	"strings"

	"github.com/geekreflex/envcheck/internal/check"
)

// Files names the env files being compared, for section headings.
type Files struct {
	Example string
	Env     string
}

// section describes how one kind of problem is presented.
type section struct {
	kind    check.Kind
	heading func(Files) string
	mark    string
}

// sections lists every kind in report order. Errors (✗) are things that
// will break at runtime; warnings (!) are drift worth cleaning up.
var sections = []section{
	{check.MissingInEnv, func(f Files) string {
		return fmt.Sprintf("Missing in %s (declared in %s)", f.Env, f.Example)
	}, "✗"},
	{check.UndeclaredInExample, func(f Files) string {
		return fmt.Sprintf("Used in code but not declared in %s", f.Example)
	}, "✗"},
	{check.Unused, func(f Files) string {
		return fmt.Sprintf("Declared in %s but never used in code", f.Example)
	}, "!"},
	{check.EmptyValue, func(f Files) string {
		return fmt.Sprintf("Empty values in %s", f.Env)
	}, "!"},
}

// Text writes the human-readable report to w.
func Text(w io.Writer, res check.Result, files Files) error {
	var b strings.Builder

	if res.OK() {
		b.WriteString("✓ No problems found\n")
		_, err := io.WriteString(w, b.String())
		return err
	}

	for _, s := range sections {
		var group []check.Problem
		width := 0
		for _, p := range res.Problems {
			if p.Kind == s.kind {
				group = append(group, p)
				width = max(width, len(p.Key))
			}
		}
		if len(group) == 0 {
			continue
		}

		fmt.Fprintf(&b, "%s\n", s.heading(files))
		for _, p := range group {
			if loc := location(p); loc != "" {
				fmt.Fprintf(&b, "  %s %-*s  %s\n", s.mark, width, p.Key, loc)
			} else {
				fmt.Fprintf(&b, "  %s %s\n", s.mark, p.Key)
			}
		}
		b.WriteString("\n")
	}

	n := len(res.Problems)
	fmt.Fprintf(&b, "%d %s found\n", n, plural(n, "problem", "problems"))

	_, err := io.WriteString(w, b.String())
	return err
}

func location(p check.Problem) string {
	switch {
	case p.File != "" && p.Line > 0:
		return fmt.Sprintf("%s:%d", p.File, p.Line)
	default:
		return p.File
	}
}

func plural(n int, one, many string) string {
	if n == 1 {
		return one
	}
	return many
}

// jsonReport is the stable top-level JSON shape. Fields may be added;
// existing ones must not change meaning.
type jsonReport struct {
	Problems []check.Problem `json:"problems"`
}

// JSON writes the machine-readable report to w: one object with a
// "problems" array, which is empty (never null) when nothing was found.
func JSON(w io.Writer, res check.Result) error {
	problems := res.Problems
	if problems == nil {
		problems = []check.Problem{}
	}
	enc := json.NewEncoder(w)
	enc.SetIndent("", "  ")
	return enc.Encode(jsonReport{Problems: problems})
}
