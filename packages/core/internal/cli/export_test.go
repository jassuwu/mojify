package cli

import (
	"bytes"
	"context"
	"errors"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/jass/mojify/packages/core/internal/exporter"
	"github.com/jass/mojify/packages/core/internal/render"
)

func TestIsTerminalWriterReturnsFalseForPlainWriter(t *testing.T) {
	var writer bytes.Buffer

	if isTerminalWriter(&writer) {
		t.Fatal("isTerminalWriter returned true for bytes.Buffer")
	}
}

type fakeFDWriter struct{}

func (fakeFDWriter) Write(p []byte) (int, error) {
	return len(p), nil
}

func (fakeFDWriter) Fd() uintptr {
	return 2
}

func TestIsTerminalWriterReturnsFalseForNonFileWriterWithFD(t *testing.T) {
	if isTerminalWriter(fakeFDWriter{}) {
		t.Fatal("isTerminalWriter returned true for non-file writer")
	}
}

func TestRunExportRejectsMissingYTDLPForPlatformURL(t *testing.T) {
	var stderr bytes.Buffer
	err := runExportWithOptions(context.Background(), "https://example.com/watch?v=demo", "out.mp4", &stderr, ExportOptions{}, exportRunnerOptions{
		YTDLPPath: "definitely-missing-yt-dlp",
	})
	if err == nil || !strings.Contains(err.Error(), "yt-dlp is required for platform URLs") {
		t.Fatalf("error = %v, want missing yt-dlp message", err)
	}
}

func TestRunExportDefersOutputValidationToExporter(t *testing.T) {
	dir := t.TempDir()
	output := filepath.Join(dir, "out.mp4")
	if err := os.WriteFile(output, []byte("existing"), 0o644); err != nil {
		t.Fatalf("write output fixture: %v", err)
	}
	argsPath := filepath.Join(dir, "yt-dlp-args.txt")
	fake := writeFakeYTDLP(t, fakeYTDLPOptions{ArgsPath: argsPath})
	exportErr := errors.New("stop after export handoff")
	var gotInputPath string

	err := runExportWithOptions(context.Background(), "https://example.com/watch?v=demo", output, io.Discard, ExportOptions{}, exportRunnerOptions{
		YTDLPPath: fake.Path,
		Export: func(ctx context.Context, inputPath string, outputPath string, stderr io.Writer, options exporter.Options) error {
			gotInputPath = inputPath
			return exportErr
		},
	})
	if !errors.Is(err, exportErr) {
		t.Fatalf("error = %v, want export sentinel", err)
	}
	if _, err := os.Stat(argsPath); err != nil {
		t.Fatalf("yt-dlp was not invoked before export handoff: %v", err)
	}
	if !strings.HasSuffix(gotInputPath, "Demo_Title [abc123].mp4") {
		t.Fatalf("export input path = %q, want resolved downloaded file", gotInputPath)
	}
}

func TestRunExportUsesOriginalURLForProgressAndResolvedPathForExport(t *testing.T) {
	fake := writeFakeYTDLP(t, fakeYTDLPOptions{})
	exportErr := errors.New("stop after export handoff")
	var gotInputPath string
	var gotInputLabel string

	err := runExportWithOptions(context.Background(), "https://example.com/watch?v=demo", "out.mp4", io.Discard, ExportOptions{}, exportRunnerOptions{
		YTDLPPath: fake.Path,
		Export: func(ctx context.Context, inputPath string, outputPath string, stderr io.Writer, options exporter.Options) error {
			gotInputPath = inputPath
			gotInputLabel = options.InputLabel
			if _, err := os.Stat(inputPath); err != nil {
				t.Fatalf("resolved source missing before export: %v", err)
			}
			return exportErr
		},
	})
	if err == nil || !errors.Is(err, exportErr) {
		t.Fatalf("error = %v, want export sentinel", err)
	}
	if !strings.HasSuffix(gotInputPath, "Demo_Title [abc123].mp4") {
		t.Fatalf("export input path = %q, want resolved downloaded file", gotInputPath)
	}
	if gotInputLabel != "https://example.com/watch?v=demo" {
		t.Fatalf("InputLabel = %q, want original URL", gotInputLabel)
	}
}

func TestRunExportPassesTimeOptionsAndUsesGeneralExporter(t *testing.T) {
	exportErr := errors.New("stop after export handoff")
	var gotOptions exporter.Options

	err := runExportWithOptions(context.Background(), "clip.mov", "out.gif", io.Discard, ExportOptions{
		HasAt:           true,
		AtSeconds:       10.5,
		HasDuration:     true,
		DurationSeconds: 3,
		Width:           320,
		FPS:             12,
		Overwrite:       true,
	}, exportRunnerOptions{
		Export: func(ctx context.Context, inputPath string, outputPath string, stderr io.Writer, options exporter.Options) error {
			gotOptions = options
			return exportErr
		},
	})
	if !errors.Is(err, exportErr) {
		t.Fatalf("error = %v, want export sentinel", err)
	}
	if !gotOptions.HasAt || gotOptions.AtSeconds != 10.5 {
		t.Fatalf("At = (%v, %v), want (true, 10.5)", gotOptions.HasAt, gotOptions.AtSeconds)
	}
	if !gotOptions.HasDuration || gotOptions.DurationSeconds != 3 {
		t.Fatalf("Duration = (%v, %v), want (true, 3)", gotOptions.HasDuration, gotOptions.DurationSeconds)
	}
}

func TestRunExportPassesRecipeToExporter(t *testing.T) {
	exportErr := errors.New("stop after export handoff")
	var gotOptions exporter.Options

	err := runExportWithOptions(context.Background(), "clip.mov", "out.png", io.Discard, ExportOptions{
		Recipe: render.MustRecipeByName("blocks"),
	}, exportRunnerOptions{
		Export: func(ctx context.Context, inputPath string, outputPath string, stderr io.Writer, options exporter.Options) error {
			gotOptions = options
			return exportErr
		},
	})
	if !errors.Is(err, exportErr) {
		t.Fatalf("error = %v, want export sentinel", err)
	}
	if gotOptions.Recipe.Name != "blocks" {
		t.Fatalf("recipe = %q, want blocks", gotOptions.Recipe.Name)
	}
}

func TestRunExportAllowsStillSourceToSingleFrameOutput(t *testing.T) {
	exportErr := errors.New("stop after export handoff")
	source := "poster.png"
	var gotInputPath string
	var gotInputLabel string

	err := runExportWithOptions(context.Background(), source, "out.txt", io.Discard, ExportOptions{}, exportRunnerOptions{
		Export: func(ctx context.Context, inputPath string, outputPath string, stderr io.Writer, options exporter.Options) error {
			gotInputPath = inputPath
			gotInputLabel = options.InputLabel
			return exportErr
		},
	})
	if !errors.Is(err, exportErr) {
		t.Fatalf("error = %v, want export sentinel", err)
	}
	if gotInputPath != source {
		t.Fatalf("export input path = %q, want original local path", gotInputPath)
	}
	if gotInputLabel != source {
		t.Fatalf("InputLabel = %q, want original local path", gotInputLabel)
	}
}

func TestRunExportRejectsTimeBasedOutputForStillSource(t *testing.T) {
	calledExport := false

	err := runExportWithOptions(context.Background(), "poster.jpg", "out.gif", io.Discard, ExportOptions{}, exportRunnerOptions{
		Export: func(ctx context.Context, inputPath string, outputPath string, stderr io.Writer, options exporter.Options) error {
			calledExport = true
			return nil
		},
	})
	if err == nil || err.Error() != "still image sources can only export single-frame outputs: .png, .jpg, .jpeg, .txt, .ansi" {
		t.Fatalf("error = %v, want still-source output validation error", err)
	}
	if calledExport {
		t.Fatal("export function was called for invalid still-source output")
	}
}

func TestRunExportRejectsAtForStillSource(t *testing.T) {
	calledExport := false

	err := runExportWithOptions(context.Background(), "poster.jpeg", "out.png", io.Discard, ExportOptions{
		HasAt:     true,
		AtSeconds: 1,
	}, exportRunnerOptions{
		Export: func(ctx context.Context, inputPath string, outputPath string, stderr io.Writer, options exporter.Options) error {
			calledExport = true
			return nil
		},
	})
	if err == nil || err.Error() != "export --at is not valid for still image sources" {
		t.Fatalf("error = %v, want still-source --at validation error", err)
	}
	if calledExport {
		t.Fatal("export function was called with --at for still source")
	}
}

func TestRunExportRejectsDurationForStillSource(t *testing.T) {
	calledExport := false

	err := runExportWithOptions(context.Background(), "poster.png", "out.png", io.Discard, ExportOptions{
		HasDuration:     true,
		DurationSeconds: 1,
	}, exportRunnerOptions{
		Export: func(ctx context.Context, inputPath string, outputPath string, stderr io.Writer, options exporter.Options) error {
			calledExport = true
			return nil
		},
	})
	if err == nil || err.Error() != "export --duration is not valid for still image sources" {
		t.Fatalf("error = %v, want still-source --duration validation error", err)
	}
	if calledExport {
		t.Fatal("export function was called with --duration for still source")
	}
}
