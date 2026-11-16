// Package report renders check results for humans and for machines.
package report

import (
	"io"

	"github.com/jerrynwosu/envcheck/internal/check"
)

// Text writes the human-readable report to w.
func Text(w io.Writer, res check.Result) error {
	// TODO: group problems by kind, mirroring the README example output.
	return nil
}

// JSON writes the machine-readable report to w.
func JSON(w io.Writer, res check.Result) error {
	// TODO: encode res with a stable top-level shape.
	return nil
}
