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

func TestRunExportRejectsExistingOutputBeforeResolvingPlatformURL(t *testing.T) {
	dir := t.TempDir()
	output := filepath.Join(dir, "out.mp4")
	if err := os.WriteFile(output, []byte("existing"), 0o644); err != nil {
		t.Fatalf("write output fixture: %v", err)
	}
	argsPath := filepath.Join(dir, "yt-dlp-args.txt")
	fake := writeFakeYTDLP(t, fakeYTDLPOptions{ArgsPath: argsPath})

	err := runExportWithOptions(context.Background(), "https://example.com/watch?v=demo", output, io.Discard, ExportOptions{}, exportRunnerOptions{
		YTDLPPath: fake.Path,
	})
	if err == nil || !strings.Contains(err.Error(), "output exists") {
		t.Fatalf("error = %v, want existing output rejection", err)
	}
	if _, err := os.Stat(argsPath); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("yt-dlp was invoked before output preflight, stat err = %v", err)
	}
}

func TestRunExportUsesOriginalURLForProgressAndResolvedPathForExport(t *testing.T) {
	fake := writeFakeYTDLP(t, fakeYTDLPOptions{})
	exportErr := errors.New("stop after export handoff")
	var gotInputPath string
	var gotInputLabel string

	err := runExportWithOptions(context.Background(), "https://example.com/watch?v=demo", "out.mp4", io.Discard, ExportOptions{}, exportRunnerOptions{
		YTDLPPath: fake.Path,
		ExportMP4: func(ctx context.Context, inputPath string, outputPath string, stderr io.Writer, options exporter.Options) error {
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
