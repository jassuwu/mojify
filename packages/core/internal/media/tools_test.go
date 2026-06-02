package media

import (
	"errors"
	"os/exec"
	"strings"
	"testing"
)

func TestFormatToolFailureMissingTool(t *testing.T) {
	err := formatToolFailure("ffmpeg", &exec.Error{Name: "ffmpeg", Err: exec.ErrNotFound}, "")
	if err == nil {
		t.Fatal("formatToolFailure returned nil")
	}
	got := err.Error()
	for _, want := range []string{
		"ffmpeg is required",
		"install ffmpeg",
	} {
		if !strings.Contains(got, want) {
			t.Fatalf("error %q missing %q", got, want)
		}
	}
}

func TestFormatToolFailurePreservesStderr(t *testing.T) {
	err := formatToolFailure("ffprobe", errors.New("exit status 1"), "invalid data")
	if err == nil {
		t.Fatal("formatToolFailure returned nil")
	}
	got := err.Error()
	want := "ffprobe failed: invalid data"
	if got != want {
		t.Fatalf("error = %q, want %q", got, want)
	}
}

func TestFormatToolStartErrorMissingTool(t *testing.T) {
	err := formatToolStartError("ffplay", &exec.Error{Name: "ffplay", Err: exec.ErrNotFound})
	if err == nil {
		t.Fatal("formatToolStartError returned nil")
	}
	got := err.Error()
	for _, want := range []string{
		"ffplay is required",
		"install ffplay",
	} {
		if !strings.Contains(got, want) {
			t.Fatalf("error %q missing %q", got, want)
		}
	}
}
