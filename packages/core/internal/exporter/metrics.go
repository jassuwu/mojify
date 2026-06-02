package exporter

import (
	"fmt"
	"sync"
	"time"
)

type exportClock interface {
	Now() time.Time
}

type realExportClock struct{}

func (realExportClock) Now() time.Time {
	return time.Now()
}

type exportClockFunc func() time.Time

func (f exportClockFunc) Now() time.Time {
	return f()
}

type exportMetrics struct {
	mu sync.Mutex

	workers int
	clock   exportClock

	startedAt  time.Time
	finishedAt time.Time

	readFrames       int
	renderedFrames   int
	rasterizedFrames int
	writtenFrames    int

	readTime      time.Duration
	renderTime    time.Duration
	rasterizeTime time.Duration
	writeTime     time.Duration
}

func newExportMetrics(workers int, clock exportClock) *exportMetrics {
	if workers < 1 {
		workers = 1
	}
	if clock == nil {
		clock = realExportClock{}
	}
	return &exportMetrics{workers: workers, clock: clock}
}

func (m *exportMetrics) Start() {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.startedAt = m.clock.Now()
	m.finishedAt = time.Time{}
}

func (m *exportMetrics) Finish() {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.finishedAt = m.clock.Now()
}

func (m *exportMetrics) RecordRead(duration time.Duration) {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.readFrames++
	m.readTime += duration
}

func (m *exportMetrics) RecordRender(duration time.Duration) {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.renderedFrames++
	m.renderTime += duration
}

func (m *exportMetrics) RecordRasterize(duration time.Duration) {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.rasterizedFrames++
	m.rasterizeTime += duration
}

func (m *exportMetrics) RecordWrite(duration time.Duration) {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.writtenFrames++
	m.writeTime += duration
}

func (m *exportMetrics) Summary() string {
	m.mu.Lock()
	defer m.mu.Unlock()

	elapsed := time.Duration(0)
	if !m.startedAt.IsZero() && !m.finishedAt.IsZero() {
		elapsed = m.finishedAt.Sub(m.startedAt)
	}

	return fmt.Sprintf(
		"export stats\nworkers: %d\nread frames: %d\nrendered frames: %d\nrasterized frames: %d\nwritten frames: %d\nelapsed: %s\navg read time: %s\navg render time: %s\navg rasterize time: %s\navg encoder write time: %s\n",
		m.workers,
		m.readFrames,
		m.renderedFrames,
		m.rasterizedFrames,
		m.writtenFrames,
		elapsed,
		averageDuration(m.readTime, m.readFrames),
		averageDuration(m.renderTime, m.renderedFrames),
		averageDuration(m.rasterizeTime, m.rasterizedFrames),
		averageDuration(m.writeTime, m.writtenFrames),
	)
}

func averageDuration(total time.Duration, count int) time.Duration {
	if count <= 0 {
		return 0
	}
	return total / time.Duration(count)
}
