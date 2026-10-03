// Command sopsy sets and uses secrets from a SOPS + age encrypted dotenv file.
package main

import (
	"os"

	"github.com/niclasedge/sopsy/internal/cli"
)

func main() {
	os.Exit(cli.Main())
}
