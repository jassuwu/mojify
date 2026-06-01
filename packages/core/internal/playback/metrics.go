package playback

import (
	"fmt"
	"sync"
	"time"
)

type Metrics struct {
	mu sync.Mutex

	gridCols int
	gridRows int

	startedAt  time.Time
	finishedAt time.Time

	renderedFrames  int
	presentedFrames int
	skippedFrames   int
	renderTime      time.Duration
	presentTime     time.Duration
	outputBytes     int
}

type Snapshot struct {
	GridCols int
	GridRows int

	RenderedFrames       int
	PresentedFrames      int
	SkippedFrames        int
	EffectiveFPS         float64
	AverageRenderTime    time.Duration
	AveragePresentTime   time.Duration
	AverageBytesPerFrame int
	Elapsed              time.Duration
}

func NewMetrics(gridCols int, gridRows int) *Metrics {
	return &Metrics{gridCols: gridCols, gridRows: gridRows}
}

func (m *Metrics) Start(now time.Time) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.startedAt = now
	m.finishedAt = time.Time{}
}

func (m *Metrics) Finish(now time.Time) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.finishedAt = now
}

func (m *Metrics) RecordRendered(duration time.Duration) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.renderedFrames++
	m.renderTime += duration
}

func (m *Metrics) RecordPresented(bytes int, duration time.Duration) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.presentedFrames++
	m.outputBytes += bytes
	m.presentTime += duration
}

func (m *Metrics) RecordSkipped(count int) {
	if count <= 0 {
		return
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	m.skippedFrames += count
}

func (m *Metrics) Snapshot() Snapshot {
	m.mu.Lock()
	defer m.mu.Unlock()

	elapsed := time.Duration(0)
	if !m.startedAt.IsZero() {
		end := m.finishedAt
		if end.IsZero() {
			end = time.Now()
		}
		elapsed = end.Sub(m.startedAt)
	}

	effectiveFPS := 0.0
	if elapsed > 0 {
		effectiveFPS = float64(m.presentedFrames) / elapsed.Seconds()
	}

	averageRenderTime := time.Duration(0)
	if m.renderedFrames > 0 {
		averageRenderTime = m.renderTime / time.Duration(m.renderedFrames)
	}

	averagePresentTime := time.Duration(0)
	averageBytesPerFrame := 0
	if m.presentedFrames > 0 {
		averagePresentTime = m.presentTime / time.Duration(m.presentedFrames)
		averageBytesPerFrame = m.outputBytes / m.presentedFrames
	}

	return Snapshot{
		GridCols:             m.gridCols,
		GridRows:             m.gridRows,
		RenderedFrames:       m.renderedFrames,
		PresentedFrames:      m.presentedFrames,
		SkippedFrames:        m.skippedFrames,
		EffectiveFPS:         effectiveFPS,
		AverageRenderTime:    averageRenderTime,
		AveragePresentTime:   averagePresentTime,
		AverageBytesPerFrame: averageBytesPerFrame,
		Elapsed:              elapsed,
	}
}

func (m *Metrics) Summary() string {
	snapshot := m.Snapshot()
	return fmt.Sprintf(
		"playback stats\nrender grid: %dx%d\nrendered frames: %d\npresented frames: %d\nskipped frames: %d\neffective fps: %.2f\navg render time: %s\navg present time: %s\navg bytes/frame: %d\n",
		snapshot.GridCols,
		snapshot.GridRows,
		snapshot.RenderedFrames,
		snapshot.PresentedFrames,
		snapshot.SkippedFrames,
		snapshot.EffectiveFPS,
		snapshot.AverageRenderTime,
		snapshot.AveragePresentTime,
		snapshot.AverageBytesPerFrame,
	)
}
