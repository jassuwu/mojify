package cli

import (
	"fmt"
	"math"
	"strconv"
	"strings"
	"unicode"

	"github.com/jass/mojify/packages/core/internal/exporter"
	"github.com/jass/mojify/packages/core/internal/render"
)

type CommandKind int

const (
	HelpCommand CommandKind = iota
	VersionCommand
	PlayCommand
	ProbeCommand
	ExportCommand
)

type ExportOptions struct {
	Width           int
	FPS             float64
	Bitrate         string
	Overwrite       bool
	Stats           bool
	Workers         int
	HasAt           bool
	AtSeconds       float64
	HasDuration     bool
	DurationSeconds float64
	Recipe          render.Recipe
}

type Command struct {
	Kind       CommandKind
	InputPath  string
	OutputPath string
	Play       PlayOptions
	Export     ExportOptions
}

func Parse(args []string) (Command, error) {
	if len(args) == 0 {
		return Command{Kind: HelpCommand}, nil
	}

	switch args[0] {
	case "-h", "--help", "help":
		return Command{Kind: HelpCommand}, nil
	case "--version", "version":
		return Command{Kind: VersionCommand}, nil
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
  mojify play [--stats] [--no-audio] [--recipe <name>] <source>
                                                        Play source media in the terminal
  mojify probe <source>                                 Print source media and render metadata
  mojify export [options] <source> <output>             Export Mojify output to a supported file format
  mojify --version                                      Print the installed Mojify version
  mojify --help                                         Show this help

Source:
  <source> may be a local video file, local still image, or an HTTP(S) platform URL.
  Still image sources can be probed and exported, but not played.

Export output formats:
  .mp4, .webm, .mov, .gif, .apng, .png, .jpg, .jpeg, .txt, .ansi

Play options:
  --stats             Print playback timing stats after completion
  --no-audio          Disable live playback audio for play
  --recipe <name>     Built-in recipe preset: default, mono, ascii, blocks

Export options:
  --width <n>         Pixel width for media/image outputs; columns for text outputs
  --fps <n>           Output frames per second for time-based outputs
  --bitrate <value>   Video bitrate, digits optionally followed by k, K, m, or M
  --at <timestamp>    Start from timestamp: 10, 10s, 1:23, or 01:02:03.250
  --duration <value>  Limit time-based exports; valid for video and animated outputs
  --overwrite         Replace an existing output file
  --stats             Print export timing stats after completion
  --workers <n>       Render and rasterize with n workers
  --recipe <name>     Built-in recipe preset: default, mono, ascii, blocks

Requirements:
  FFmpeg and ffprobe must be available on PATH.
  yt-dlp is required for platform URL inputs.
  ffplay is required for live playback audio unless --no-audio is used.
`
}

func parseInputCommand(kind CommandKind, args []string) (Command, error) {
	if len(args) < 2 {
		return Command{}, fmt.Errorf("%s requires a source input", args[0])
	}

	var inputPath string
	stats := false
	noAudio := false
	recipe := render.DefaultRecipe()
	seenRecipe := false
	for i := 1; i < len(args); i++ {
		arg := args[i]
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
		case "--recipe":
			if kind != PlayCommand {
				return Command{}, fmt.Errorf("%s does not accept --recipe", args[0])
			}
			if seenRecipe {
				return Command{}, fmt.Errorf("%s accepts --recipe only once", args[0])
			}
			if i+1 >= len(args) {
				return Command{}, fmt.Errorf("%s requires a value for --recipe", args[0])
			}
			i++
			selected, err := render.RecipeByName(args[i])
			if err != nil {
				return Command{}, err
			}
			recipe = selected
			seenRecipe = true
		default:
			if strings.HasPrefix(arg, "--") {
				return Command{}, fmt.Errorf("unknown %s option %q", args[0], arg)
			}
			if inputPath != "" {
				return Command{}, fmt.Errorf("%s accepts exactly one source input", args[0])
			}
			inputPath = arg
		}
	}
	if inputPath == "" {
		return Command{}, fmt.Errorf("%s requires a source input", args[0])
	}
	if hasUnsupportedSourceProtocol(inputPath) {
		return Command{}, fmt.Errorf("%s accepts local source paths or HTTP(S) platform URLs only", args[0])
	}
	play := PlayOptions{
		Stats:   stats,
		NoAudio: noAudio,
		Recipe:  recipe,
	}
	return Command{Kind: kind, InputPath: inputPath, Play: play}, nil
}

func parseExportCommand(args []string) (Command, error) {
	paths := make([]string, 0, 2)
	options := ExportOptions{Recipe: render.DefaultRecipe()}
	seenOverwrite := false
	seenStats := false
	seenRecipe := false

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
		case "--at":
			if i+1 >= len(args) {
				return Command{}, fmt.Errorf("export requires a value for --at")
			}
			i++
			seconds, err := parseExportTimeValue(args[i])
			if err != nil {
				return Command{}, fmt.Errorf("export requires --at to be a timestamp: %w", err)
			}
			options.HasAt = true
			options.AtSeconds = seconds
		case "--duration":
			if i+1 >= len(args) {
				return Command{}, fmt.Errorf("export requires a value for --duration")
			}
			i++
			seconds, err := parseExportTimeValue(args[i])
			if err != nil || seconds <= 0 {
				return Command{}, fmt.Errorf("export requires --duration to be greater than 0")
			}
			options.HasDuration = true
			options.DurationSeconds = seconds
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
		case "--recipe":
			if seenRecipe {
				return Command{}, fmt.Errorf("export accepts --recipe only once")
			}
			if i+1 >= len(args) {
				return Command{}, fmt.Errorf("export requires a value for --recipe")
			}
			i++
			recipe, err := render.RecipeByName(args[i])
			if err != nil {
				return Command{}, err
			}
			options.Recipe = recipe
			seenRecipe = true
		default:
			if strings.HasPrefix(arg, "--") {
				return Command{}, fmt.Errorf("unknown export option %q", arg)
			}
			if len(paths) == 2 {
				return Command{}, fmt.Errorf("export accepts exactly one source input and one output")
			}
			paths = append(paths, arg)
		}
	}

	if len(paths) != 2 {
		return Command{}, fmt.Errorf("export requires a source input and output path")
	}
	if hasUnsupportedSourceProtocol(paths[0]) {
		return Command{}, fmt.Errorf("export accepts local source paths or HTTP(S) platform URLs only")
	}
	if hasProtocolInput(paths[1]) {
		return Command{}, fmt.Errorf("export accepts local output file paths only")
	}
	format, err := exporter.ResolveOutputFormat(paths[1])
	if err != nil {
		return Command{}, err
	}
	if options.HasDuration && format.SingleFrame {
		return Command{}, fmt.Errorf("export --duration is valid only for video and animated outputs")
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

func parseExportTimeValue(value string) (float64, error) {
	value = strings.TrimSpace(value)
	if value == "" {
		return 0, fmt.Errorf("time value is required")
	}
	if strings.HasSuffix(value, "s") {
		seconds, err := strconv.ParseFloat(strings.TrimSuffix(value, "s"), 64)
		if err != nil || seconds < 0 || math.IsNaN(seconds) || math.IsInf(seconds, 0) {
			return 0, fmt.Errorf("invalid time value %q", value)
		}
		return seconds, nil
	}
	if strings.Contains(value, ":") {
		parts := strings.Split(value, ":")
		if len(parts) < 2 || len(parts) > 3 {
			return 0, fmt.Errorf("invalid time value %q", value)
		}
		total := 0.0
		for i, part := range parts {
			number, err := strconv.ParseFloat(part, 64)
			if err != nil || number < 0 || math.IsNaN(number) || math.IsInf(number, 0) {
				return 0, fmt.Errorf("invalid time value %q", value)
			}
			if i < len(parts)-1 && number >= 60 {
				return 0, fmt.Errorf("invalid time value %q", value)
			}
			total = total*60 + number
		}
		return total, nil
	}
	seconds, err := strconv.ParseFloat(value, 64)
	if err != nil || seconds < 0 || math.IsNaN(seconds) || math.IsInf(seconds, 0) {
		return 0, fmt.Errorf("invalid time value %q", value)
	}
	return seconds, nil
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
