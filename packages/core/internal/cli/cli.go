package cli

import (
	"fmt"
	"math"
	"path/filepath"
	"strconv"
	"strings"
	"unicode"
)

type CommandKind int

const (
	HelpCommand CommandKind = iota
	PlayCommand
	ProbeCommand
	ExportCommand
)

type ExportOptions struct {
	Width     int
	FPS       float64
	Bitrate   string
	Overwrite bool
	Stats     bool
	Workers   int
}

type Command struct {
	Kind       CommandKind
	InputPath  string
	OutputPath string
	Stats      bool
	NoAudio    bool
	Export     ExportOptions
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
	case "export":
		return parseExportCommand(args)
	default:
		return Command{}, fmt.Errorf("unknown command %q", args[0])
	}
}

func HelpText() string {
	return `mojify

Terminal-first video playback with colored, edge-aware character frames.

Usage:
  mojify play [--stats] [--no-audio] <source>           Play source media in the terminal
  mojify probe <source>                                 Print source media and render metadata
  mojify export [options] <source> <output.mp4>         Export Mojify visuals to an MP4 file
  mojify --help                                         Show this help

Source:
  <source> may be a local video file or an HTTP(S) platform URL.

Play options:
  --stats             Print playback timing stats after completion
  --no-audio          Disable live playback audio for play

Export options:
  --width <px>        Output MP4 width in pixels
  --fps <n>           Output frames per second
  --bitrate <value>   Video bitrate, digits optionally followed by k, K, m, or M
  --overwrite         Replace an existing output file
  --stats             Print export timing stats after completion
  --workers <n>       Render and rasterize with n workers

Requirements:
  FFmpeg and ffprobe must be available on PATH.
  yt-dlp is required for platform URL inputs.
  ffplay is required for live playback audio unless --no-audio is used.
`
}

func parseInputCommand(kind CommandKind, args []string) (Command, error) {
	if len(args) < 2 {
		return Command{}, fmt.Errorf("%s requires a video input", args[0])
	}

	var inputPath string
	stats := false
	noAudio := false
	for _, arg := range args[1:] {
		switch arg {
		case "--stats":
			if kind != PlayCommand {
				return Command{}, fmt.Errorf("%s does not accept --stats", args[0])
			}
			if stats {
				return Command{}, fmt.Errorf("%s accepts --stats only once", args[0])
			}
			stats = true
		case "--no-audio":
			if kind != PlayCommand {
				return Command{}, fmt.Errorf("%s does not accept --no-audio", args[0])
			}
			if noAudio {
				return Command{}, fmt.Errorf("%s accepts --no-audio only once", args[0])
			}
			noAudio = true
		default:
			if inputPath != "" {
				return Command{}, fmt.Errorf("%s accepts exactly one video input", args[0])
			}
			inputPath = arg
		}
	}
	if inputPath == "" {
		return Command{}, fmt.Errorf("%s requires a video input", args[0])
	}
	if hasUnsupportedSourceProtocol(inputPath) {
		return Command{}, fmt.Errorf("%s accepts local video file paths or HTTP(S) platform URLs only", args[0])
	}
	return Command{Kind: kind, InputPath: inputPath, Stats: stats, NoAudio: noAudio}, nil
}

func parseExportCommand(args []string) (Command, error) {
	paths := make([]string, 0, 2)
	options := ExportOptions{}
	seenOverwrite := false
	seenStats := false

	for i := 1; i < len(args); i++ {
		arg := args[i]
		switch arg {
		case "--width":
			if i+1 >= len(args) {
				return Command{}, fmt.Errorf("export requires a value for --width")
			}
			i++
			width, err := strconv.Atoi(args[i])
			if err != nil || width <= 0 {
				return Command{}, fmt.Errorf("export requires --width to be greater than 0")
			}
			options.Width = width
		case "--fps":
			if i+1 >= len(args) {
				return Command{}, fmt.Errorf("export requires a value for --fps")
			}
			i++
			fps, err := strconv.ParseFloat(args[i], 64)
			if err != nil || fps <= 0 || math.IsNaN(fps) || math.IsInf(fps, 0) {
				return Command{}, fmt.Errorf("export requires --fps to be greater than 0")
			}
			options.FPS = fps
		case "--bitrate":
			if i+1 >= len(args) {
				return Command{}, fmt.Errorf("export requires a value for --bitrate")
			}
			i++
			if !isValidExportBitrate(args[i]) {
				return Command{}, fmt.Errorf("export requires --bitrate to be digits optionally ending in k, K, m, or M")
			}
			options.Bitrate = args[i]
		case "--overwrite":
			if seenOverwrite {
				return Command{}, fmt.Errorf("export accepts --overwrite only once")
			}
			seenOverwrite = true
			options.Overwrite = true
		case "--stats":
			if seenStats {
				return Command{}, fmt.Errorf("export accepts --stats only once")
			}
			seenStats = true
			options.Stats = true
		case "--no-audio":
			return Command{}, fmt.Errorf("export does not accept --no-audio")
		case "--workers":
			if i+1 >= len(args) {
				return Command{}, fmt.Errorf("export requires a value for --workers")
			}
			i++
			workers, err := strconv.Atoi(args[i])
			if err != nil || workers <= 0 {
				return Command{}, fmt.Errorf("export requires --workers to be greater than 0")
			}
			options.Workers = workers
		default:
			if strings.HasPrefix(arg, "--") {
				return Command{}, fmt.Errorf("unknown export option %q", arg)
			}
			if len(paths) == 2 {
				return Command{}, fmt.Errorf("export accepts exactly one input and one output")
			}
			paths = append(paths, arg)
		}
	}

	if len(paths) != 2 {
		return Command{}, fmt.Errorf("export requires an input video and output MP4 path")
	}
	if hasUnsupportedSourceProtocol(paths[0]) {
		return Command{}, fmt.Errorf("export accepts local video file paths or HTTP(S) platform URLs only")
	}
	if hasProtocolInput(paths[1]) {
		return Command{}, fmt.Errorf("export accepts local output file paths only")
	}
	if !strings.EqualFold(filepath.Ext(paths[1]), ".mp4") {
		return Command{}, fmt.Errorf("export output must use a .mp4 extension")
	}

	return Command{
		Kind:       ExportCommand,
		InputPath:  paths[0],
		OutputPath: paths[1],
		Export:     options,
	}, nil
}

func isValidExportBitrate(value string) bool {
	if value == "" {
		return false
	}
	digitEnd := len(value)
	switch value[len(value)-1] {
	case 'k', 'K', 'm', 'M':
		digitEnd--
	}
	if digitEnd == 0 {
		return false
	}
	for _, r := range value[:digitEnd] {
		if r < '0' || r > '9' {
			return false
		}
	}
	return true
}

func isHTTPSource(input string) bool {
	lower := strings.ToLower(input)
	return strings.HasPrefix(lower, "http://") || strings.HasPrefix(lower, "https://")
}

func hasUnsupportedSourceProtocol(input string) bool {
	if isHTTPSource(input) {
		return false
	}
	return hasProtocolInput(input)
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
