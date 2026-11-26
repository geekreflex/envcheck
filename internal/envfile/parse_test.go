package envfile

import (
	"errors"
	"io/fs"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
)

// writeFixture writes content to a temp file and returns its path. Fixtures
// are generated per test rather than committed, so no .env-like file ever
// lands in the repository.
func writeFixture(t *testing.T, content string) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), "fixture.env")
	if err := os.WriteFile(path, []byte(content), 0o600); err != nil {
		t.Fatal(err)
	}
	return path
}

func TestParse(t *testing.T) {
	tests := []struct {
		name    string
		content string
		want    []Var
	}{
		{"empty file", "", []Var{}},
		{"only comments and blanks", "# header\n\n   \n\t# indented\n", []Var{}},
		{"simple", "PORT=8080\n", []Var{{"PORT", "8080", 1}}},
		{"no trailing newline", "PORT=8080", []Var{{"PORT", "8080", 1}}},
		{
			name:    "line numbers skip comments and blanks",
			content: "# db\n\nDATABASE_URL=postgres://x\n\nAPP_ENV=dev\n",
			want:    []Var{{"DATABASE_URL", "postgres://x", 3}, {"APP_ENV", "dev", 5}},
		},
		{"empty value", "API_KEY=\n", []Var{{"API_KEY", "", 1}}},
		{"empty value with comment", "API_KEY= # fill me in\n", []Var{{"API_KEY", "", 1}}},
		{"empty double quotes", `API_KEY=""`, []Var{{"API_KEY", "", 1}}},
		{"empty single quotes", `API_KEY=''`, []Var{{"API_KEY", "", 1}}},
		{"spaces around equals", "  PORT  =  8080  \n", []Var{{"PORT", "8080", 1}}},
		{"export prefix", "export PORT=8080\n", []Var{{"PORT", "8080", 1}}},
		{"export prefix with tab", "export\tPORT=8080\n", []Var{{"PORT", "8080", 1}}},
		{"variable named export", "export=1\n", []Var{{"export", "1", 1}}},
		{"variable prefixed export", "exported=1\n", []Var{{"exported", "1", 1}}},
		{"inline comment", "PORT=8080 # default\n", []Var{{"PORT", "8080", 1}}},
		{"hash without space is value", "THEME=#fff\n", []Var{{"THEME", "#fff", 1}}},
		{"hash mid value is value", "URL=http://x/a#b\n", []Var{{"URL", "http://x/a#b", 1}}},
		{"equals in value", "DSN=a=b=c\n", []Var{{"DSN", "a=b=c", 1}}},
		{"no expansion", "URL=${HOST}:${PORT}\n", []Var{{"URL", "${HOST}:${PORT}", 1}}},
		{"single quotes literal", `MSG='a\nb # not comment'`, []Var{{"MSG", `a\nb # not comment`, 1}}},
		{"double quotes escapes", `MSG="a\nb\t\"c\"\\"`, []Var{{"MSG", "a\nb\t\"c\"\\", 1}}},
		{"double quotes unknown escape kept", `RE="\d+"`, []Var{{"RE", `\d+`, 1}}},
		{"backticks literal", "MSG=`it's \"fine\"`", []Var{{"MSG", `it's "fine"`, 1}}},
		{"quoted then comment", `MSG="hi" # greeting`, []Var{{"MSG", "hi", 1}}},
		{"quoted keeps inner spaces", `MSG="  padded  "`, []Var{{"MSG", "  padded  ", 1}}},
		{
			name:    "multi-line double quotes",
			content: "KEY=\"-----BEGIN-----\nabc\n-----END-----\"\nNEXT=1\n",
			want:    []Var{{"KEY", "-----BEGIN-----\nabc\n-----END-----", 1}, {"NEXT", "1", 4}},
		},
		{
			name:    "multi-line single quotes",
			content: "KEY='line1\nline2'\nNEXT=1\n",
			want:    []Var{{"KEY", "line1\nline2", 1}, {"NEXT", "1", 3}},
		},
		{"crlf line endings", "A=1\r\nB=2\r\n", []Var{{"A", "1", 1}, {"B", "2", 2}}},
		{"utf-8 bom", "\xef\xbb\xbfA=1\n", []Var{{"A", "1", 1}}},
		{
			name:    "duplicates kept in order",
			content: "PORT=1\nPORT=2\n",
			want:    []Var{{"PORT", "1", 1}, {"PORT", "2", 2}},
		},
		{
			name:    "long line",
			content: "BIG=" + strings.Repeat("x", 200_000) + "\n",
			want:    []Var{{"BIG", strings.Repeat("x", 200_000), 1}},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			path := writeFixture(t, tt.content)
			got, err := Parse(path)
			if err != nil {
				t.Fatalf("Parse() error = %v", err)
			}
			if got.Path != path {
				t.Errorf("Path = %q, want %q", got.Path, path)
			}
			if !reflect.DeepEqual(got.Vars, tt.want) {
				t.Errorf("Vars = %#v, want %#v", got.Vars, tt.want)
			}
		})
	}
}

func TestParseSyntaxErrors(t *testing.T) {
	tests := []struct {
		name     string
		content  string
		wantLine int
	}{
		{"missing equals", "A=1\nJUSTAWORD\n", 2},
		{"export without assignment", "export PORT\n", 1},
		{"empty key", "=value\n", 1},
		{"key starts with digit", "1ABC=x\n", 1},
		{"key with dash", "MY-KEY=x\n", 1},
		{"key with space", "MY KEY=x\n", 1},
		{"unterminated double quote", "A=1\nB=\"open\nstill open\n", 2},
		{"unterminated single quote", "A='open", 1},
		{"text after closing quote", `A="x"y`, 1},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			path := writeFixture(t, tt.content)
			_, err := Parse(path)

			var se *SyntaxError
			if !errors.As(err, &se) {
				t.Fatalf("Parse() error = %v, want *SyntaxError", err)
			}
			if se.Line != tt.wantLine || se.Path != path {
				t.Errorf("error at %s:%d, want %s:%d", se.Path, se.Line, path, tt.wantLine)
			}
		})
	}
}

// TestParseErrorsNeverContainFileContent guards the core invariant: a parse
// error must identify where, never what. Each fixture embeds a marker that
// stands in for a secret, placed where a careless error message would quote it.
func TestParseErrorsNeverContainFileContent(t *testing.T) {
	const secret = "sk_live_DO_NOT_LEAK"
	tests := []struct {
		name    string
		content string
	}{
		{"pasted secret with no key", secret + "\n"},
		{"secret in invalid key", "BAD KEY " + secret + "=x\n"},
		{"secret in unterminated value", `TOKEN="` + secret + "\n"},
		{"secret after closing quote", `TOKEN="ok"` + secret + "\n"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := Parse(writeFixture(t, tt.content))
			if err == nil {
				t.Fatal("Parse() error = nil, want syntax error")
			}
			if strings.Contains(err.Error(), secret) {
				t.Errorf("error message leaks file content: %q", err.Error())
			}
		})
	}
}

func TestParseMissingFile(t *testing.T) {
	path := filepath.Join(t.TempDir(), "does-not-exist.env")
	_, err := Parse(path)
	if !errors.Is(err, fs.ErrNotExist) {
		t.Fatalf("Parse() error = %v, want fs.ErrNotExist", err)
	}
	if !strings.Contains(err.Error(), path) {
		t.Errorf("error %q does not name the path", err.Error())
	}
}
