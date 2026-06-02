package exporter

import (
	"strings"
	"testing"
	"time"
)

func TestExportMetricsSummary(t *testing.T) {
	metrics := newExportMetrics(4, fakeExportClock(time.Unix(0, 0)))
	metrics.Start()
	metrics.RecordRead(2 * time.Millisecond)
	metrics.RecordRender(3 * time.Millisecond)
	metrics.RecordRasterize(5 * time.Millisecond)
	metrics.RecordWrite(7 * time.Millisecond)
	metrics.RecordWrite(9 * time.Millisecond)
	metrics.Finish()

	summary := metrics.Summary()
	for _, want := range []string{
		"export stats",
		"workers: 4",
		"read frames: 1",
		"rendered frames: 1",
		"rasterized frames: 1",
		"written frames: 2",
		"elapsed: 0s",
		"avg read time: 2ms",
		"avg render time: 3ms",
		"avg rasterize time: 5ms",
		"avg encoder write time: 8ms",
	} {
		if !strings.Contains(summary, want) {
			t.Fatalf("Summary missing %q in:\n%s", want, summary)
		}
	}
}

func TestExportMetricsSummaryHandlesNoFrames(t *testing.T) {
	metrics := newExportMetrics(1, fakeExportClock(time.Unix(0, 0)))
	metrics.Start()
	metrics.Finish()

	summary := metrics.Summary()
	for _, want := range []string{
		"workers: 1",
		"read frames: 0",
		"rendered frames: 0",
		"rasterized frames: 0",
		"written frames: 0",
		"elapsed: 0s",
		"avg read time: 0s",
		"avg render time: 0s",
		"avg rasterize time: 0s",
		"avg encoder write time: 0s",
	} {
		if !strings.Contains(summary, want) {
			t.Fatalf("Summary missing %q in:\n%s", want, summary)
		}
	}
}

type fakeExportClock time.Time

func (c fakeExportClock) Now() time.Time {
	return time.Time(c)
}
