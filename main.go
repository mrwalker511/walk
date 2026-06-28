// Package main is the entry point for the walk CLI.
package main

import (
	"os"

	"github.com/walk-labs/walk/cmd"
)

func main() {
	if err := cmd.Execute(); err != nil {
		os.Exit(1)
	}
}