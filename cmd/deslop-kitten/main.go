// Command deslop-kitten scores a pull request for review cost and policy
// compliance and prints a report a maintainer reads.
package main

import (
	"os"

	"github.com/dorcha-inc/deslop-kitten/cmd/deslop-kitten/commands"
)

func main() {
	if err := commands.NewRoot().Execute(); err != nil {
		os.Exit(1)
	}
}
