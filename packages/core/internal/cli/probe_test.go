package cli

import (
	"bytes"
	"context"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"github.com/jass/mojify/packages/core/internal/media"
)

func TestPrintProbeInfoForLocalSource(t *testing.T) {
	var out bytes.Buffer
	printProbeInfo(&out, probeOutput{
		OriginalSource:  "clip.mp4",
		Width:           1920,
		Height:          1080,
		FPS:             60,
		FrameCount:      120,
		DurationSeconds: 2,
		HasAudio:        true,
		RenderCols:      120,
		RenderRows:      33,
	})
	got := out.String()
	for _, want := range []string{
		"input: clip.mp4\n",
		"video: 1920x1080\n",
		"fps: 60.000\n",
		"frames: 120\n",
		"duration: 2.000s\n",
		"audio: yes\n",
		"render-grid: 120x33 (sample terminal 120x40)\n",
	} {
		if !strings.Contains(got, want) {
			t.Fatalf("probe output missing %q in:\n%s", want, got)
		}
	}
	if strings.Contains(got, "resolved-source:") {
		t.Fatalf("local probe printed resolved-source: %q", got)
	}
}

func TestPrintProbeInfoForResolvedPlatformSource(t *testing.T) {
	var out bytes.Buffer
	printProbeInfo(&out, probeOutput{
		OriginalSource:      "https://example.com/watch?v=demo",
		ResolvedDisplayName: "Demo_Title [abc123].mp4",
		Width:               1280,
		Height:              720,
		FPS:                 30,
		FrameCount:          90,
		DurationSeconds:     3,
		HasAudio:            false,
		RenderCols:          120,
		RenderRows:          33,
	})
	got := out.String()
	for _, want := range []string{
		"input: https://example.com/watch?v=demo\n",
		"resolved-source: Demo_Title [abc123].mp4\n",
		"audio: no\n",
	} {
		if !strings.Contains(got, want) {
			t.Fatalf("probe output missing %q in:\n%s", want, got)
		}
	}
}

func TestRunProbeResolvesPlatformSource(t *testing.T) {
	fakeYTDLPPath := writeProbeFakeYTDLP(t)
	var stdout bytes.Buffer
	var stderr bytes.Buffer

	err := runProbeWithOptions(context.Background(), "https://example.com/watch?v=demo", &stdout, &stderr, probeRunnerOptions{
		YTDLPPath: fakeYTDLPPath,
		Probe: func(ctx context.Context, path string) (media.Info, error) {
			if !strings.HasSuffix(path, "Demo_Title [abc123].mp4") {
				t.Fatalf("probe path = %q, want resolved downloaded file", path)
			}
			return media.Info{
				Width:           1280,
				Height:          720,
				FPS:             30,
				FrameCount:      90,
				DurationSeconds: 3,
				HasAudio:        true,
			}, nil
		},
	})
	if err != nil {
		t.Fatalf("runProbeWithOptions returned error: %v", err)
	}
	if !strings.Contains(stderr.String(), "source media ready: Demo_Title [abc123].mp4") {
		t.Fatalf("stderr missing source ready status: %q", stderr.String())
	}
	if !strings.Contains(stdout.String(), "input: https://example.com/watch?v=demo\n") {
		t.Fatalf("stdout missing original input: %q", stdout.String())
	}
	if !strings.Contains(stdout.String(), "resolved-source: Demo_Title [abc123].mp4\n") {
		t.Fatalf("stdout missing resolved source: %q", stdout.String())
	}
}

func writeProbeFakeYTDLP(t *testing.T) string {
	t.Helper()
	if runtime.GOOS == "windows" {
		t.Skip("fake yt-dlp bash script contract tests are Unix-only")
	}

	path := filepath.Join(t.TempDir(), "yt-dlp")
	script := "#!/usr/bin/env bash\n" +
		"set -euo pipefail\n" +
		"for arg in \"$@\"; do\n" +
		"  if [[ \"${arg}\" == \"--dump-single-json\" ]]; then printf '%s\\n' '{\"id\":\"abc123\",\"title\":\"Demo Title\",\"is_live\":false}'; exit 0; fi\n" +
		"done\n" +
		"home=\"\"\n" +
		"prev=\"\"\n" +
		"for arg in \"$@\"; do\n" +
		"  if [[ \"${prev}\" == \"--paths\" && \"${arg}\" == home:* ]]; then home=\"${arg#home:}\"; fi\n" +
		"  prev=\"${arg}\"\n" +
		"done\n" +
		"if [[ -z \"${home}\" ]]; then echo 'missing home path' >&2; exit 7; fi\n" +
		"out=\"${home}/Demo_Title [abc123].mp4\"\n" +
		"mkdir -p \"${home}\"\n" +
		"printf 'fake media' > \"${out}\"\n" +
		"printf '%s\\n' \"${out}\"\n"
	if err := os.WriteFile(path, []byte(script), 0o755); err != nil {
		t.Fatalf("write fake yt-dlp: %v", err)
	}
	return path
}
