// Package check compares env files against each other and against code usage.
package check

import (
	"github.com/jerrynwosu/envcheck/internal/envfile"
	"github.com/jerrynwosu/envcheck/internal/scanner"
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
	Example *envfile.File
	Env     *envfile.File
	Refs    []scanner.Ref
	Ignore  []string
}

// Run compares the example file, the local env file and the code references.
func Run(in Input) Result {
	// TODO: build key sets, apply Ignore, emit one Problem per finding.
	return Result{}
}
