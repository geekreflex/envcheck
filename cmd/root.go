package cmd

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
)

// version is set at build time via -ldflags.
var version = "0.1.0-dev"

var opts struct {
	example string
	env     string
	ci      bool
	json    bool
	config  string
}

var rootCmd = &cobra.Command{
	Use:   "envcheck",
	Short: "Catch missing environment variables before they break production",
	Long: `envcheck compares your .env.example, your local .env, and the variables
your code actually uses, then reports what is missing, unused, or empty.

It only ever works with variable names: secret values are never printed,
logged, or sent anywhere.`,
	Version:       version,
	SilenceUsage:  true,
	SilenceErrors: true,
	RunE: func(cmd *cobra.Command, args []string) error {
		dir := "."
		if len(args) > 0 {
			dir = args[0]
		}
		return run(dir)
	},
	Args: cobra.MaximumNArgs(1),
}

// Execute runs the root command and exits with a non-zero status on failure.
func Execute() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, "envcheck:", err)
		os.Exit(1)
	}
}

func init() {
	f := rootCmd.Flags()
	f.StringVar(&opts.example, "example", "", "path to the example env file (default .env.example)")
	f.StringVar(&opts.env, "env", "", "path to the local env file (default .env)")
	f.BoolVar(&opts.ci, "ci", false, "exit non-zero when problems are found")
	f.BoolVar(&opts.json, "json", false, "emit machine-readable JSON output")

	rootCmd.PersistentFlags().StringVar(&opts.config, "config", "", "path to .envcheck.yaml (default: discovered from the project root)")
}

// run is the entry point for the default check. Not implemented yet.
func run(dir string) error {
	return fmt.Errorf("not implemented yet (target dir %q)", dir)
}
