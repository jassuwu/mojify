package doctor

import (
	"bytes"
	"context"
	"errors"
	"os/exec"
	"strings"
	"testing"
	"time"
)

type fakeResponse struct {
	stdout         string
	stderr         string
	err            error
	waitForContext bool
}

type fakeRunner map[string]fakeResponse

func (runner fakeRunner) Run(ctx context.Context, name string, args ...string) ([]byte, []byte, error) {
	response, ok := runner[name]
	if !ok {
		return nil, nil, &exec.Error{Name: name, Err: exec.ErrNotFound}
	}
	if response.waitForContext {
		<-ctx.Done()
		return nil, nil, ctx.Err()
	}
	return []byte(response.stdout), []byte(response.stderr), response.err
}

func TestRunReportsVersionsAndOptionalWarnings(t *testing.T) {
	report := Run(context.Background(), Options{
		Runner: fakeRunner{
			"ffmpeg":  {stdout: "ffmpeg version 8.0.1 Copyright\n"},
			"ffprobe": {stdout: "ffprobe version 8.0.1 Copyright\n"},
			"yt-dlp":  {stdout: "2026.05.22\n"},
		}.Run,
		Timeout: time.Second,
	})

	if !report.OK() {
		t.Fatalf("report.OK() = false, want true: %#v", report.Results)
	}
	assertResult(t, report, "ffmpeg", StatusOK, "8.0.1", "")
	assertResult(t, report, "ffprobe", StatusOK, "8.0.1", "")
	assertResult(t, report, "ffplay", StatusWarn, "", "missing; install ffplay and try again")
	assertResult(t, report, "yt-dlp", StatusOK, "2026.05.22", "")

	wantSummary := "Mojify can play and export local media. Live audio needs ffplay."
	if got := report.Summary(); got != wantSummary {
		t.Fatalf("Summary() = %q, want %q", got, wantSummary)
	}
}

func TestRunFailsWhenRequiredToolIsMissing(t *testing.T) {
	report := Run(context.Background(), Options{
		Runner: fakeRunner{
			"ffmpeg": {stdout: "ffmpeg version 8.0.1 Copyright\n"},
			"ffplay": {stdout: "ffplay version 8.0.1 Copyright\n"},
			"yt-dlp": {stdout: "2026.05.22\n"},
		}.Run,
		Timeout: time.Second,
	})

	if report.OK() {
		t.Fatalf("report.OK() = true, want false: %#v", report.Results)
	}
	assertResult(t, report, "ffprobe", StatusError, "", "missing; install ffprobe and try again")

	wantSummary := "Mojify cannot play or export local media until required tools are installed."
	if got := report.Summary(); got != wantSummary {
		t.Fatalf("Summary() = %q, want %q", got, wantSummary)
	}
}

func TestRunClassifiesFailuresBySeverity(t *testing.T) {
	report := Run(context.Background(), Options{
		Runner: fakeRunner{
			"ffmpeg":  {stderr: "broken ffmpeg\nextra detail", err: errors.New("exit status 1")},
			"ffprobe": {stdout: "ffprobe version 8.0.1 Copyright\n"},
			"ffplay":  {stderr: "audio unavailable\n", err: errors.New("exit status 1")},
			"yt-dlp":  {stdout: "2026.05.22\n"},
		}.Run,
		Timeout: time.Second,
	})

	if report.OK() {
		t.Fatalf("report.OK() = true, want false: %#v", report.Results)
	}
	assertResult(t, report, "ffmpeg", StatusError, "", "failed: broken ffmpeg")
	assertResult(t, report, "ffplay", StatusWarn, "", "failed: audio unavailable")
}

func TestRunDoesNotTreatEveryExecErrorAsMissing(t *testing.T) {
	report := Run(context.Background(), Options{
		Runner: fakeRunner{
			"ffmpeg":  {err: &exec.Error{Name: "ffmpeg", Err: exec.ErrDot}},
			"ffprobe": {stdout: "ffprobe version 8.0.1 Copyright\n"},
			"ffplay":  {stdout: "ffplay version 8.0.1 Copyright\n"},
			"yt-dlp":  {stdout: "2026.05.22\n"},
		}.Run,
		Timeout: time.Second,
	})

	if report.OK() {
		t.Fatalf("report.OK() = true, want false: %#v", report.Results)
	}
	assertResult(t, report, "ffmpeg", StatusError, "", "failed: exec: \"ffmpeg\": cannot run executable found relative to current directory")
}

func TestRunReportsTimeouts(t *testing.T) {
	report := Run(context.Background(), Options{
		Runner: fakeRunner{
			"ffmpeg":  {waitForContext: true},
			"ffprobe": {stdout: "ffprobe version 8.0.1 Copyright\n"},
			"ffplay":  {stdout: "ffplay version 8.0.1 Copyright\n"},
			"yt-dlp":  {stdout: "2026.05.22\n"},
		}.Run,
		Timeout: 5 * time.Millisecond,
	})

	if report.OK() {
		t.Fatalf("report.OK() = true, want false: %#v", report.Results)
	}
	assertResult(t, report, "ffmpeg", StatusError, "", "timed out while checking version")
}

func TestRunStopsOnCancellationWithoutHealthFailure(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	calls := 0
	report := Run(ctx, Options{
		Runner: func(ctx context.Context, name string, args ...string) ([]byte, []byte, error) {
			calls++
			cancel()
			<-ctx.Done()
			return nil, nil, ctx.Err()
		},
		Timeout: time.Second,
	})

	if !report.Interrupted {
		t.Fatalf("Interrupted = false, want true: %#v", report)
	}
	if len(report.Results) != 0 {
		t.Fatalf("Results = %#v, want no health results for interrupted check", report.Results)
	}
	if calls != 1 {
		t.Fatalf("runner calls = %d, want 1", calls)
	}
	wantSummary := "Mojify doctor was interrupted before all checks completed."
	if got := report.Summary(); got != wantSummary {
		t.Fatalf("Summary() = %q, want %q", got, wantSummary)
	}
}

func TestWriteReport(t *testing.T) {
	report := Report{Results: []Result{
		{Name: "ffmpeg", Status: StatusOK, Version: "8.0.1", Required: true},
		{Name: "ffprobe", Status: StatusOK, Version: "8.0.1", Required: true},
		{Name: "ffplay", Status: StatusWarn, Detail: "missing; install ffplay and try again"},
		{Name: "yt-dlp", Status: StatusWarn, Detail: "missing; install yt-dlp and try again"},
	}}

	var out bytes.Buffer
	Write(&out, report)
	got := out.String()
	for _, want := range []string{
		"mojify doctor\n",
		"ok    ffmpeg",
		"ok    ffprobe",
		"warn  ffplay",
		"warn  yt-dlp",
		"Mojify can play and export local media. Live audio needs ffplay; platform URL input needs yt-dlp.",
	} {
		if !strings.Contains(got, want) {
			t.Fatalf("report output missing %q in:\n%s", want, got)
		}
	}
}

func assertResult(t *testing.T, report Report, name string, status Status, version string, detail string) {
	t.Helper()
	for _, result := range report.Results {
		if result.Name != name {
			continue
		}
		if result.Status != status || result.Version != version || result.Detail != detail {
			t.Fatalf("%s result = %#v, want status=%q version=%q detail=%q", name, result, status, version, detail)
		}
		return
	}
	t.Fatalf("missing result for %s in %#v", name, report.Results)
}
