package report

import (
	"bytes"
	"encoding/json"
	"reflect"
	"testing"

	"github.com/geekreflex/envcheck/internal/check"
)

var files = Files{Example: ".env.example", Env: ".env"}

func TestText(t *testing.T) {
	tests := []struct {
		name string
		res  check.Result
		want string
	}{
		{"clean", check.Result{}, "✓ No problems found\n"},
		{
			name: "all sections, aligned locations",
			res: check.Result{Problems: []check.Problem{
				{Kind: check.MissingInEnv, Key: "STRIPE_WEBHOOK_SECRET", File: ".env.example", Line: 3},
				{Kind: check.UndeclaredInExample, Key: "REDIS_URL", File: "src/lib/cache.ts", Line: 4},
				{Kind: check.UndeclaredInExample, Key: "SENTRY_DSN", File: "cmd/api/main.go", Line: 22},
				{Kind: check.Unused, Key: "LEGACY_API_KEY", File: ".env.example", Line: 9},
				{Kind: check.EmptyValue, Key: "NEXT_PUBLIC_API_URL", File: ".env", Line: 2},
			}},
			want: `Missing in .env (declared in .env.example)
  ✗ STRIPE_WEBHOOK_SECRET  .env.example:3

Used in code but not declared in .env.example
  ✗ REDIS_URL   src/lib/cache.ts:4
  ✗ SENTRY_DSN  cmd/api/main.go:22

Declared in .env.example but never used in code
  ! LEGACY_API_KEY  .env.example:9

Empty values in .env
  ! NEXT_PUBLIC_API_URL  .env:2

5 problems found
`,
		},
		{
			name: "singular and no location",
			res:  check.Result{Problems: []check.Problem{{Kind: check.MissingInEnv, Key: "A"}}},
			want: "Missing in .env (declared in .env.example)\n  ✗ A\n\n1 problem found\n",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var b bytes.Buffer
			if err := Text(&b, tt.res, files); err != nil {
				t.Fatal(err)
			}
			if b.String() != tt.want {
				t.Errorf("Text() =\n%s\nwant\n%s", b.String(), tt.want)
			}
		})
	}
}

func TestJSON(t *testing.T) {
	tests := []struct {
		name string
		res  check.Result
		want string
	}{
		{"nil problems encode as empty array", check.Result{}, `{"problems":[]}`},
		{
			name: "omits empty file and line",
			res: check.Result{Problems: []check.Problem{
				{Kind: check.MissingInEnv, Key: "A"},
				{Kind: check.UndeclaredInExample, Key: "B", File: "a.ts", Line: 4},
			}},
			want: `{"problems":[{"kind":"missing_in_env","key":"A"},{"kind":"undeclared_in_example","key":"B","file":"a.ts","line":4}]}`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var b bytes.Buffer
			if err := JSON(&b, tt.res); err != nil {
				t.Fatal(err)
			}
			var got, want any
			if err := json.Unmarshal(b.Bytes(), &got); err != nil {
				t.Fatalf("output is not valid JSON: %v\n%s", err, b.String())
			}
			if err := json.Unmarshal([]byte(tt.want), &want); err != nil {
				t.Fatal(err)
			}
			if !reflect.DeepEqual(got, want) {
				t.Errorf("JSON() = %s, want %s", b.String(), tt.want)
			}
		})
	}
}
