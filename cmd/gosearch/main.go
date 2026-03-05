// Package main provides the entry point for the gosearch CLI application.
package main

import (
	"fmt"
	"os"

	"github.com/abuiliazeed/gosearch/pkg/cli"
)

// Version info injected by goreleaser
var (
	version = "dev"
	commit  = "none"
	date    = "unknown"
)

func main() {
	cli.SetVersion(version, commit, date)
	if err := cli.Execute(); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}
