package cmd

import (
	"bytes"
	"errors"
	"fmt"
	"strings"
	"testing"
)

func TestRunVersion(t *testing.T) {
	var stdout, stderr bytes.Buffer

	if code := run([]string{"--version"}, &stdout, &stderr); code != ExitOK {
		t.Errorf("exit code = %d, want %d (stderr: %s)", code, ExitOK, stderr.String())
	}
	if got := stdout.String(); !strings.HasPrefix(got, "envcheck ") {
		t.Errorf("version output = %q, want it to start with \"envcheck \"", got)
	}
	if stderr.Len() != 0 {
		t.Errorf("stderr = %q, want empty", stderr.String())
	}
}

func TestRunHelpGoesToStdout(t *testing.T) {
	var stdout, stderr bytes.Buffer

	if code := run([]string{"--help"}, &stdout, &stderr); code != ExitOK {
		t.Errorf("exit code = %d, want %d", code, ExitOK)
	}

	// Every documented flag must appear in help; a flag that exists but is
	// undocumented is a bug for a tool people run in CI.
	for _, flag := range []string{"--example", "--env", "--ci", "--json", "--config"} {
		if !strings.Contains(stdout.String(), flag) {
			t.Errorf("help output is missing %s", flag)
		}
	}
	if stderr.Len() != 0 {
		t.Errorf("stderr = %q, want empty", stderr.String())
	}
}

func TestRunUnknownFlagExitsWithError(t *testing.T) {
	var stdout, stderr bytes.Buffer

	code := run([]string{"--definitely-not-a-flag"}, &stdout, &stderr)
	if code != ExitError {
		t.Errorf("exit code = %d, want %d", code, ExitError)
	}
	if !strings.Contains(stderr.String(), "envcheck:") {
		t.Errorf("stderr = %q, want it to carry the envcheck prefix", stderr.String())
	}
}

func TestRunTooManyArgs(t *testing.T) {
	var stdout, stderr bytes.Buffer

	if code := run([]string{"dir-one", "dir-two"}, &stdout, &stderr); code != ExitError {
		t.Errorf("exit code = %d, want %d", code, ExitError)
	}
}

func TestClassify(t *testing.T) {
	// A completed check that found problems must be distinguishable from a
	// crash: ExitProblems (1) vs ExitError (2). CI pipelines rely on it.
	tests := []struct {
		name       string
		err        error
		want       int
		wantStderr bool
	}{
		{"success", nil, ExitOK, false},
		{"problems found", &problemsError{count: 4}, ExitProblems, false},
		{"wrapped problems found", fmt.Errorf("check: %w", &problemsError{count: 1}), ExitProblems, false},
		{"genuine failure", errors.New("cannot read .env"), ExitError, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var stderr bytes.Buffer

			if got := classify(tt.err, &stderr); got != tt.want {
				t.Errorf("classify() = %d, want %d", got, tt.want)
			}
			if gotStderr := stderr.Len() > 0; gotStderr != tt.wantStderr {
				t.Errorf("wrote to stderr = %v, want %v (got %q)", gotStderr, tt.wantStderr, stderr.String())
			}
		})
	}
}

func TestProblemsErrorMessage(t *testing.T) {
	if got, want := (&problemsError{count: 4}).Error(), "4 problems found"; got != want {
		t.Errorf("Error() = %q, want %q", got, want)
	}
}
