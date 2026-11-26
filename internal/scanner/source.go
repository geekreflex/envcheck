package scanner

import (
	"regexp"
	"sort"
)

// source is a file prepared for pattern matching: comments are blanked out,
// and each byte is classified as code or string-literal content, so a match
// that starts inside a string ("see process.env.X") can be rejected.
type source struct {
	code     []byte // src with comments replaced by spaces; same length
	inString []bool // true for bytes inside a string literal, quotes included
	lines    lineIndex
}

// lexMode selects the string-literal rules of the language being scanned.
type lexMode int

const (
	// lexGo: "interpreted" and `raw` strings, 'runes'. No interpolation.
	lexGo lexMode = iota
	// lexJS: "double", 'single', and `template ${expr}` literals.
	lexJS
)

// prepare lexes src just far enough to find comments and string literals.
// It is not a full tokenizer: double and single quoted strings end at a
// newline, which bounds the damage if it is ever confused (for example by a
// JavaScript regex literal containing a quote).
func prepare(src []byte, mode lexMode) source {
	s := source{
		code:     make([]byte, len(src)),
		inString: make([]bool, len(src)),
		lines:    newLineIndex(src),
	}
	copy(s.code, src)

	blank := func(from, to int) {
		for k := from; k < to; k++ {
			if s.code[k] != '\n' {
				s.code[k] = ' '
			}
		}
	}
	mark := func(from, to int) {
		for k := from; k < to; k++ {
			s.inString[k] = true
		}
	}

	// braces has one entry per open template ${ ... } expression, counting
	// the unclosed { } blocks inside it. Empty means top-level code.
	var braces []int

	n := len(src)
	for i := 0; i < n; i++ {
		c := src[i]
		switch {
		case c == '/' && i+1 < n && src[i+1] == '/':
			end := i
			for end < n && src[end] != '\n' {
				end++
			}
			blank(i, end)
			i = end - 1

		case c == '/' && i+1 < n && src[i+1] == '*':
			end := i + 2
			for end+1 < n && (src[end] != '*' || src[end+1] != '/') {
				end++
			}
			end = min(end+2, n)
			blank(i, end)
			i = end - 1

		case c == '"' || c == '\'':
			end := closeQuoted(src, i)
			mark(i, end)
			i = end - 1

		case c == '`' && mode == lexGo:
			end := i + 1
			for end < n && src[end] != '`' {
				end++
			}
			end = min(end+1, n)
			mark(i, end)
			i = end - 1

		case c == '`' && mode == lexJS:
			end, expr := closeTemplate(src, i)
			mark(i, end)
			if expr {
				braces = append(braces, 0) // code resumes inside ${
			}
			i = end - 1

		case c == '{' && len(braces) > 0:
			braces[len(braces)-1]++

		case c == '}' && len(braces) > 0:
			if top := len(braces) - 1; braces[top] > 0 {
				braces[top]--
				break
			}
			// This } closes a ${ } expression: the template text resumes.
			braces = braces[:len(braces)-1]
			end, expr := closeTemplate(src, i)
			mark(i, end)
			if expr {
				braces = append(braces, 0)
			}
			i = end - 1
		}
	}
	return s
}

// closeQuoted returns the end (exclusive) of the "double" or 'single' quoted
// literal opening at src[i]. It stops at an unescaped newline.
func closeQuoted(src []byte, i int) int {
	quote := src[i]
	for j := i + 1; j < len(src); j++ {
		switch src[j] {
		case '\\':
			j++
		case '\n':
			return j
		case quote:
			return j + 1
		}
	}
	return len(src)
}

// closeTemplate scans template-literal text starting just after src[i]
// (the opening backtick, or the } closing an expression). It returns the
// end (exclusive) of the literal text, and whether it stopped at a ${ that
// opens an expression rather than at the closing backtick.
func closeTemplate(src []byte, i int) (end int, expr bool) {
	for j := i + 1; j < len(src); j++ {
		switch {
		case src[j] == '\\':
			j++
		case src[j] == '`':
			return j + 1, false
		case src[j] == '$' && j+1 < len(src) && src[j+1] == '{':
			return j + 2, true
		}
	}
	return len(src), false
}

// lineIndex maps byte offsets to 1-indexed line numbers.
type lineIndex []int

func newLineIndex(src []byte) lineIndex {
	starts := lineIndex{0}
	for i, c := range src {
		if c == '\n' {
			starts = append(starts, i+1)
		}
	}
	return starts
}

// line returns the 1-indexed line containing byte offset off.
func (l lineIndex) line(off int) int {
	return sort.Search(len(l), func(i int) bool { return l[i] > off })
}

// matches returns the submatch indices of every match of re that starts in
// code rather than inside a string literal.
func (s source) matches(re *regexp.Regexp) [][]int {
	var out [][]int
	for _, m := range re.FindAllSubmatchIndex(s.code, -1) {
		if !s.inString[m[0]] {
			out = append(out, m)
		}
	}
	return out
}

// findKeys returns a Ref for every code match of each pattern. A pattern
// names the variable with one or more capture groups called "key" (one per
// alternative); whichever took part in the match is used.
func (s source) findKeys(path string, patterns ...*regexp.Regexp) []Ref {
	var refs []Ref
	for _, re := range patterns {
		var keyGroups []int
		for i, name := range re.SubexpNames() {
			if name == "key" {
				keyGroups = append(keyGroups, i)
			}
		}
		for _, m := range s.matches(re) {
			for _, k := range keyGroups {
				if m[2*k] >= 0 {
					refs = append(refs, s.ref(path, m[2*k], m[2*k+1]))
					break
				}
			}
		}
	}
	return refs
}

func (s source) ref(path string, from, to int) Ref {
	return Ref{Key: string(s.code[from:to]), File: path, Line: s.lines.line(from)}
}

// keyPattern is a portable environment variable name, matching what the
// envfile parser accepts.
const keyPattern = `[A-Za-z_][A-Za-z0-9_]*`

// quotedKey matches a key in any of the given quote characters, with one
// "key" group per quote style.
func quotedKey(quotes ...string) string {
	alts := ""
	for i, q := range quotes {
		if i > 0 {
			alts += "|"
		}
		alts += q + `(?P<key>` + keyPattern + `)` + q
	}
	return `(?:` + alts + `)`
}
