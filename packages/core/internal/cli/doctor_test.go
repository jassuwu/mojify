package cli

import (
	"bytes"
	"context"
	"errors"
	"os/exec"
	"strings"
	"testing"
	"time"

	"github.com/jass/mojify/packages/core/internal/doctor"
)

func TestRunDoctorSucceedsWithOptionalWarnings(t *testing.T) {
	var stdout bytes.Buffer
	err := runDoctorWithOptions(context.Background(), &stdout, doctor.Options{
		Runner: func(ctx context.Context, name string, args ...string) ([]byte, []byte, error) {
			switch name {
			case "ffmpeg":
				return []byte("ffmpeg version 8.0.1 Copyright\n"), nil, nil
			case "ffprobe":
				return []byte("ffprobe version 8.0.1 Copyright\n"), nil, nil
			default:
				return nil, nil, &exec.Error{Name: name, Err: exec.ErrNotFound}
			}
		},
		Timeout: time.Second,
	})

	if err != nil {
		t.Fatalf("runDoctorWithOptions returned error: %v", err)
	}
	got := stdout.String()
	for _, want := range []string{
		"mojify doctor",
		"ok    ffmpeg",
		"ok    ffprobe",
		"warn  ffplay",
		"warn  yt-dlp",
		"Mojify can play and export local media.",
	} {
		if !strings.Contains(got, want) {
			t.Fatalf("doctor output missing %q in:\n%s", want, got)
		}
	}
}

func TestRunDoctorFailsForRequiredErrors(t *testing.T) {
	var stdout bytes.Buffer
	err := runDoctorWithOptions(context.Background(), &stdout, doctor.Options{
		Runner: func(ctx context.Context, name string, args ...string) ([]byte, []byte, error) {
			return nil, nil, &exec.Error{Name: name, Err: exec.ErrNotFound}
		},
		Timeout: time.Second,
	})

	if err == nil {
		t.Fatal("runDoctorWithOptions returned nil error for missing required tools")
	}
	if !strings.Contains(err.Error(), "required runtime tools are missing or unhealthy") {
		t.Fatalf("error = %q, want required runtime wording", err.Error())
	}
	got := stdout.String()
	for _, want := range []string{
		"error ffmpeg",
		"error ffprobe",
		"Mojify cannot play or export local media until required tools are installed.",
	} {
		if !strings.Contains(got, want) {
			t.Fatalf("doctor output missing %q in:\n%s", want, got)
		}
	}
}

func TestRunDoctorReturnsCanceledForInterruptedReport(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	var stdout bytes.Buffer
	err := runDoctorWithOptions(ctx, &stdout, doctor.Options{
		Runner: func(ctx context.Context, name string, args ...string) ([]byte, []byte, error) {
			cancel()
			<-ctx.Done()
			return nil, nil, ctx.Err()
		},
		Timeout: time.Second,
	})

	if !errors.Is(err, context.Canceled) {
		t.Fatalf("runDoctorWithOptions error = %v, want context.Canceled", err)
	}
	got := stdout.String()
	if !strings.Contains(got, "Mojify doctor was interrupted before all checks completed.") {
		t.Fatalf("doctor output missing interrupted summary in:\n%s", got)
	}
}
