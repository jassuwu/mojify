package main

import (
	"fmt"
	"os"

	"github.com/jass/mojify/packages/core/internal/cli"
	"github.com/jass/mojify/packages/core/internal/media"
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
		info, err := media.Probe(cmd.InputPath)
		if err != nil {
			fmt.Fprintf(os.Stderr, "probe failed: %v\n", err)
			os.Exit(1)
		}
		fmt.Printf("input: %s\n", cmd.InputPath)
		fmt.Printf("video: %dx%d\n", info.Width, info.Height)
		fmt.Printf("fps: %.3f\n", info.FPS)
		fmt.Printf("frames: %d\n", info.FrameCount)
		fmt.Printf("duration: %.3fs\n", info.DurationSeconds)
	default:
		fmt.Fprintln(os.Stderr, "unknown command")
		os.Exit(2)
	}
}
