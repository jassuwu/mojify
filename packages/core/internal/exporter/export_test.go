package exporter

import (
	"context"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/jass/mojify/packages/core/internal/render"
)

func TestCheckOutputPathRejectsExistingWithoutOverwrite(t *testing.T) {
	dir := t.TempDir()
	output := filepath.Join(dir, "out.mp4")
	if err := os.WriteFile(output, []byte("existing"), 0o644); err != nil {
		t.Fatalf("write output fixture: %v", err)
	}
	err := checkOutputPath(output, Options{})
	if err == nil {
		t.Fatal("checkOutputPath returned nil error for existing output")
	}
}

func TestCheckOutputPathAllowsOverwrite(t *testing.T) {
	dir := t.TempDir()
	output := filepath.Join(dir, "out.mp4")
	if err := os.WriteFile(output, []byte("existing"), 0o644); err != nil {
		t.Fatalf("write output fixture: %v", err)
	}
	err := checkOutputPath(output, Options{Overwrite: true})
	if err != nil {
		t.Fatalf("checkOutputPath returned error: %v", err)
	}
}

func TestCheckOutputPathRejectsUnsupportedFormat(t *testing.T) {
	err := CheckOutputPath("out.webp", Options{})
	if err == nil {
		t.Fatal("CheckOutputPath returned nil error for unsupported format")
	}
}

func TestCheckOutputPathRejectsDurationForSingleFrameFormat(t *testing.T) {
	err := CheckOutputPath("out.png", Options{HasDuration: true, DurationSeconds: 3})
	if err == nil {
		t.Fatal("CheckOutputPath returned nil error for duration with still output")
	}
}

func TestProgressDurationUsesSelectedWindowAfterAt(t *testing.T) {
	got := progressDuration(12, Options{HasAt: true, AtSeconds: 5})
	if got != 7 {
		t.Fatalf("progressDuration = %v, want 7", got)
	}
}

func TestProgressDurationDoesNotGoNegativeAfterAt(t *testing.T) {
	got := progressDuration(4, Options{HasAt: true, AtSeconds: 5})
	if got != 0 {
		t.Fatalf("progressDuration = %v, want 0", got)
	}
}

func TestProgressDurationPrefersExplicitDuration(t *testing.T) {
	got := progressDuration(12, Options{HasAt: true, AtSeconds: 5, HasDuration: true, DurationSeconds: 2})
	if got != 2 {
		t.Fatalf("progressDuration = %v, want 2", got)
	}
}

func TestProgressDurationClampsExplicitDurationToRemainingSource(t *testing.T) {
	got := progressDuration(5, Options{HasAt: true, AtSeconds: 4, HasDuration: true, DurationSeconds: 10})
	if got != 1 {
		t.Fatalf("progressDuration = %v, want 1", got)
	}
}

func TestExportRoutesByFormatFamily(t *testing.T) {
	tests := []struct {
		output string
		want   string
	}{
		{"out.mp4", "time"},
		{"out.gif", "time"},
		{"out.png", "single"},
		{"out.txt", "text"},
	}
	for _, tc := range tests {
		t.Run(tc.output, func(t *testing.T) {
			oldTime := exportTimeBasedMediaFunc
			oldSingle := exportSingleFrameMediaFunc
			oldText := exportTextFunc
			t.Cleanup(func() {
				exportTimeBasedMediaFunc = oldTime
				exportSingleFrameMediaFunc = oldSingle
				exportTextFunc = oldText
			})
			called := ""
			exportTimeBasedMediaFunc = func(ctx context.Context, inputPath string, outputPath string, stderr io.Writer, options Options, format OutputFormat) error {
				called = "time"
				return nil
			}
			exportSingleFrameMediaFunc = func(ctx context.Context, inputPath string, outputPath string, stderr io.Writer, options Options, format OutputFormat) error {
				called = "single"
				return nil
			}
			exportTextFunc = func(ctx context.Context, inputPath string, outputPath string, stderr io.Writer, options Options, format OutputFormat) error {
				called = "text"
				return nil
			}
			if err := Export(context.Background(), "in.mov", tc.output, io.Discard, Options{Overwrite: true}); err != nil {
				t.Fatalf("Export returned error: %v", err)
			}
			if called != tc.want {
				t.Fatalf("called = %q, want %q", called, tc.want)
			}
		})
	}
}

func TestExportTextWritesSingleFrameFile(t *testing.T) {
	dir := t.TempDir()
	output := filepath.Join(dir, "frame.txt")
	frame := render.NewRGBFrame(2, 2, []byte{
		255, 255, 255, 0, 0, 0,
		0, 0, 0, 255, 255, 255,
	})
	err := exportSingleTextFrameForTest(frame, output, Options{
		Width:     2,
		Overwrite: true,
		Format: OutputFormat{
			Extension:   ".txt",
			Family:      OutputFamilyText,
			Text:        true,
			SingleFrame: true,
		},
	})
	if err != nil {
		t.Fatalf("exportSingleTextFrameForTest returned error: %v", err)
	}
	data, err := os.ReadFile(output)
	if err != nil {
		t.Fatalf("read output: %v", err)
	}
	if len(data) == 0 || !strings.Contains(string(data), "\n") {
		t.Fatalf("text output = %q, want non-empty multi-line text", string(data))
	}
}

func TestExportSingleTextFrameForTestUsesSelectedRecipe(t *testing.T) {
	output := filepath.Join(t.TempDir(), "out.txt")
	frame := render.NewRGBFrame(1, 1, []byte{255, 255, 255})

	err := exportSingleTextFrameForTest(frame, output, Options{
		Format: OutputFormat{Extension: ".txt", Family: OutputFamilyText, Text: true, SingleFrame: true},
		Recipe: render.MustRecipeByName("blocks"),
	})
	if err != nil {
		t.Fatalf("exportSingleTextFrameForTest returned error: %v", err)
	}
	content, err := os.ReadFile(output)
	if err != nil {
		t.Fatalf("ReadFile returned error: %v", err)
	}
	if !strings.Contains(string(content), "█") {
		t.Fatalf("text output = %q, want block glyph", string(content))
	}
}
