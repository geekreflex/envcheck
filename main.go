// Command envcheck reports environment variables that are missing, unused, or
// empty across your .env files and your source code.
package main

import (
	"os"

	"github.com/geekreflex/envcheck/cmd"
)

func main() {
	os.Exit(cmd.Execute())
}
