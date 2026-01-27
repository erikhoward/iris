// Iris CLI - AI agent development command-line interface.
package main

import (
	"os"

	"github.com/erikhoward/iris/cli/commands"
)

func main() {
	if err := commands.Execute(); err != nil {
		os.Exit(1)
	}
}
