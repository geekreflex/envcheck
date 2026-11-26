package check

import (
	"reflect"
	"testing"

	"github.com/geekreflex/envcheck/internal/envfile"
	"github.com/geekreflex/envcheck/internal/scanner"
)

// vars builds an env file from key/value pairs, numbering lines from 1.
func vars(path string, kv ...string) *envfile.File {
	f := &envfile.File{Path: path}
	for i := 0; i+1 < len(kv); i += 2 {
		f.Vars = append(f.Vars, envfile.Var{Key: kv[i], Value: kv[i+1], Line: i/2 + 1})
	}
	return f
}

func TestRun(t *testing.T) {
	const ex, env = ".env.example", ".env"

	tests := []struct {
		name string
		in   Input
		want []Problem
	}{
		{
			name: "everything in order",
			in: Input{
				Example:      vars(ex, "A", "", "B", ""),
				Env:          vars(env, "A", "1", "B", "2"),
				Refs:         []scanner.Ref{{Key: "A", File: "a.go", Line: 3}, {Key: "B", File: "b.ts", Line: 1}},
				ScannedFiles: 2,
			},
			want: []Problem{},
		},
		{
			name: "missing in env",
			in: Input{
				Example: vars(ex, "A", "", "B", ""),
				Env:     vars(env, "A", "1"),
			},
			want: []Problem{{Kind: MissingInEnv, Key: "B", File: ex, Line: 2}},
		},
		{
			name: "extra keys in env are not problems",
			in: Input{
				Example: vars(ex, "A", ""),
				Env:     vars(env, "A", "1", "LOCAL_ONLY", "x"),
			},
			want: []Problem{},
		},
		{
			name: "empty value in env",
			in: Input{
				Example: vars(ex, "A", ""),
				Env:     vars(env, "A", ""),
			},
			want: []Problem{{Kind: EmptyValue, Key: "A", File: env, Line: 1}},
		},
		{
			name: "empty values in example are fine",
			in: Input{
				Example: vars(ex, "A", ""),
				Env:     vars(env, "A", "set"),
			},
			want: []Problem{},
		},
		{
			name: "emptiness uses the last declaration",
			in: Input{
				Example: vars(ex, "A", "", "B", ""),
				Env:     vars(env, "A", "", "A", "set", "B", "set", "B", ""),
			},
			want: []Problem{{Kind: EmptyValue, Key: "B", File: env, Line: 4}},
		},
		{
			name: "no env file skips missing and empty",
			in: Input{
				Example: vars(ex, "A", "", "B", ""),
			},
			want: []Problem{},
		},
		{
			name: "undeclared reports the first occurrence once",
			in: Input{
				Example: vars(ex),
				Refs: []scanner.Ref{
					{Key: "REDIS_URL", File: "src/z.ts", Line: 1},
					{Key: "REDIS_URL", File: "src/a.ts", Line: 9},
					{Key: "REDIS_URL", File: "src/a.ts", Line: 4},
				},
				ScannedFiles: 2,
			},
			want: []Problem{{Kind: UndeclaredInExample, Key: "REDIS_URL", File: "src/a.ts", Line: 4}},
		},
		{
			name: "unused",
			in: Input{
				Example:      vars(ex, "USED", "", "LEGACY", ""),
				Refs:         []scanner.Ref{{Key: "USED", File: "main.go", Line: 1}},
				ScannedFiles: 1,
			},
			want: []Problem{{Kind: Unused, Key: "LEGACY", File: ex, Line: 2}},
		},
		{
			name: "no scanned files skips unused and undeclared",
			in: Input{
				Example: vars(ex, "A", ""),
			},
			want: []Problem{},
		},
		{
			name: "scanned files with no refs makes everything unused",
			in: Input{
				Example:      vars(ex, "A", ""),
				ScannedFiles: 3,
			},
			want: []Problem{{Kind: Unused, Key: "A", File: ex, Line: 1}},
		},
		{
			name: "duplicate example keys reported once",
			in: Input{
				Example: vars(ex, "A", "", "A", ""),
				Env:     vars(env),
			},
			want: []Problem{{Kind: MissingInEnv, Key: "A", File: ex, Line: 1}},
		},
		{
			name: "ignore by name and glob",
			in: Input{
				Example: vars(ex, "NEXT_PUBLIC_URL", "", "KEEP", ""),
				Env:     vars(env, "CI", ""),
				Refs: []scanner.Ref{
					{Key: "NODE_ENV", File: "a.js", Line: 1},
					{Key: "NEXT_PUBLIC_URL", File: "a.js", Line: 2},
				},
				ScannedFiles: 1,
				Ignore:       []string{"NODE_ENV", "NEXT_PUBLIC_*", "CI"},
			},
			want: []Problem{
				{Kind: MissingInEnv, Key: "KEEP", File: ex, Line: 2},
				{Kind: Unused, Key: "KEEP", File: ex, Line: 2},
			},
		},
		{
			name: "malformed ignore pattern is ignored, not fatal",
			in: Input{
				Example: vars(ex, "A", ""),
				Env:     vars(env),
				Ignore:  []string{"[bad"},
			},
			want: []Problem{{Kind: MissingInEnv, Key: "A", File: ex, Line: 1}},
		},
		{
			name: "ordered by kind then key",
			in: Input{
				Example: vars(ex, "Z", "", "Y", "", "UNUSED", ""),
				Env:     vars(env, "UNUSED", ""),
				Refs: []scanner.Ref{
					{Key: "Z", File: "a.go", Line: 1},
					{Key: "Y", File: "a.go", Line: 2},
					{Key: "NEW", File: "a.go", Line: 3},
				},
				ScannedFiles: 1,
			},
			want: []Problem{
				{Kind: MissingInEnv, Key: "Y", File: ex, Line: 2},
				{Kind: MissingInEnv, Key: "Z", File: ex, Line: 1},
				{Kind: UndeclaredInExample, Key: "NEW", File: "a.go", Line: 3},
				{Kind: Unused, Key: "UNUSED", File: ex, Line: 3},
				{Kind: EmptyValue, Key: "UNUSED", File: env, Line: 1},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := Run(tt.in).Problems
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("Run() problems =\n  %+v\nwant\n  %+v", got, tt.want)
			}
		})
	}
}
