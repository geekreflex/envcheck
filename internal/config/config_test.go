package config

import "testing"

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
	want := []string{"node_modules/**", ".git/**", "vendor/**"}
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

func TestLoadEmptyPathReturnsDefaults(t *testing.T) {
	got, err := Load("")
	if err != nil {
		t.Fatalf("Load(\"\") returned error: %v", err)
	}
	if got.Example != Default().Example || got.Env != Default().Env {
		t.Errorf("Load(\"\") = %+v, want defaults %+v", got, Default())
	}
}
