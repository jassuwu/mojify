package playback

import (
	"strings"
	"testing"
	"time"
)

func TestMetricsSnapshotCalculatesPlaybackValues(t *testing.T) {
	metrics := NewMetrics(120, 40)
	start := time.Unix(10, 0)
	metrics.Start(start)
	metrics.RecordRendered(10 * time.Millisecond)
	metrics.RecordRendered(30 * time.Millisecond)
	metrics.RecordPresented(1000, 5*time.Millisecond)
	metrics.RecordPresented(1400, 15*time.Millisecond)
	metrics.RecordSkipped(3)
	metrics.Finish(start.Add(2 * time.Second))

	snapshot := metrics.Snapshot()
	if snapshot.GridCols != 120 || snapshot.GridRows != 40 {
		t.Fatalf("grid = %dx%d, want 120x40", snapshot.GridCols, snapshot.GridRows)
	}
	if snapshot.RenderedFrames != 2 {
		t.Fatalf("RenderedFrames = %d, want 2", snapshot.RenderedFrames)
	}
	if snapshot.PresentedFrames != 2 {
		t.Fatalf("PresentedFrames = %d, want 2", snapshot.PresentedFrames)
	}
	if snapshot.SkippedFrames != 3 {
		t.Fatalf("SkippedFrames = %d, want 3", snapshot.SkippedFrames)
	}
	if snapshot.EffectiveFPS != 1 {
		t.Fatalf("EffectiveFPS = %f, want 1", snapshot.EffectiveFPS)
	}
	if snapshot.AverageRenderTime != 20*time.Millisecond {
		t.Fatalf("AverageRenderTime = %s, want 20ms", snapshot.AverageRenderTime)
	}
	if snapshot.AveragePresentTime != 10*time.Millisecond {
		t.Fatalf("AveragePresentTime = %s, want 10ms", snapshot.AveragePresentTime)
	}
	if snapshot.AverageBytesPerFrame != 1200 {
		t.Fatalf("AverageBytesPerFrame = %d, want 1200", snapshot.AverageBytesPerFrame)
	}
}

func TestMetricsSummaryContainsHumanReadableFields(t *testing.T) {
	metrics := NewMetrics(80, 24)
	start := time.Unix(10, 0)
	metrics.Start(start)
	metrics.RecordRendered(2 * time.Millisecond)
	metrics.RecordPresented(512, 3*time.Millisecond)
	metrics.RecordSkipped(1)
	metrics.Finish(start.Add(time.Second))

	summary := metrics.Summary()
	for _, want := range []string{
		"playback stats",
		"render grid: 80x24",
		"rendered frames: 1",
		"presented frames: 1",
		"skipped frames: 1",
		"effective fps: 1.00",
		"avg render time: 2ms",
		"avg present time: 3ms",
		"avg bytes/frame: 512",
	} {
		if !strings.Contains(summary, want) {
			t.Fatalf("Summary missing %q in:\n%s", want, summary)
		}
	}
}
