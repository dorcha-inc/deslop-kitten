// Command vetkitten scores a pull request for review cost and policy
// compliance and prints a report a maintainer reads.
package main

import (
	"os"

	"github.com/jadidbourbaki/vetkitten/cmd/vetkitten/commands"
)

func main() {
	if err := commands.NewRoot().Execute(); err != nil {
		os.Exit(1)
	}
}
