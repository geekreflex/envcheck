package cmd

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// project writes files into a fresh directory. Env fixtures are generated
// per test, never committed.
func project(t *testing.T, files map[string]string) string {
	t.Helper()
	dir := t.TempDir()
	for name, content := range files {
		p := filepath.Join(dir, filepath.FromSlash(name))
		if err := os.MkdirAll(filepath.Dir(p), 0o750); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(p, []byte(content), 0o600); err != nil {
			t.Fatal(err)
		}
	}
	return dir
}

// A project with one problem of every kind.
var messy = map[string]string{
	".env.example": "DATABASE_URL=\nSTRIPE_KEY=\nLEGACY_KEY=\n",
	".env":         "DATABASE_URL=\nLEGACY_KEY=x\n",
	"main.go":      "package main\n\nimport \"os\"\n\nfunc main() {\n\t_ = os.Getenv(\"DATABASE_URL\")\n\t_ = os.Getenv(\"STRIPE_KEY\")\n}\n",
	"web/app.ts":   "export const r = process.env.REDIS_URL\n",
}

var clean = map[string]string{
	".env.example": "PORT=\n",
	".env":         "PORT=8080\n",
	"server.js":    "listen(process.env.PORT)\n",
}

func TestRunCheck(t *testing.T) {
	tests := []struct {
		name        string
		files       map[string]string
		args        []string
		wantCode    int
		wantOut     []string
		wantErr     []string
		notInOut    []string
		emptyErr    bool
		emptyStdout bool
	}{
		{
			name:     "clean project",
			files:    clean,
			wantCode: ExitOK,
			wantOut:  []string{"No problems found"},
			emptyErr: true,
		},
		{
			name:     "problems without --ci are informational",
			files:    messy,
			wantCode: ExitOK,
			wantOut: []string{
				"STRIPE_KEY", "REDIS_URL", "web/app.ts:1", "LEGACY_KEY", "DATABASE_URL", "4 problems found",
			},
			emptyErr: true,
		},
		{
			name:     "problems with --ci exit 1 without an error message",
			files:    messy,
			args:     []string{"--ci"},
			wantCode: ExitProblems,
			wantOut:  []string{"4 problems found"},
			emptyErr: true,
		},
		{
			name:     "clean with --ci exits 0",
			files:    clean,
			args:     []string{"--ci"},
			wantCode: ExitOK,
		},
		{
			name:     "missing .env is skipped with a note",
			files:    map[string]string{".env.example": "PORT=\n", "a.js": "process.env.PORT"},
			args:     []string{"--ci"},
			wantCode: ExitOK,
			wantErr:  []string{"no ", "skipping"},
		},
		{
			name:     "missing explicit --env is an error",
			files:    map[string]string{".env.example": "PORT=\n"},
			args:     []string{"--env", "nope.env"},
			wantCode: ExitError,
			wantErr:  []string{"nope.env"},
		},
		{
			name:        "missing .env.example is an error",
			files:       map[string]string{"a.js": "process.env.PORT"},
			wantCode:    ExitError,
			wantErr:     []string{".env.example", "--example"},
			emptyStdout: true,
		},
		{
			name:     "malformed env file is an error naming the line",
			files:    map[string]string{".env.example": "PORT=\nBROKEN LINE\n"},
			wantCode: ExitError,
			wantErr:  []string{".env.example:2"},
		},
		{
			name: "config file is honoured",
			files: map[string]string{
				".envcheck.yaml": "example: .env.sample\nignore: [REDIS_*]\n",
				".env.sample":    "PORT=\n",
				".env":           "PORT=1\n",
				"a.js":           "process.env.PORT; process.env.REDIS_URL",
			},
			wantCode: ExitOK,
			wantOut:  []string{"No problems found"},
		},
		{
			name:     "malformed config is an error",
			files:    map[string]string{".envcheck.yaml": "exampel: x\n", ".env.example": ""},
			wantCode: ExitError,
			wantErr:  []string{".envcheck.yaml"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			dir := project(t, tt.files)
			args := tt.args
			// Relative flag paths resolve against the working directory, so
			// make them relative to the project for these tests.
			for i, a := range args {
				if i > 0 && args[i-1] == "--env" {
					args[i] = filepath.Join(dir, a)
				}
			}

			var stdout, stderr bytes.Buffer
			code := run(append([]string{dir}, args...), &stdout, &stderr)

			if code != tt.wantCode {
				t.Errorf("exit code = %d, want %d\nstdout:\n%s\nstderr:\n%s", code, tt.wantCode, stdout.String(), stderr.String())
			}
			for _, s := range tt.wantOut {
				if !strings.Contains(stdout.String(), s) {
					t.Errorf("stdout missing %q:\n%s", s, stdout.String())
				}
			}
			for _, s := range tt.wantErr {
				if !strings.Contains(stderr.String(), s) {
					t.Errorf("stderr missing %q:\n%s", s, stderr.String())
				}
			}
			if tt.emptyErr && stderr.Len() > 0 {
				t.Errorf("stderr = %q, want empty", stderr.String())
			}
			if tt.emptyStdout && stdout.Len() > 0 {
				t.Errorf("stdout = %q, want empty", stdout.String())
			}
		})
	}
}

func TestRunCheckJSON(t *testing.T) {
	dir := project(t, messy)
	var stdout, stderr bytes.Buffer
	if code := run([]string{dir, "--json", "--ci"}, &stdout, &stderr); code != ExitProblems {
		t.Fatalf("exit code = %d, want %d (stderr: %s)", code, ExitProblems, stderr.String())
	}

	var got struct {
		Problems []struct {
			Kind string `json:"kind"`
			Key  string `json:"key"`
		} `json:"problems"`
	}
	if err := json.Unmarshal(stdout.Bytes(), &got); err != nil {
		t.Fatalf("stdout is not JSON: %v\n%s", err, stdout.String())
	}
	kinds := map[string]string{}
	for _, p := range got.Problems {
		kinds[p.Key] = p.Kind
	}
	want := map[string]string{
		"STRIPE_KEY":   "missing_in_env",
		"REDIS_URL":    "undeclared_in_example",
		"LEGACY_KEY":   "unused",
		"DATABASE_URL": "empty_value",
	}
	for k, v := range want {
		if kinds[k] != v {
			t.Errorf("problem for %s = %q, want %q (all: %v)", k, kinds[k], v, kinds)
		}
	}
}

// TestRunCheckNeverPrintsValues runs the whole pipeline over files full of
// marker secrets and asserts none reach any output stream.
func TestRunCheckNeverPrintsValues(t *testing.T) {
	const secret = "sk_live_DO_NOT_LEAK"
	dir := project(t, map[string]string{
		".env.example": "API_KEY=" + secret + "_example\nUNUSED=" + secret + "_unused\n",
		".env":         "API_KEY=" + secret + "\nEXTRA=" + secret + "_extra\nEMPTY=\n",
		"a.js":         "process.env.API_KEY; process.env.EMPTY; process.env.NEW",
	})

	for _, args := range [][]string{{dir}, {dir, "--json"}, {dir, "--ci"}} {
		var stdout, stderr bytes.Buffer
		run(args, &stdout, &stderr)
		if out := stdout.String() + stderr.String(); strings.Contains(out, secret) {
			t.Errorf("%v output leaks a value:\n%s", args[1:], out)
		}
	}
}
