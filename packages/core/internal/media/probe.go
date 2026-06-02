package media

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"os/exec"
	"strconv"
	"strings"
)

type Info struct {
	Width           int
	Height          int
	FPS             float64
	FrameCount      int
	DurationSeconds float64
	HasAudio        bool
}

func Probe(path string) (Info, error) {
	return ProbeContext(context.Background(), path)
}

func ProbeContext(ctx context.Context, path string) (Info, error) {
	cmd := exec.CommandContext(ctx, "ffprobe", "-v", "error", "-print_format", "json", "-show_streams", "-show_format", path)

	var stderr bytes.Buffer
	cmd.Stderr = &stderr

	output, err := cmd.Output()
	if err != nil {
		return Info{}, formatToolFailure("ffprobe", err, stderr.String())
	}

	return ParseProbeJSON(output)
}

func ParseProbeJSON(data []byte) (Info, error) {
	var probe probeJSON
	if err := json.Unmarshal(data, &probe); err != nil {
		return Info{}, fmt.Errorf("parse ffprobe json: %w", err)
	}

	hasAudio := false
	for _, stream := range probe.Streams {
		if stream.CodecType == "audio" {
			hasAudio = true
			break
		}
	}

	for _, stream := range probe.Streams {
		if stream.CodecType != "video" {
			continue
		}
		if stream.Width <= 0 || stream.Height <= 0 {
			return Info{}, fmt.Errorf("invalid video dimensions %dx%d", stream.Width, stream.Height)
		}

		fps, err := parseRate(stream.AvgFrameRate)
		if err != nil {
			return Info{}, fmt.Errorf("parse avg_frame_rate: %w", err)
		}

		frameCount, err := parseOptionalInt(stream.NBFrames)
		if err != nil {
			return Info{}, fmt.Errorf("parse nb_frames: %w", err)
		}

		duration, err := parseOptionalFloat(probe.Format.Duration)
		if err != nil {
			return Info{}, fmt.Errorf("parse duration: %w", err)
		}

		return Info{
			Width:           stream.Width,
			Height:          stream.Height,
			FPS:             fps,
			FrameCount:      frameCount,
			DurationSeconds: duration,
			HasAudio:        hasAudio,
		}, nil
	}

	return Info{}, fmt.Errorf("missing video stream")
}

func parseRate(rate string) (float64, error) {
	rate = strings.TrimSpace(rate)
	if rate == "" || rate == "N/A" {
		return 0, nil
	}

	parts := strings.Split(rate, "/")
	if len(parts) == 1 {
		return strconv.ParseFloat(parts[0], 64)
	}
	if len(parts) != 2 {
		return 0, fmt.Errorf("invalid rate %q", rate)
	}

	numerator, err := strconv.ParseFloat(parts[0], 64)
	if err != nil {
		return 0, err
	}
	denominator, err := strconv.ParseFloat(parts[1], 64)
	if err != nil {
		return 0, err
	}
	if denominator == 0 {
		return 0, fmt.Errorf("invalid zero denominator in %q", rate)
	}

	return numerator / denominator, nil
}

func parseOptionalInt(value string) (int, error) {
	value = strings.TrimSpace(value)
	if value == "" || value == "N/A" {
		return 0, nil
	}
	return strconv.Atoi(value)
}

func parseOptionalFloat(value string) (float64, error) {
	value = strings.TrimSpace(value)
	if value == "" || value == "N/A" {
		return 0, nil
	}
	return strconv.ParseFloat(value, 64)
}

type probeJSON struct {
	Streams []probeStreamJSON `json:"streams"`
	Format  probeFormatJSON   `json:"format"`
}

type probeStreamJSON struct {
	CodecType    string `json:"codec_type"`
	Width        int    `json:"width"`
	Height       int    `json:"height"`
	AvgFrameRate string `json:"avg_frame_rate"`
	NBFrames     string `json:"nb_frames"`
}

type probeFormatJSON struct {
	Duration string `json:"duration"`
}
