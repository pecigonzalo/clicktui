// Command clicktui is the entrypoint for the ClickUp terminal UI and CLI.
package main

import (
	"fmt"
	"os"

	"github.com/pecigonzalo/clicktui/internal/cli"
)

func main() {
	if err := run(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func run() error {
	return cli.New().Execute()
}
