// Package buildinfo reports version metadata for the running binary.
//
// Release builds inject the values via -ldflags (see the Makefile and
// .goreleaser.yaml). Builds produced by `go install` or `go build` carry no
// ldflags, so the values are recovered from the module metadata the Go
// toolchain embeds instead. Either way callers get something useful.
package buildinfo

import (
	"fmt"
	"runtime"
	"runtime/debug"
	"strings"
)

// Injected by the linker at release time; empty in every other build.
var (
	version string
	commit  string
	date    string
)

// DevVersion is the version reported when nothing better is available.
const DevVersion = "dev"

// Info describes the running binary.
type Info struct {
	Version string
	Commit  string
	Date    string
	Dirty   bool
	Go      string
	OS      string
	Arch    string
}

// Read resolves build metadata, preferring linker-injected values and falling
// back to the embedded module and VCS metadata.
func Read() Info {
	info := Info{
		Version: version,
		Commit:  commit,
		Date:    date,
		Go:      runtime.Version(),
		OS:      runtime.GOOS,
		Arch:    runtime.GOARCH,
	}

	if bi, ok := debug.ReadBuildInfo(); ok {
		if info.Version == "" && bi.Main.Version != "" && bi.Main.Version != "(devel)" {
			info.Version = bi.Main.Version
		}
		for _, s := range bi.Settings {
			switch s.Key {
			case "vcs.revision":
				if info.Commit == "" {
					info.Commit = s.Value
				}
			case "vcs.time":
				if info.Date == "" {
					info.Date = s.Value
				}
			case "vcs.modified":
				info.Dirty = s.Value == "true"
			}
		}
	}

	if info.Version == "" {
		info.Version = DevVersion
	}
	return info
}

// ShortCommit returns the abbreviated commit hash, or "" when unknown.
func (i Info) ShortCommit() string {
	if len(i.Commit) > 12 {
		return i.Commit[:12]
	}
	return i.Commit
}

// String renders the line printed by `envcheck --version`.
func (i Info) String() string {
	var b strings.Builder
	b.WriteString(i.Version)
	if c := i.ShortCommit(); c != "" {
		fmt.Fprintf(&b, " (%s", c)
		if i.Dirty {
			b.WriteString("-dirty")
		}
		b.WriteString(")")
	}
	if i.Date != "" {
		fmt.Fprintf(&b, " built %s", i.Date)
	}
	fmt.Fprintf(&b, " %s %s/%s", i.Go, i.OS, i.Arch)
	return b.String()
}
