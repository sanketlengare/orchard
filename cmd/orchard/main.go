package main

import (
	"fmt"
	"io"
	"os"
)

func run(args []string, stdout, stderr io.Writer) int {
	if len(args) == 0 {
		fmt.Fprintln(stderr, "usage: orchard <command>")
		return 2
	}

	if args[0] == "version" {
		fmt.Fprintln(stdout, "orchard dev")
		return 0
	}

	fmt.Fprintln(stderr, "unknown command:", args[0])
	return 2
}

func main() {
	os.Exit(run(os.Args[1:], os.Stdout, os.Stderr))
}
