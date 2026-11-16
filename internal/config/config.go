// Package config loads optional .envcheck.yaml settings from the project root.
package config

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
			Exclude: []string{"node_modules/**", ".git/**", "vendor/**"},
		},
	}
}

// Load reads the config file at path, falling back to Default when path is
// empty or the file does not exist.
func Load(path string) (Config, error) {
	// TODO: discover .envcheck.yaml and unmarshal it over Default().
	return Default(), nil
}
