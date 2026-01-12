// Package main is the entry point for the CodeAI CLI.
package main

import (
	"fmt"
	"os"

	"github.com/bargom/codeai/cmd/codeai/cmd"
)

func main() {
	if err := cmd.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
