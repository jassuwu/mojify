package media

import (
	"context"
	"fmt"
	"io"
	"os/exec"
	"strings"
)

func FFplayAudioArgs(inputPath string) ([]string, error) {
	if strings.TrimSpace(inputPath) == "" {
		return nil, fmt.Errorf("input path is required")
	}
	return []string{
		"-nodisp",
		"-autoexit",
		"-loglevel", "error",
		"-vn",
		inputPath,
	}, nil
}

func StartFFplayAudioContext(ctx context.Context, inputPath string, stderr io.Writer) (*exec.Cmd, io.WriteCloser, error) {
	args, err := FFplayAudioArgs(inputPath)
	if err != nil {
		return nil, nil, err
	}

	cmd := exec.CommandContext(ctx, "ffplay", args...)
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
