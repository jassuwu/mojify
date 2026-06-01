package cli

import (
	"fmt"
	"strings"
	"unicode"
)

type CommandKind int

const (
	HelpCommand CommandKind = iota
	PlayCommand
	ProbeCommand
)

type Command struct {
	Kind      CommandKind
	InputPath string
}

func Parse(args []string) (Command, error) {
	if len(args) == 0 {
		return Command{Kind: HelpCommand}, nil
	}

	switch args[0] {
	case "-h", "--help", "help":
		return Command{Kind: HelpCommand}, nil
	case "play":
		return parseInputCommand(PlayCommand, args)
	case "probe":
		return parseInputCommand(ProbeCommand, args)
	default:
		return Command{}, fmt.Errorf("unknown command %q", args[0])
	}
}

func HelpText() string {
	return `mojify

Terminal-first video playback with colored, edge-aware character frames.

Usage:
  mojify play <video>    Play a local video file in the terminal
  mojify probe <video>   Print media and render metadata
  mojify --help          Show this help

Requirements:
  FFmpeg and ffprobe must be available on PATH for v1.
`
}

func parseInputCommand(kind CommandKind, args []string) (Command, error) {
	if len(args) < 2 {
		return Command{}, fmt.Errorf("%s requires a video input", args[0])
	}
	if len(args) > 2 {
		return Command{}, fmt.Errorf("%s accepts exactly one video input", args[0])
	}
	if hasProtocolInput(args[1]) {
		return Command{}, fmt.Errorf("%s accepts local video file paths only", args[0])
	}
	return Command{Kind: kind, InputPath: args[1]}, nil
}

func hasProtocolInput(input string) bool {
	if input == "-" {
		return true
	}
	colon := strings.IndexByte(input, ':')
	if colon <= 0 {
		return false
	}
	scheme := input[:colon]
	if len(scheme) == 1 {
		rest := input[colon+1:]
		if strings.HasPrefix(rest, `\`) || strings.HasPrefix(rest, `/`) {
			return false
		}
	}
	for i, r := range scheme {
		if i == 0 && !unicode.IsLetter(r) {
			return false
		}
		if !unicode.IsLetter(r) && !unicode.IsDigit(r) && r != '+' && r != '-' && r != '.' {
			return false
		}
	}
	return true
}
