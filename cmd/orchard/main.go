package main

import (
	"fmt"
	"io"
	"os"
)

type serverRunner func(io.Writer) error

func run(args []string, stdout io.Writer, stderr io.Writer) int {
	return runWithServer(
		args,
		stdout,
		stderr,
		serveHTTP,
	)
}

func runWithServer(args []string, stdout io.Writer, stderr io.Writer, serve serverRunner) int {
	if len(args) == 0 {
		fmt.Fprintln(stderr, "usage: orchard <command>")
		return 2
	}

	switch args[0] {
	case "version":
		fmt.Fprintln(stdout, "orchard dev")
		return 0

	case "server":
		if err := serve(stdout); err != nil {
			fmt.Fprintln(stderr, "server error:", err)
			return 1
		}

		return 0

	default:
		fmt.Fprintln(stderr, "unknown command:", args[0])
		return 2
	}
}

func main() {
	os.Exit(run(os.Args[1:], os.Stdout, os.Stderr))
}
