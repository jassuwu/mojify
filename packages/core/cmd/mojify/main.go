package main

import (
	"fmt"
	"os"

	"github.com/jass/mojify/packages/core/internal/cli"
)

func main() {
	cmd, err := cli.Parse(os.Args[1:])
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		fmt.Fprintln(os.Stderr)
		fmt.Fprint(os.Stderr, cli.HelpText())
		os.Exit(2)
	}

	switch cmd.Kind {
	case cli.HelpCommand:
		fmt.Print(cli.HelpText())
	case cli.PlayCommand:
		fmt.Fprintf(os.Stderr, "mojify play is not implemented yet: %s\n", cmd.InputPath)
		os.Exit(1)
	case cli.ProbeCommand:
		fmt.Fprintf(os.Stderr, "mojify probe is not implemented yet: %s\n", cmd.InputPath)
		os.Exit(1)
	default:
		fmt.Fprintln(os.Stderr, "unknown command")
		os.Exit(2)
	}
}
