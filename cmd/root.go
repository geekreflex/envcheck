// Package cmd wires the envcheck command-line interface.
package cmd

import (
	"errors"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"

	"github.com/spf13/cobra"

	"github.com/geekreflex/envcheck/internal/buildinfo"
	"github.com/geekreflex/envcheck/internal/check"
	"github.com/geekreflex/envcheck/internal/config"
	"github.com/geekreflex/envcheck/internal/envfile"
	"github.com/geekreflex/envcheck/internal/report"
	"github.com/geekreflex/envcheck/internal/scanner"
)

// Exit codes. These are a public contract: CI pipelines branch on them, so
// treat a change here as a breaking change.
const (
	// ExitOK means the check ran and found nothing to fix.
	ExitOK = 0
	// ExitProblems means the check ran and found problems. Only returned
	// when --ci is set; without it a report is informational.
	ExitProblems = 1
	// ExitError means envcheck could not complete: bad flags, unreadable
	// files, malformed config.
	ExitError = 2
)

// options holds the resolved flag values for a single invocation.
type options struct {
	example string
	env     string
	ci      bool
	json    bool
	config  string
}

// Execute runs the CLI and returns the process exit code.
func Execute() int {
	return run(os.Args[1:], os.Stdout, os.Stderr)
}

// run is Execute with its environment injected, so tests can drive the CLI
// without touching os.Args or the real standard streams.
func run(args []string, stdout, stderr io.Writer) int {
	root := newRootCmd(stdout, stderr)
	root.SetArgs(args)

	return classify(root.Execute(), stderr)
}

// classify maps the outcome of a command run onto an exit code, reporting any
// genuine failure to stderr. A completed check that merely found problems is
// not a failure, so it is never printed as one.
func classify(err error, stderr io.Writer) int {
	if err == nil {
		return ExitOK
	}

	var pe *problemsError
	if errors.As(err, &pe) {
		return ExitProblems
	}

	fmt.Fprintln(stderr, "envcheck:", err)
	return ExitError
}

func newRootCmd(stdout, stderr io.Writer) *cobra.Command {
	var opts options

	cmd := &cobra.Command{
		Use:   "envcheck [path]",
		Short: "Catch missing environment variables before they break production",
		Long: `envcheck compares your .env.example, your local .env, and the variables
your code actually uses, then reports what is missing, unused, or empty.

It only ever works with variable names: secret values are never printed,
logged, or sent anywhere.`,
		Args:    cobra.MaximumNArgs(1),
		Version: buildinfo.Read().String(),

		// Errors and usage are rendered by run(), so cobra must not also
		// print them itself.
		SilenceUsage:  true,
		SilenceErrors: true,

		RunE: func(_ *cobra.Command, args []string) error {
			dir := "."
			if len(args) > 0 {
				dir = args[0]
			}
			return runCheck(dir, opts, stdout, stderr)
		},
	}

	cmd.SetOut(stdout)
	cmd.SetErr(stderr)
	cmd.SetVersionTemplate("envcheck {{.Version}}\n")

	f := cmd.Flags()
	f.StringVar(&opts.example, "example", "", "path to the example env file (default .env.example)")
	f.StringVar(&opts.env, "env", "", "path to the local env file (default .env)")
	f.BoolVar(&opts.ci, "ci", false, "exit non-zero when problems are found")
	f.BoolVar(&opts.json, "json", false, "emit machine-readable JSON output")
	f.StringVar(&opts.config, "config", "", "path to .envcheck.yaml (default: discovered from the project root)")

	return cmd
}

// problemsError signals that the check completed successfully but found
// problems, and --ci asked for a non-zero exit. It is deliberately distinct
// from a real failure so run() can tell the two apart.
type problemsError struct {
	count int
}

func (e *problemsError) Error() string {
	return fmt.Sprintf("%d problems found", e.count)
}

// runCheck runs the default end-to-end check against dir.
func runCheck(dir string, opts options, stdout, stderr io.Writer) error {
	info, err := os.Stat(dir)
	if err != nil {
		var pe *fs.PathError
		if errors.As(err, &pe) {
			err = pe.Err // its message would repeat the path
		}
		return fmt.Errorf("reading %s: %w", dir, err)
	}
	if !info.IsDir() {
		return fmt.Errorf("%s is not a directory", dir)
	}

	cfg, err := config.Load(dir, opts.config)
	if err != nil {
		return err
	}

	// Flags are paths as the user typed them; config values are relative
	// to the project directory.
	examplePath := pick(opts.example, filepath.Join(dir, cfg.Example))
	envPath := pick(opts.env, filepath.Join(dir, cfg.Env))

	example, err := envfile.Parse(examplePath)
	if errors.Is(err, fs.ErrNotExist) {
		return fmt.Errorf("%w (create it, or point to it with --example)", err)
	}
	if err != nil {
		return err
	}

	// A missing .env is normal in CI, where variables come from the
	// platform. Only an explicitly requested file must exist.
	env, err := envfile.Parse(envPath)
	if errors.Is(err, fs.ErrNotExist) && opts.env == "" {
		env = nil
		fmt.Fprintf(stderr, "envcheck: no %s found; skipping missing and empty value checks\n", envPath)
	} else if err != nil {
		return err
	}

	scan, err := scanner.Walk(dir, cfg)
	if err != nil {
		return err
	}

	res := check.Run(check.Input{
		Example:      example,
		Env:          env,
		Refs:         scan.Refs,
		ScannedFiles: scan.Files,
		Ignore:       cfg.Ignore,
	})

	if opts.json {
		err = report.JSON(stdout, res)
	} else {
		err = report.Text(stdout, res, report.Files{Example: examplePath, Env: envPath})
	}
	if err != nil {
		return fmt.Errorf("writing report: %w", err)
	}

	if opts.ci && !res.OK() {
		return &problemsError{count: len(res.Problems)}
	}
	return nil
}

// pick returns flag when it was set, otherwise fallback.
func pick(flag, fallback string) string {
	if flag != "" {
		return flag
	}
	return fallback
}
