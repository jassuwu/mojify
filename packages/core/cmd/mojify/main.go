package main

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"syscall"

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
	case cli.VersionCommand:
		fmt.Print(cli.VersionText())
	case cli.PlayCommand:
		ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
		defer stop()
		if err := cli.RunPlay(
			ctx,
			cmd.InputPath,
			os.Stdin,
			os.Stdout,
			os.Stderr,
			cli.PlayOptions{Stats: cmd.Stats, NoAudio: cmd.NoAudio},
		); err != nil {
			fmt.Fprintf(os.Stderr, "play failed: %v\n", err)
			os.Exit(1)
		}
	case cli.ProbeCommand:
		ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
		defer stop()
		if err := cli.RunProbe(ctx, cmd.InputPath, os.Stdout, os.Stderr); err != nil {
			fmt.Fprintf(os.Stderr, "probe failed: %v\n", err)
			os.Exit(1)
		}
	case cli.ExportCommand:
		ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
		defer stop()
		if err := cli.RunExport(
			ctx,
			cmd.InputPath,
			cmd.OutputPath,
			os.Stderr,
			cmd.Export,
		); err != nil {
			fmt.Fprintf(os.Stderr, "export failed: %v\n", err)
			os.Exit(1)
		}
	default:
		fmt.Fprintln(os.Stderr, "unknown command")
		os.Exit(2)
	}
}
