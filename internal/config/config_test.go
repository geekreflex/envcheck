package config

import (
	"errors"
	"io/fs"
	"os"
	"path/filepath"
	"reflect"
	"slices"
	"strings"
	"testing"
)

func TestDefaultMatchesDocumentedPaths(t *testing.T) {
	// These defaults are part of the CLI's documented contract in the README.
	got := Default()
	if got.Example != ".env.example" {
		t.Errorf("Example = %q, want \".env.example\"", got.Example)
	}
	if got.Env != ".env" {
		t.Errorf("Env = %q, want \".env\"", got.Env)
	}
	if len(got.Scan.Include) == 0 {
		t.Error("Scan.Include is empty; the default must scan something")
	}
}

func TestDefaultExcludesVendoredTrees(t *testing.T) {
	// Scanning node_modules or .git is both slow and a source of false
	// positives, so the defaults must rule them out.
	want := []string{"**/node_modules/**", "**/.git/**", "**/vendor/**"}
	excluded := make(map[string]bool)
	for _, p := range Default().Scan.Exclude {
		excluded[p] = true
	}

	for _, p := range want {
		if !excluded[p] {
			t.Errorf("default Scan.Exclude is missing %q", p)
		}
	}
}

func TestLoadWithoutFileReturnsDefaults(t *testing.T) {
	got, err := Load(t.TempDir(), "")
	if err != nil {
		t.Fatalf("Load() returned error: %v", err)
	}
	if !reflect.DeepEqual(got, Default()) {
		t.Errorf("Load() = %+v, want defaults %+v", got, Default())
	}
}

func writeConfig(t *testing.T, dir, name, content string) string {
	t.Helper()
	p := filepath.Join(dir, name)
	if err := os.WriteFile(p, []byte(content), 0o600); err != nil {
		t.Fatal(err)
	}
	return p
}

func TestLoadDiscoversConfig(t *testing.T) {
	for _, name := range FileNames {
		t.Run(name, func(t *testing.T) {
			dir := t.TempDir()
			writeConfig(t, dir, name, "example: .env.sample\n")
			got, err := Load(dir, "")
			if err != nil {
				t.Fatal(err)
			}
			if got.Example != ".env.sample" {
				t.Errorf("Example = %q, want .env.sample", got.Example)
			}
		})
	}
}

func TestLoadLayersOverDefaults(t *testing.T) {
	dir := t.TempDir()
	writeConfig(t, dir, ".envcheck.yaml", `
env: .env.local
scan:
  include: ["src/**"]
  exclude: ["**/*.test.ts"]
ignore:
  - NEXT_PUBLIC_*
`)
	got, err := Load(dir, "")
	if err != nil {
		t.Fatal(err)
	}
	def := Default()

	if got.Example != def.Example {
		t.Errorf("Example = %q, want default %q when unset", got.Example, def.Example)
	}
	if got.Env != ".env.local" {
		t.Errorf("Env = %q, want .env.local", got.Env)
	}
	if !reflect.DeepEqual(got.Scan.Include, []string{"src/**"}) {
		t.Errorf("Include = %v, want it replaced", got.Scan.Include)
	}
	if want := append(slices.Clone(def.Scan.Exclude), "**/*.test.ts"); !reflect.DeepEqual(got.Scan.Exclude, want) {
		t.Errorf("Exclude = %v, want defaults plus the file's", got.Scan.Exclude)
	}
	if want := append(slices.Clone(def.Ignore), "NEXT_PUBLIC_*"); !reflect.DeepEqual(got.Ignore, want) {
		t.Errorf("Ignore = %v, want defaults plus the file's", got.Ignore)
	}
}

func TestLoadEmptyFile(t *testing.T) {
	dir := t.TempDir()
	writeConfig(t, dir, ".envcheck.yaml", "")
	got, err := Load(dir, "")
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(got, Default()) {
		t.Errorf("Load() = %+v, want defaults", got)
	}
}

func TestLoadErrors(t *testing.T) {
	tests := []struct {
		name    string
		content string
	}{
		{"unknown key", "exampel: .env.example\n"},
		{"wrong type", "ignore: NODE_ENV\n"},
		{"bad yaml", "scan: [\n"},
		{"bad exclude glob", "scan:\n  exclude: [\"[oops\"]\n"},
		{"bad ignore glob", "ignore: [\"[oops\"]\n"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			dir := t.TempDir()
			p := writeConfig(t, dir, ".envcheck.yaml", tt.content)
			_, err := Load(dir, "")
			if err == nil {
				t.Fatal("Load() returned nil error")
			}
			if !strings.Contains(err.Error(), p) {
				t.Errorf("error %q does not name the file", err)
			}
		})
	}
}

func TestLoadExplicitFileMustExist(t *testing.T) {
	dir := t.TempDir()
	_, err := Load(dir, filepath.Join(dir, "missing.yaml"))
	if !errors.Is(err, fs.ErrNotExist) {
		t.Fatalf("Load() error = %v, want fs.ErrNotExist", err)
	}
}
