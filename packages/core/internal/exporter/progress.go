package exporter

import (
	"fmt"
	"io"
	"math"
	"sync"
	"time"
)

const (
	progressUpdateInterval = 100 * time.Millisecond
	progressLogStepPercent = 10
	progressLogStepFrames  = 100
	clearProgressLine      = "\x1b[2K\r"
)

type InputProgressInfo struct {
	SourceFPS       float64
	FrameCount      int
	DurationSeconds float64
}

type progressReporterOptions struct {
	Interactive bool
	TotalFrames int
	Now         func() time.Time
}

type progressReporter struct {
	mu              sync.Mutex
	out             io.Writer
	interactive     bool
	totalFrames     int
	now             func() time.Time
	lastUpdate      time.Time
	nextLogPercent  int
	nextLogFrame    int
	statusLineOpen  bool
	lastStatusValue string
}

type progressLineSafeWriter struct {
	progress *progressReporter
	out      io.Writer
}

func estimateExportFrameTotal(info InputProgressInfo, layout Layout, options Options) int {
	if info.DurationSeconds > 0 && layout.FPS > 0 && (options.FPS > 0 || info.SourceFPS > 0) {
		total := int(math.Round(info.DurationSeconds * layout.FPS))
		if total > 0 {
			return total
		}
	}
	if options.FPS <= 0 && info.FrameCount > 0 {
		return info.FrameCount
	}
	return 0
}

func newProgressReporter(out io.Writer, options progressReporterOptions) *progressReporter {
	now := options.Now
	if now == nil {
		now = time.Now
	}
	return &progressReporter{
		out:            out,
		interactive:    options.Interactive,
		totalFrames:    max(options.TotalFrames, 0),
		now:            now,
		nextLogPercent: progressLogStepPercent,
		nextLogFrame:   progressLogStepFrames,
	}
}

func (p *progressReporter) Start(inputPath string, outputPath string, layout Layout) {
	if p == nil || p.out == nil {
		return
	}
	p.mu.Lock()
	defer p.mu.Unlock()
	fmt.Fprintf(p.out, "export: %s -> %s\n", inputPath, outputPath)
	fmt.Fprintf(p.out, "output: %dx%d @ %.3f fps\n", layout.OutputWidth, layout.OutputHeight, layout.FPS)
	p.writeStatusLocked(p.formatFrameStatus(0, false), true)
}

func (p *progressReporter) Frame(renderedFrames int) {
	if p == nil || p.out == nil {
		return
	}
	if renderedFrames < 0 {
		renderedFrames = 0
	}
	p.mu.Lock()
	defer p.mu.Unlock()
	if p.interactive {
		if !p.lastUpdate.IsZero() && p.now().Sub(p.lastUpdate) < progressUpdateInterval {
			return
		}
		p.writeStatusLocked(p.formatFrameStatus(renderedFrames, false), true)
		return
	}
	if p.totalFrames <= 0 {
		if renderedFrames >= p.nextLogFrame {
			p.writeStatusLocked(p.formatFrameStatus(renderedFrames, false), false)
			for p.nextLogFrame <= renderedFrames {
				p.nextLogFrame += progressLogStepFrames
			}
		}
		return
	}
	percent := p.clampedPercent(renderedFrames, false)
	if percent >= p.nextLogPercent {
		p.writeStatusLocked(p.formatFrameStatus(renderedFrames, false), false)
		for p.nextLogPercent <= percent {
			p.nextLogPercent += progressLogStepPercent
		}
	}
}

func (p *progressReporter) AllFramesWritten(renderedFrames int) {
	if p == nil || p.out == nil {
		return
	}
	p.mu.Lock()
	defer p.mu.Unlock()
	p.writeStatusLocked(p.formatFrameStatus(renderedFrames, true), true)
}

func (p *progressReporter) Finalizing() {
	if p == nil || p.out == nil {
		return
	}
	p.mu.Lock()
	defer p.mu.Unlock()
	p.writePhaseLocked("finalizing mp4...")
}

func (p *progressReporter) Complete(outputPath string) {
	if p == nil || p.out == nil {
		return
	}
	p.mu.Lock()
	defer p.mu.Unlock()
	p.writePhaseLocked(fmt.Sprintf("export complete: %s", outputPath))
}

func (p *progressReporter) ErrorLine() {
	if p == nil || p.out == nil {
		return
	}
	p.mu.Lock()
	defer p.mu.Unlock()
	p.errorLineLocked()
}

func (p *progressReporter) lineSafeWriter(out io.Writer) io.Writer {
	if p == nil || out == nil {
		return out
	}
	return progressLineSafeWriter{progress: p, out: out}
}

func (w progressLineSafeWriter) Write(data []byte) (int, error) {
	if w.progress == nil {
		return w.out.Write(data)
	}
	w.progress.mu.Lock()
	defer w.progress.mu.Unlock()
	w.progress.errorLineLocked()
	return w.out.Write(data)
}

func (p *progressReporter) errorLineLocked() {
	if p == nil || p.out == nil || !p.statusLineOpen {
		return
	}
	fmt.Fprint(p.out, "\n")
	p.statusLineOpen = false
}

func (p *progressReporter) formatFrameStatus(renderedFrames int, complete bool) string {
	if p.totalFrames <= 0 {
		return fmt.Sprintf("exporting video: %d frames", renderedFrames)
	}
	if complete && renderedFrames != p.totalFrames {
		return fmt.Sprintf("exporting video: %d frames complete", renderedFrames)
	}
	percent := p.clampedPercent(renderedFrames, complete)
	displayFrames := renderedFrames
	if complete {
		displayFrames = p.totalFrames
	}
	return fmt.Sprintf("exporting video: %d/%d frames %d%%", displayFrames, p.totalFrames, percent)
}

func (p *progressReporter) clampedPercent(renderedFrames int, complete bool) int {
	if p.totalFrames <= 0 {
		return 0
	}
	if complete {
		return 100
	}
	percent := renderedFrames * 100 / p.totalFrames
	if percent >= 100 {
		return 99
	}
	if percent < 0 {
		return 0
	}
	return percent
}

func (p *progressReporter) writeStatusLocked(status string, force bool) {
	if !force && status == p.lastStatusValue {
		return
	}
	p.lastStatusValue = status
	p.lastUpdate = p.now()
	if p.interactive {
		fmt.Fprintf(p.out, "%s%s", clearProgressLine, status)
		p.statusLineOpen = true
		return
	}
	fmt.Fprintf(p.out, "%s\n", status)
}

func (p *progressReporter) writePhaseLocked(status string) {
	if p.interactive && p.statusLineOpen {
		fmt.Fprint(p.out, "\n")
		p.statusLineOpen = false
	}
	fmt.Fprintf(p.out, "%s\n", status)
}
