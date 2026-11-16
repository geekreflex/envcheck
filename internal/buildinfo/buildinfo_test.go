package buildinfo

import "testing"

func TestReadFallsBackToDevVersion(t *testing.T) {
	// No ldflags are set under `go test`, so Read must still produce a
	// usable version rather than an empty string.
	got := Read()
	if got.Version == "" {
		t.Fatal("Version is empty; expected a fallback")
	}
	if got.Go == "" || got.OS == "" || got.Arch == "" {
		t.Errorf("runtime fields not populated: %+v", got)
	}
}

func TestShortCommit(t *testing.T) {
	tests := []struct {
		name   string
		commit string
		want   string
	}{
		{"empty", "", ""},
		{"short is returned whole", "abc123", "abc123"},
		{"exactly twelve", "0123456789ab", "0123456789ab"},
		{"full hash is truncated", "0123456789abcdef0123456789abcdef01234567", "0123456789ab"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := (Info{Commit: tt.commit}).ShortCommit(); got != tt.want {
				t.Errorf("ShortCommit() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestInfoString(t *testing.T) {
	tests := []struct {
		name string
		info Info
		want string
	}{
		{
			name: "version only",
			info: Info{Version: "dev", Go: "go1.26.0", OS: "linux", Arch: "amd64"},
			want: "dev go1.26.0 linux/amd64",
		},
		{
			name: "full release build",
			info: Info{
				Version: "v1.2.3",
				Commit:  "0123456789abcdef",
				Date:    "2026-11-16T00:00:00Z",
				Go:      "go1.26.0",
				OS:      "darwin",
				Arch:    "arm64",
			},
			want: "v1.2.3 (0123456789ab) built 2026-11-16T00:00:00Z go1.26.0 darwin/arm64",
		},
		{
			name: "dirty tree is marked",
			info: Info{Version: "v1.2.3", Commit: "abc123", Dirty: true, Go: "go1.26.0", OS: "darwin", Arch: "arm64"},
			want: "v1.2.3 (abc123-dirty) go1.26.0 darwin/arm64",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.info.String(); got != tt.want {
				t.Errorf("String()\n got: %q\nwant: %q", got, tt.want)
			}
		})
	}
}
