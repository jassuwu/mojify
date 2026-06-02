package media

import (
	"context"
	"fmt"
	"io"
	"os/exec"
	"strings"
)

type MP4EncodeOptions struct {
	InputPath  string
	OutputPath string
	Width      int
	Height     int
	FPS        float64
	Bitrate    string
	Overwrite  bool
}

const defaultMP4VideoPreset = "veryfast"

func MP4EncodeArgs(options MP4EncodeOptions) ([]string, error) {
	if strings.TrimSpace(options.InputPath) == "" {
		return nil, fmt.Errorf("input path is required")
	}
	if strings.TrimSpace(options.OutputPath) == "" {
		return nil, fmt.Errorf("output path is required")
	}
	if options.Width <= 0 || options.Height <= 0 {
		return nil, fmt.Errorf("invalid output dimensions %dx%d", options.Width, options.Height)
	}
	if options.FPS <= 0 {
		return nil, fmt.Errorf("invalid FPS %s", formatFPS(options.FPS))
	}

	overwriteFlag := "-n"
	if options.Overwrite {
		overwriteFlag = "-y"
	}

	args := []string{
		"-v", "error",
		overwriteFlag,
		"-f", "rawvideo",
		"-pix_fmt", "rgb24",
		"-s", fmt.Sprintf("%dx%d", options.Width, options.Height),
		"-r", formatFPS(options.FPS),
		"-i", "pipe:0",
		"-i", options.InputPath,
		"-map", "0:v:0",
		"-map", "1:a?",
		"-c:v", "libx264",
		"-preset", defaultMP4VideoPreset,
		"-pix_fmt", "yuv420p",
		"-c:a", "aac",
		"-shortest",
	}

	if bitrate := strings.TrimSpace(options.Bitrate); bitrate != "" {
		args = append(args, "-b:v", bitrate)
	}
	args = append(args, options.OutputPath)

	return args, nil
}

func StartMP4EncoderContext(ctx context.Context, options MP4EncodeOptions, stderr io.Writer) (*exec.Cmd, io.WriteCloser, error) {
	args, err := MP4EncodeArgs(options)
	if err != nil {
		return nil, nil, err
	}

	cmd := exec.CommandContext(ctx, "ffmpeg", args...)
	if stderr != nil {
		cmd.Stderr = stderr
	}
	stdin, err := cmd.StdinPipe()
	if err != nil {
		return nil, nil, err
	}
	if err := cmd.Start(); err != nil {
		return nil, nil, err
	}
	return cmd, stdin, nil
}
