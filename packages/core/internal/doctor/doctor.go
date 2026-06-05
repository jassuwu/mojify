package doctor

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"os/exec"
	"strings"
	"time"
)

const DefaultTimeout = 2 * time.Second

type Status string

const (
	StatusOK    Status = "ok"
	StatusWarn  Status = "warn"
	StatusError Status = "error"
)

type Check struct {
	Name     string
	Args     []string
	Required bool
}

type Result struct {
	Name     string
	Status   Status
	Version  string
	Detail   string
	Required bool
}

type Report struct {
	Results     []Result
	Interrupted bool
}

type Runner func(ctx context.Context, name string, args ...string) ([]byte, []byte, error)

type Options struct {
	Runner  Runner
	Timeout time.Duration
	Checks  []Check
}

func DefaultChecks() []Check {
	return []Check{
		{Name: "ffmpeg", Args: []string{"-version"}, Required: true},
		{Name: "ffprobe", Args: []string{"-version"}, Required: true},
		{Name: "ffplay", Args: []string{"-version"}, Required: false},
		{Name: "yt-dlp", Args: []string{"--version"}, Required: false},
	}
}

func Run(ctx context.Context, options Options) Report {
	runner := options.Runner
	if runner == nil {
		runner = execRunner
	}
	timeout := options.Timeout
	if timeout <= 0 {
		timeout = DefaultTimeout
	}
	checks := options.Checks
	if len(checks) == 0 {
		checks = DefaultChecks()
	}

	results := make([]Result, 0, len(checks))
	for _, check := range checks {
		if ctx.Err() != nil {
			return Report{Results: results, Interrupted: true}
		}
		result, interrupted := runCheck(ctx, check, runner, timeout)
		if interrupted {
			return Report{Results: results, Interrupted: true}
		}
		results = append(results, result)
	}
	return Report{Results: results}
}

func (report Report) OK() bool {
	if report.Interrupted {
		return false
	}
	for _, result := range report.Results {
		if result.Required && result.Status == StatusError {
			return false
		}
	}
	return true
}

func (report Report) Summary() string {
	if report.Interrupted {
		return "Mojify doctor was interrupted before all checks completed."
	}
	if !report.OK() {
		return "Mojify cannot play or export local media until required tools are installed."
	}

	ffplayWarn := hasNonOK(report, "ffplay")
	ytdlpWarn := hasNonOK(report, "yt-dlp")
	switch {
	case ffplayWarn && ytdlpWarn:
		return "Mojify can play and export local media. Live audio needs ffplay; platform URL input needs yt-dlp."
	case ffplayWarn:
		return "Mojify can play and export local media. Live audio needs ffplay."
	case ytdlpWarn:
		return "Mojify can play and export local media. Platform URL input needs yt-dlp."
	default:
		return "Mojify can play and export local media, play live audio, and resolve platform URLs."
	}
}

func Write(w io.Writer, report Report) {
	if w == nil {
		return
	}
	fmt.Fprintln(w, "mojify doctor")
	fmt.Fprintln(w)
	for _, result := range report.Results {
		value := result.Version
		if value == "" {
			value = result.Detail
		}
		fmt.Fprintf(w, "%-5s %-8s %s\n", result.Status, result.Name, value)
	}
	fmt.Fprintln(w)
	fmt.Fprintln(w, report.Summary())
}

func runCheck(ctx context.Context, check Check, runner Runner, timeout time.Duration) (Result, bool) {
	checkCtx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	stdout, stderr, err := runner(checkCtx, check.Name, check.Args...)
	if err != nil {
		if isInterrupted(checkCtx, err) {
			return Result{}, true
		}
		return failedResult(check, detailForFailure(checkCtx, check.Name, stderr, err)), false
	}

	output := string(stdout)
	if strings.TrimSpace(output) == "" {
		output = string(stderr)
	}
	version := parseVersion(check.Name, output)
	if version == "" {
		version = "available"
	}
	return Result{Name: check.Name, Status: StatusOK, Version: version, Required: check.Required}, false
}

func failedResult(check Check, detail string) Result {
	status := StatusWarn
	if check.Required {
		status = StatusError
	}
	return Result{Name: check.Name, Status: status, Detail: detail, Required: check.Required}
}

func detailForFailure(ctx context.Context, name string, stderr []byte, err error) string {
	if errors.Is(ctx.Err(), context.DeadlineExceeded) || errors.Is(err, context.DeadlineExceeded) {
		return "timed out while checking version"
	}
	if isMissingExecutable(err) {
		return fmt.Sprintf("missing; install %s and try again", name)
	}
	if line := firstLine(string(stderr)); line != "" {
		return fmt.Sprintf("failed: %s", line)
	}
	return fmt.Sprintf("failed: %v", err)
}

func isInterrupted(ctx context.Context, err error) bool {
	return errors.Is(ctx.Err(), context.Canceled) || errors.Is(err, context.Canceled)
}

func isMissingExecutable(err error) bool {
	if errors.Is(err, exec.ErrNotFound) {
		return true
	}
	var execErr *exec.Error
	return errors.As(err, &execErr) && errors.Is(execErr.Err, exec.ErrNotFound)
}

func parseVersion(name string, output string) string {
	for _, line := range strings.Split(output, "\n") {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		if name == "yt-dlp" {
			fields := strings.Fields(line)
			if len(fields) > 0 {
				return fields[0]
			}
			return line
		}
		fields := strings.Fields(line)
		for i, field := range fields {
			if field == "version" && i+1 < len(fields) {
				return fields[i+1]
			}
		}
		return line
	}
	return ""
}

func firstLine(output string) string {
	for _, line := range strings.Split(output, "\n") {
		line = strings.TrimSpace(line)
		if line != "" {
			return line
		}
	}
	return ""
}

func hasNonOK(report Report, name string) bool {
	for _, result := range report.Results {
		if result.Name == name {
			return result.Status != StatusOK
		}
	}
	return false
}

func execRunner(ctx context.Context, name string, args ...string) ([]byte, []byte, error) {
	cmd := exec.CommandContext(ctx, name, args...)
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	err := cmd.Run()
	return stdout.Bytes(), stderr.Bytes(), err
}
