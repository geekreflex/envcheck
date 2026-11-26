// Package envfile parses .env style files into variable names and values.
//
// Values are parsed because detecting empty values requires them, but they
// must never leave this package and internal/check. In particular, errors
// produced here identify a location (path and line) and a fixed reason, and
// never include any text from the file.
package envfile

import (
	"bufio"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"os"
	"strings"
)

// Var is a single declaration in an env file.
type Var struct {
	Key   string
	Value string
	Line  int
}

// File is a parsed env file.
type File struct {
	Path string
	Vars []Var
}

// Keys returns the declared variable names in file order.
func (f *File) Keys() []string {
	keys := make([]string, 0, len(f.Vars))
	for _, v := range f.Vars {
		keys = append(keys, v.Key)
	}
	return keys
}

// SyntaxError reports a malformed line. Reason is always one of a fixed set
// of messages and never contains file content, so it is safe to print.
type SyntaxError struct {
	Path   string
	Line   int
	Reason string
}

func (e *SyntaxError) Error() string {
	return fmt.Sprintf("parsing %s:%d: %s", e.Path, e.Line, e.Reason)
}

// Fixed syntax error reasons. Never build a reason from file content.
const (
	reasonMissingEquals = "expected KEY=value"
	reasonInvalidKey    = "invalid variable name"
	reasonUnterminated  = "unterminated quoted value"
	reasonTrailing      = "unexpected text after closing quote"
)

// Parse reads the env file at path.
//
// Supported syntax, one declaration per line:
//
//	KEY=value              unquoted; " #" starts an inline comment
//	export KEY=value       optional export prefix
//	KEY='literal'          single quotes: no escapes, may span lines
//	KEY="a\nb"             double quotes: \n \r \t \\ \" escapes, may span lines
//	KEY=`literal`          backticks: like single quotes
//	# comment              full-line comments and blank lines are skipped
//
// Variable references such as ${OTHER} are kept literally, not expanded.
// Duplicate keys are all returned, in file order.
//
// A missing file yields an error wrapping fs.ErrNotExist.
func Parse(path string) (*File, error) {
	f, err := os.Open(path) //nolint:gosec // G304: reading the user-named env file is the purpose
	if err != nil {
		return nil, fmt.Errorf("reading %s: %w", path, unwrapPath(err))
	}
	defer f.Close() //nolint:errcheck // read-only; close error carries no data loss

	return parse(path, f)
}

// unwrapPath drops the *fs.PathError wrapper, whose message repeats the
// path the caller already names. The cause (fs.ErrNotExist etc.) is kept.
func unwrapPath(err error) error {
	var pe *fs.PathError
	if errors.As(err, &pe) {
		return pe.Err
	}
	return err
}

// parse does the work of Parse on any reader; path is used only for
// reporting.
func parse(path string, r io.Reader) (*File, error) {
	lines, err := readLines(r)
	if err != nil {
		return nil, fmt.Errorf("reading %s: %w", path, err)
	}

	file := &File{Path: path, Vars: []Var{}}
	for i := 0; i < len(lines); i++ {
		lineNo := i + 1
		line := strings.TrimSpace(lines[i])
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}

		line = stripExport(line)

		key, rest, ok := strings.Cut(line, "=")
		if !ok {
			return nil, &SyntaxError{Path: path, Line: lineNo, Reason: reasonMissingEquals}
		}
		key = strings.TrimSpace(key)
		if !validKey(key) {
			return nil, &SyntaxError{Path: path, Line: lineNo, Reason: reasonInvalidKey}
		}

		value, consumed, reason := parseValue(rest, lines[i+1:])
		if reason != "" {
			return nil, &SyntaxError{Path: path, Line: lineNo, Reason: reason}
		}

		file.Vars = append(file.Vars, Var{Key: key, Value: value, Line: lineNo})
		i += consumed
	}
	return file, nil
}

// readLines splits r into lines without a length limit, dropping a leading
// UTF-8 BOM and trailing CR so CRLF files parse like LF files.
func readLines(r io.Reader) ([]string, error) {
	br := bufio.NewReader(r)
	var lines []string
	for {
		line, err := br.ReadString('\n')
		if line != "" || err == nil {
			line = strings.TrimSuffix(line, "\n")
			line = strings.TrimSuffix(line, "\r")
			if len(lines) == 0 {
				line = strings.TrimPrefix(line, "\uFEFF")
			}
			lines = append(lines, line)
		}
		if errors.Is(err, io.EOF) {
			return lines, nil
		}
		if err != nil {
			return nil, err
		}
	}
}

// stripExport removes a leading "export" keyword followed by whitespace. A
// variable literally named "export" (export=1) is left alone.
func stripExport(line string) string {
	rest, ok := strings.CutPrefix(line, "export")
	if !ok || rest == "" || (rest[0] != ' ' && rest[0] != '\t') {
		return line
	}
	return strings.TrimLeft(rest, " \t")
}

// validKey reports whether s is a portable environment variable name:
// a letter or underscore followed by letters, digits, or underscores.
func validKey(s string) bool {
	if s == "" {
		return false
	}
	for i, c := range s {
		switch {
		case c == '_', c >= 'A' && c <= 'Z', c >= 'a' && c <= 'z':
		case c >= '0' && c <= '9' && i > 0:
		default:
			return false
		}
	}
	return true
}

// parseValue parses the text after '=' on the current line. next holds the
// following lines, which a quoted value may continue into. It returns the
// value, how many of next it consumed, and a fixed reason on failure.
func parseValue(raw string, next []string) (value string, consumed int, reason string) {
	s := strings.TrimLeft(raw, " \t")
	// "KEY= # note" is an empty value with a comment; "KEY=#abc" is not.
	if s == "" || (s[0] == '#' && len(s) < len(raw)) {
		return "", 0, ""
	}

	quote := s[0]
	if quote != '\'' && quote != '"' && quote != '`' {
		return unquoted(s), 0, ""
	}

	// Find the closing quote, continuing onto later lines if needed.
	body := s[1:]
	for {
		if end := closingQuote(body, quote); end >= 0 {
			if !onlyComment(body[end+1:]) {
				return "", 0, reasonTrailing
			}
			body = body[:end]
			break
		}
		if consumed == len(next) {
			return "", 0, reasonUnterminated
		}
		body += "\n" + next[consumed]
		consumed++
	}

	if quote == '"' {
		body = unescape(body)
	}
	return body, consumed, ""
}

// unquoted trims an unquoted value and strips an inline comment, which must
// be preceded by whitespace so values like a#b survive.
func unquoted(s string) string {
	for i := 1; i < len(s); i++ {
		if s[i] == '#' && (s[i-1] == ' ' || s[i-1] == '\t') {
			s = s[:i]
			break
		}
	}
	return strings.TrimSpace(s)
}

// closingQuote returns the index of the first unescaped quote in s, or -1.
// Backslash escapes apply only inside double quotes.
func closingQuote(s string, quote byte) int {
	for i := 0; i < len(s); i++ {
		switch {
		case quote == '"' && s[i] == '\\':
			i++ // skip the escaped byte
		case s[i] == quote:
			return i
		}
	}
	return -1
}

// onlyComment reports whether s is blank or blank followed by a comment.
func onlyComment(s string) bool {
	s = strings.TrimLeft(s, " \t")
	return s == "" || s[0] == '#'
}

// unescape expands the escapes recognised inside double quotes. Unknown
// escapes are kept verbatim, backslash included.
func unescape(s string) string {
	if !strings.Contains(s, `\`) {
		return s
	}
	var b strings.Builder
	b.Grow(len(s))
	for i := 0; i < len(s); i++ {
		if s[i] != '\\' || i+1 == len(s) {
			b.WriteByte(s[i])
			continue
		}
		i++
		switch s[i] {
		case 'n':
			b.WriteByte('\n')
		case 'r':
			b.WriteByte('\r')
		case 't':
			b.WriteByte('\t')
		case '\\', '"':
			b.WriteByte(s[i])
		default:
			b.WriteByte('\\')
			b.WriteByte(s[i])
		}
	}
	return b.String()
}
