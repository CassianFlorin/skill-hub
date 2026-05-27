package main

import (
	"fmt"
	"os"

	"github.com/cassian/skill-hub/internal/cli"
)

func main() {
	workDir, err := os.Getwd()
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	if err := cli.Run(os.Args[1:], os.Stdout, os.Stderr, workDir); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
