package exporter

import (
	"bytes"
	"strconv"
	"strings"
	"testing"
	"time"
)

func TestEstimateExportFrameTotalUsesDurationAndExplicitFPS(t *testing.T) {
	info := InputProgressInfo{
		SourceFPS:       60,
		FrameCount:      600,
		DurationSeconds: 10,
	}
	layout := Layout{FPS: 12}
	options := Options{FPS: 12}

	total := estimateExportFrameTotal(info, layout, options)

	if total != 120 {
		t.Fatalf("total = %d, want 120", total)
	}
}

func TestEstimateExportFrameTotalUsesDurationAndProbedFPS(t *testing.T) {
	info := InputProgressInfo{
		SourceFPS:       29.97,
		FrameCount:      0,
		DurationSeconds: 8.008,
	}
	layout := Layout{FPS: 29.97}

	total := estimateExportFrameTotal(info, layout, Options{})

	if total != 240 {
		t.Fatalf("total = %d, want 240", total)
	}
}

func TestEstimateExportFrameTotalFallsBackToSourceFramesWithoutFPSConversion(t *testing.T) {
	info := InputProgressInfo{
		SourceFPS:       0,
		FrameCount:      240,
		DurationSeconds: 0,
	}
	layout := Layout{FPS: 24}

	total := estimateExportFrameTotal(info, layout, Options{})

	if total != 240 {
		t.Fatalf("total = %d, want 240", total)
	}
}

func TestEstimateExportFrameTotalRejectsSourceFramesWhenFPSConversionIsRequested(t *testing.T) {
	info := InputProgressInfo{
		SourceFPS:       60,
		FrameCount:      600,
		DurationSeconds: 0,
	}
	layout := Layout{FPS: 12}
	options := Options{FPS: 12}

	total := estimateExportFrameTotal(info, layout, options)

	if total != 0 {
		t.Fatalf("total = %d, want 0", total)
	}
}

func TestEstimateExportFrameTotalDoesNotUseDefaultFPSForUnknownSourceFPS(t *testing.T) {
	info := InputProgressInfo{
		SourceFPS:       0,
		FrameCount:      0,
		DurationSeconds: 10,
	}
	layout := Layout{FPS: 24}

	total := estimateExportFrameTotal(info, layout, Options{})

	if total != 0 {
		t.Fatalf("total = %d, want 0", total)
	}
}

func TestProgressReporterInteractiveKnownTotal(t *testing.T) {
	var out bytes.Buffer
	clock := &fakeClock{now: time.Unix(0, 0)}
	reporter := newProgressReporter(&out, progressReporterOptions{
		Interactive: true,
		TotalFrames: 10,
		Now:         clock.Now,
	})

	reporter.Start("in.mp4", "out.mp4", Layout{OutputWidth: 320, OutputHeight: 184, FPS: 24})
	reporter.Frame(1)
	reporter.Frame(5)
	reporter.AllFramesWritten(10)
	reporter.Finalizing()
	reporter.Complete("out.mp4")

	got := out.String()
	for _, want := range []string{
		"export: in.mp4 -> out.mp4\n",
		"output: 320x184 @ 24.000 fps\n",
		"\x1b[2K\rexporting video: 0/10 frames 0%",
		"\x1b[2K\rexporting video: 10/10 frames 100%\n",
		"finalizing mp4...\n",
		"export complete: out.mp4\n",
	} {
		if !strings.Contains(got, want) {
			t.Fatalf("progress output missing %q in %q", want, got)
		}
	}
	if strings.Contains(strings.ToLower(got), "eta") || strings.Contains(strings.ToLower(got), "remaining") {
		t.Fatalf("progress output contains ETA wording: %q", got)
	}
}

func TestProgressReporterUnknownTotalDoesNotPrintPercent(t *testing.T) {
	var out bytes.Buffer
	reporter := newProgressReporter(&out, progressReporterOptions{
		Interactive: false,
		TotalFrames: 0,
		Now:         time.Now,
	})

	reporter.Start("in.mp4", "out.mp4", Layout{OutputWidth: 320, OutputHeight: 184, FPS: 24})
	reporter.Frame(17)
	reporter.Frame(100)
	reporter.Frame(101)
	reporter.AllFramesWritten(117)
	reporter.Finalizing()

	got := out.String()
	for _, want := range []string{
		"exporting video: 0 frames\n",
		"exporting video: 100 frames\n",
		"exporting video: 117 frames\n",
	} {
		if !strings.Contains(got, want) {
			t.Fatalf("progress output missing %q in %q", want, got)
		}
	}
	if strings.Contains(got, "%") {
		t.Fatalf("unknown total printed percent: %q", got)
	}
	if strings.Contains(got, "\nexporting video: 17 frames\n") ||
		strings.Contains(got, "\nexporting video: 101 frames\n") {
		t.Fatalf("unknown-total non-TTY progress was not sparse: %q", got)
	}
}

func TestProgressReporterNonTTYPrintsSparseMilestones(t *testing.T) {
	var out bytes.Buffer
	reporter := newProgressReporter(&out, progressReporterOptions{
		Interactive: false,
		TotalFrames: 100,
		Now:         time.Now,
	})

	reporter.Start("in.mp4", "out.mp4", Layout{OutputWidth: 320, OutputHeight: 184, FPS: 24})
	reporter.Frame(1)
	reporter.Frame(2)
	reporter.Frame(10)
	reporter.Frame(11)
	reporter.Frame(20)
	reporter.AllFramesWritten(100)

	got := out.String()
	if strings.Count(got, "exporting video:") != 4 {
		t.Fatalf("progress output = %q, want start plus 10%%, 20%%, and 100%% milestones", got)
	}
	for _, want := range []string{
		"exporting video: 0/100 frames 0%\n",
		"exporting video: 10/100 frames 10%\n",
		"exporting video: 20/100 frames 20%\n",
		"exporting video: 100/100 frames 100%\n",
	} {
		if !strings.Contains(got, want) {
			t.Fatalf("progress output missing %q in %q", want, got)
		}
	}
}

func TestProgressReporterInteractiveThrottlesFrameUpdates(t *testing.T) {
	var out bytes.Buffer
	clock := &fakeClock{now: time.Unix(0, 0)}
	reporter := newProgressReporter(&out, progressReporterOptions{
		Interactive: true,
		TotalFrames: 100,
		Now:         clock.Now,
	})

	reporter.Start("in.mp4", "out.mp4", Layout{OutputWidth: 320, OutputHeight: 184, FPS: 24})
	reporter.Frame(1)
	reporter.Frame(2)

	clock.now = clock.now.Add(100 * time.Millisecond)
	reporter.Frame(3)

	got := out.String()
	if strings.Contains(got, "1/100") || strings.Contains(got, "2/100") {
		t.Fatalf("throttled frame update was printed: %q", got)
	}
	if !strings.Contains(got, "3/100 frames 3%") {
		t.Fatalf("progress output missing throttled update after interval: %q", got)
	}
}

func TestProgressReporterClampsPercentUntilAllFramesWritten(t *testing.T) {
	var out bytes.Buffer
	reporter := newProgressReporter(&out, progressReporterOptions{
		Interactive: false,
		TotalFrames: 10,
		Now:         time.Now,
	})

	reporter.Start("in.mp4", "out.mp4", Layout{OutputWidth: 320, OutputHeight: 184, FPS: 24})
	reporter.Frame(10)

	got := out.String()
	if strings.Contains(got, "100%") {
		t.Fatalf("frame update printed 100%% before EOF: %q", got)
	}

	reporter.AllFramesWritten(10)
	got = out.String()
	if !strings.Contains(got, "10/10 frames 100%") {
		t.Fatalf("all-frames-written did not print 100%%: %q", got)
	}
}

func TestProgressReporterCompleteWithEstimateMismatchDoesNotPrintPercent(t *testing.T) {
	for _, renderedFrames := range []int{0, 8, 12} {
		var out bytes.Buffer
		reporter := newProgressReporter(&out, progressReporterOptions{
			Interactive: false,
			TotalFrames: 10,
			Now:         time.Now,
		})

		reporter.Start("in.mp4", "out.mp4", Layout{OutputWidth: 320, OutputHeight: 184, FPS: 24})
		reporter.AllFramesWritten(renderedFrames)

		got := out.String()
		want := "exporting video: " + strconv.Itoa(renderedFrames) + " frames complete\n"
		if !strings.Contains(got, want) {
			t.Fatalf("progress output missing %q in %q", want, got)
		}
		if strings.Contains(got, "100%") {
			t.Fatalf("mismatched estimate printed 100%%: %q", got)
		}
	}
}

func TestProgressReporterErrorLineCleansInteractiveStatus(t *testing.T) {
	var out bytes.Buffer
	reporter := newProgressReporter(&out, progressReporterOptions{
		Interactive: true,
		TotalFrames: 10,
		Now:         time.Now,
	})

	reporter.Start("in.mp4", "out.mp4", Layout{OutputWidth: 320, OutputHeight: 184, FPS: 24})
	reporter.Frame(1)
	reporter.ErrorLine()

	got := out.String()
	if !strings.HasSuffix(got, "\n") {
		t.Fatalf("error cleanup did not leave a clean line: %q", got)
	}
}

func TestProgressReporterLineSafeWriterCleansStatusBeforeChildOutput(t *testing.T) {
	var out bytes.Buffer
	reporter := newProgressReporter(&out, progressReporterOptions{
		Interactive: true,
		TotalFrames: 10,
		Now:         time.Now,
	})

	reporter.Start("in.mp4", "out.mp4", Layout{OutputWidth: 320, OutputHeight: 184, FPS: 24})
	writer := reporter.lineSafeWriter(&out)
	if _, err := writer.Write([]byte("ffmpeg error\n")); err != nil {
		t.Fatalf("lineSafeWriter.Write returned error: %v", err)
	}

	got := out.String()
	if !strings.Contains(got, "0/10 frames 0%\nffmpeg error\n") {
		t.Fatalf("child output did not start on a clean line: %q", got)
	}
}

type fakeClock struct {
	now time.Time
}

func (c *fakeClock) Now() time.Time {
	return c.now
}
