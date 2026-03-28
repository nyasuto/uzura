package main

import (
	"fmt"
	"os"
)

// Version is set at build time or defaults to "dev".
var Version = "dev"

func main() {
	if len(os.Args) < 2 {
		fmt.Fprintln(os.Stderr, "usage: uzura <command> [options]")
		fmt.Fprintln(os.Stderr, "commands: parse, version")
		os.Exit(1)
	}

	switch os.Args[1] {
	case "version":
		fmt.Println("uzura", Version)
	case "parse":
		if err := runParse(); err != nil {
			fmt.Fprintln(os.Stderr, "error:", err)
			os.Exit(1)
		}
	default:
		fmt.Fprintf(os.Stderr, "unknown command: %s\n", os.Args[1])
		os.Exit(1)
	}
}
