// Package config loads optional .envcheck.yaml settings from the project root.
package config

import (
	"errors"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path"
	"path/filepath"
	"strings"

	"go.yaml.in/yaml/v3"
)

// FileNames are the config file names discovered in the project root, in
// order of preference.
var FileNames = []string{".envcheck.yaml", ".envcheck.yml"}

// Scan limits which files the code scanners walk.
type Scan struct {
	Include []string `yaml:"include"`
	Exclude []string `yaml:"exclude"`
}

// Config mirrors the .envcheck.yaml schema. Flags override these values.
type Config struct {
	Example string   `yaml:"example"`
	Env     string   `yaml:"env"`
	Scan    Scan     `yaml:"scan"`
	Ignore  []string `yaml:"ignore"`
}

// Default returns the configuration used when no file is present.
func Default() Config {
	return Config{
		Example: ".env.example",
		Env:     ".env",
		Scan: Scan{
			Include: []string{"**"},
			// Dependencies, VCS metadata, and build output. Scanning them is
			// slow, and compiled bundles inline process.env reads that the
			// user never wrote.
			Exclude: []string{
				"**/node_modules/**",
				"**/.git/**",
				"**/vendor/**",
				"**/dist/**",
				"**/build/**",
				"**/.next/**",
				"**/.nuxt/**",
				"**/.output/**",
				"**/.svelte-kit/**",
				"**/coverage/**",
				"**/*.min.js",
			},
		},
		// Set by the OS, shell, runtime, or CI rather than by the project,
		// so they never belong in .env.example.
		Ignore: []string{
			"NODE_ENV", "CI", "HOME", "PATH", "PWD", "USER", "SHELL",
			"TMPDIR", "TERM", "TZ", "LANG",
		},
	}
}

// Load returns the configuration for the project in dir.
//
// If file is non-empty it must exist. Otherwise the first of FileNames found
// in dir is used, and if there is none the defaults apply.
//
// Values in the file are layered over Default: example, env, and
// scan.include replace the defaults when set, while scan.exclude and ignore
// add to them, so a project cannot accidentally start scanning
// node_modules by listing one extra exclude.
func Load(dir, file string) (Config, error) {
	cfg := Default()

	if file == "" {
		for _, name := range FileNames {
			candidate := filepath.Join(dir, name)
			if _, err := os.Stat(candidate); err == nil {
				file = candidate
				break
			} else if !errors.Is(err, fs.ErrNotExist) {
				return Config{}, fmt.Errorf("reading %s: %w", candidate, err)
			}
		}
		if file == "" {
			return cfg, nil
		}
	}

	f, err := os.Open(file) //nolint:gosec // G304: reading the user's config file is the purpose
	if err != nil {
		var pe *fs.PathError
		if errors.As(err, &pe) {
			err = pe.Err // its message would repeat the path
		}
		return Config{}, fmt.Errorf("reading %s: %w", file, err)
	}
	defer f.Close() //nolint:errcheck // read-only

	var fromFile Config
	dec := yaml.NewDecoder(f)
	dec.KnownFields(true) // a typo in a key should fail loudly, not be ignored
	if err := dec.Decode(&fromFile); err != nil && !errors.Is(err, io.EOF) {
		return Config{}, fmt.Errorf("parsing %s: %w", file, err)
	}

	if fromFile.Example != "" {
		cfg.Example = fromFile.Example
	}
	if fromFile.Env != "" {
		cfg.Env = fromFile.Env
	}
	if len(fromFile.Scan.Include) > 0 {
		cfg.Scan.Include = fromFile.Scan.Include
	}
	cfg.Scan.Exclude = append(cfg.Scan.Exclude, fromFile.Scan.Exclude...)
	cfg.Ignore = append(cfg.Ignore, fromFile.Ignore...)

	if err := cfg.validate(); err != nil {
		return Config{}, fmt.Errorf("parsing %s: %w", file, err)
	}
	return cfg, nil
}

// validate rejects malformed glob patterns up front, so they fail the run
// with a clear message instead of silently matching nothing.
func (c Config) validate() error {
	check := func(field string, patterns []string, segmented bool) error {
		for _, p := range patterns {
			segs := []string{p}
			if segmented {
				segs = strings.Split(p, "/")
			}
			for _, s := range segs {
				if _, err := path.Match(s, ""); err != nil {
					return fmt.Errorf("%s: invalid pattern %q", field, p)
				}
			}
		}
		return nil
	}
	if err := check("scan.include", c.Scan.Include, true); err != nil {
		return err
	}
	if err := check("scan.exclude", c.Scan.Exclude, true); err != nil {
		return err
	}
	return check("ignore", c.Ignore, false)
}
