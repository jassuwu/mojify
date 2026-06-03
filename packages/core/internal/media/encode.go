package media

import (
	"context"
	"fmt"
	"io"
	"os/exec"
	"strings"
)

type MP4EncodeOptions struct {
	InputPath       string
	OutputPath      string
	Width           int
	Height          int
	FPS             float64
	Bitrate         string
	Overwrite       bool
	HasAt           bool
	AtSeconds       float64
	HasDuration     bool
	DurationSeconds float64
}

type EncodeFormat string

const (
	EncodeFormatMP4  EncodeFormat = "mp4"
	EncodeFormatWebM EncodeFormat = "webm"
	EncodeFormatMOV  EncodeFormat = "mov"
	EncodeFormatGIF  EncodeFormat = "gif"
	EncodeFormatAPNG EncodeFormat = "apng"
	EncodeFormatPNG  EncodeFormat = "png"
	EncodeFormatJPEG EncodeFormat = "jpeg"
)

type RawVideoEncodeOptions struct {
	Format          EncodeFormat
	InputPath       string
	OutputPath      string
	Width           int
	Height          int
	FPS             float64
	Bitrate         string
	Overwrite       bool
	IncludeAudio    bool
	HasAt           bool
	AtSeconds       float64
	HasDuration     bool
	DurationSeconds float64
}

const defaultMP4VideoPreset = "veryfast"

func MP4EncodeArgs(options MP4EncodeOptions) ([]string, error) {
	return RawVideoEncodeArgs(RawVideoEncodeOptions{
		Format:          EncodeFormatMP4,
		InputPath:       options.InputPath,
		OutputPath:      options.OutputPath,
		Width:           options.Width,
		Height:          options.Height,
		FPS:             options.FPS,
		Bitrate:         options.Bitrate,
		Overwrite:       options.Overwrite,
		IncludeAudio:    true,
		HasAt:           options.HasAt,
		AtSeconds:       options.AtSeconds,
		HasDuration:     options.HasDuration,
		DurationSeconds: options.DurationSeconds,
	})
}

func RawVideoEncodeArgs(options RawVideoEncodeOptions) ([]string, error) {
	if strings.TrimSpace(options.OutputPath) == "" {
		return nil, fmt.Errorf("output path is required")
	}
	if options.Width <= 0 || options.Height <= 0 {
		return nil, fmt.Errorf("invalid output dimensions %dx%d", options.Width, options.Height)
	}
	if options.FPS <= 0 {
		return nil, fmt.Errorf("invalid FPS %s", formatFPS(options.FPS))
	}
	if options.IncludeAudio && strings.TrimSpace(options.InputPath) == "" {
		return nil, fmt.Errorf("input path is required for audio-capable export")
	}
	if options.HasAt && options.AtSeconds < 0 {
		return nil, fmt.Errorf("start offset must be non-negative")
	}
	if options.HasDuration && options.DurationSeconds <= 0 {
		return nil, fmt.Errorf("duration must be greater than 0")
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
	}
	if options.IncludeAudio {
		if options.HasAt {
			args = append(args, "-ss", formatFPS(options.AtSeconds))
		}
		if options.HasDuration {
			args = append(args, "-t", formatFPS(options.DurationSeconds))
		}
		args = append(args, "-i", options.InputPath, "-map", "0:v:0", "-map", "1:a?")
	}

	switch options.Format {
	case EncodeFormatMP4:
		args = append(args, "-c:v", "libx264", "-preset", defaultMP4VideoPreset, "-pix_fmt", "yuv420p")
		if options.IncludeAudio {
			args = append(args, "-c:a", "aac", "-shortest")
		}
	case EncodeFormatMOV:
		args = append(args, "-c:v", "libx264", "-preset", defaultMP4VideoPreset, "-pix_fmt", "yuv420p")
		if options.IncludeAudio {
			args = append(args, "-c:a", "aac", "-shortest")
		}
	case EncodeFormatWebM:
		args = append(args, "-c:v", "libvpx-vp9", "-pix_fmt", "yuv420p")
		if options.IncludeAudio {
			args = append(args, "-c:a", "libopus", "-shortest")
		}
	case EncodeFormatGIF:
		args = append(args, "-filter_complex", "split[s0][s1];[s0]palettegen[p];[s1][p]paletteuse", "-loop", "0")
	case EncodeFormatAPNG:
		args = append(args, "-c:v", "apng", "-plays", "0")
	case EncodeFormatPNG:
		args = append(args, "-frames:v", "1", "-c:v", "png")
	case EncodeFormatJPEG:
		args = append(args, "-frames:v", "1", "-c:v", "mjpeg", "-q:v", "2")
	default:
		return nil, fmt.Errorf("unsupported encode format %q", options.Format)
	}

	if bitrate := strings.TrimSpace(options.Bitrate); bitrate != "" && (options.Format == EncodeFormatMP4 || options.Format == EncodeFormatMOV || options.Format == EncodeFormatWebM) {
		args = append(args, "-b:v", bitrate)
	}
	args = append(args, options.OutputPath)

	return args, nil
}

func StartMP4EncoderContext(ctx context.Context, options MP4EncodeOptions, stderr io.Writer) (*exec.Cmd, io.WriteCloser, error) {
	return StartRawVideoEncoderContext(ctx, RawVideoEncodeOptions{
		Format:          EncodeFormatMP4,
		InputPath:       options.InputPath,
		OutputPath:      options.OutputPath,
		Width:           options.Width,
		Height:          options.Height,
		FPS:             options.FPS,
		Bitrate:         options.Bitrate,
		Overwrite:       options.Overwrite,
		IncludeAudio:    true,
		HasAt:           options.HasAt,
		AtSeconds:       options.AtSeconds,
		HasDuration:     options.HasDuration,
		DurationSeconds: options.DurationSeconds,
	}, stderr)
}

func StartRawVideoEncoderContext(ctx context.Context, options RawVideoEncodeOptions, stderr io.Writer) (*exec.Cmd, io.WriteCloser, error) {
	args, err := RawVideoEncodeArgs(options)
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
		return nil, nil, formatToolStartError("ffmpeg", err)
	}
	return cmd, stdin, nil
}
