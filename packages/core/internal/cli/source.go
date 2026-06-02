package cli

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

const (
	defaultYTDLPPath          = "yt-dlp"
	sourceFilenameTemplate    = "%(title).120B [%(id)s].%(ext)s"
	platformDownloadFormat    = "bv*[ext=mp4]+ba[ext=m4a]/b[ext=mp4]/b"
	platformMergeOutput       = "mp4"
	sourceTempDirectoryPrefix = "mojify-source-"
)

type resolvedSourceMedia struct {
	Original    string
	Path        string
	DisplayName string
	Temporary   bool
	Cleanup     func() error
}

type sourceResolverOptions struct {
	Stderr    io.Writer
	YTDLPPath string
}

type sourceMetadata struct {
	Type       string            `json:"_type"`
	IsLive     bool              `json:"is_live"`
	LiveStatus string            `json:"live_status"`
	Entries    []json.RawMessage `json:"entries"`
}

func resolveSourceMedia(ctx context.Context, source string, stderr io.Writer) (resolvedSourceMedia, error) {
	return resolveSourceMediaWithOptions(ctx, source, sourceResolverOptions{Stderr: stderr})
}

func resolveSourceMediaWithOptions(ctx context.Context, source string, options sourceResolverOptions) (resolvedSourceMedia, error) {
	if !isHTTPPlatformSource(source) {
		return resolvedSourceMedia{
			Original:    source,
			Path:        source,
			DisplayName: filepath.Base(source),
			Cleanup:     func() error { return nil },
		}, nil
	}

	ytdlpPath := options.YTDLPPath
	if ytdlpPath == "" {
		ytdlpPath = defaultYTDLPPath
	}
	status := options.Stderr
	if status != nil {
		fmt.Fprintf(status, "resolving source media: %s\n", source)
	}

	if err := preflightPlatformSource(ctx, ytdlpPath, source); err != nil {
		return resolvedSourceMedia{}, err
	}

	tempDir, err := os.MkdirTemp("", sourceTempDirectoryPrefix)
	if err != nil {
		return resolvedSourceMedia{}, fmt.Errorf("create source temp directory: %w", err)
	}
	cleanup := func() error {
		return os.RemoveAll(tempDir)
	}
	cleanupPending := true
	defer func() {
		if cleanupPending {
			_ = cleanup()
		}
	}()

	if status != nil {
		fmt.Fprintln(status, "downloading source media...")
	}
	finalPath, err := downloadPlatformSource(ctx, ytdlpPath, tempDir, source)
	if err != nil {
		return resolvedSourceMedia{}, err
	}
	finalPath = strings.TrimSpace(finalPath)
	if finalPath == "" {
		return resolvedSourceMedia{}, fmt.Errorf("resolve source media: yt-dlp did not report a downloaded source path")
	}
	displayName := filepath.Base(finalPath)
	if status != nil {
		fmt.Fprintf(status, "source media ready: %s\n", displayName)
	}

	cleanupPending = false
	return resolvedSourceMedia{
		Original:    source,
		Path:        finalPath,
		DisplayName: displayName,
		Temporary:   true,
		Cleanup:     cleanup,
	}, nil
}

func preflightPlatformSource(ctx context.Context, ytdlpPath string, source string) error {
	args := []string{
		"--ignore-config",
		"--no-playlist",
		"--no-progress",
		"--dump-single-json",
		"--skip-download",
		source,
	}
	output, stderr, err := runYTDLP(ctx, ytdlpPath, args)
	if err != nil {
		return formatYTDLPError("resolve source media", err, stderr)
	}
	var metadata sourceMetadata
	if err := json.Unmarshal(output, &metadata); err != nil {
		return fmt.Errorf("resolve source media: parse yt-dlp metadata: %w", err)
	}
	if metadata.Type == "playlist" || len(metadata.Entries) > 0 {
		return fmt.Errorf("resolve source media: playlists are not supported")
	}
	if isLiveSourceMetadata(metadata) {
		return fmt.Errorf("resolve source media: live streams are not supported")
	}
	return nil
}

func isLiveSourceMetadata(metadata sourceMetadata) bool {
	if metadata.IsLive {
		return true
	}
	switch strings.ToLower(metadata.LiveStatus) {
	case "is_live", "is_upcoming", "post_live":
		return true
	default:
		return false
	}
}

func downloadPlatformSource(ctx context.Context, ytdlpPath string, tempDir string, source string) (string, error) {
	args := platformDownloadArgs(tempDir, source)
	output, stderr, err := runYTDLP(ctx, ytdlpPath, args)
	if err != nil {
		return "", formatYTDLPError("download source media", err, stderr)
	}
	lines := strings.Split(strings.TrimSpace(string(output)), "\n")
	if len(lines) == 0 {
		return "", nil
	}
	return strings.TrimSpace(lines[len(lines)-1]), nil
}

func platformDownloadArgs(tempDir string, source string) []string {
	return []string{
		"--ignore-config",
		"--no-playlist",
		"--match-filters", "!is_live",
		"--no-progress",
		"--paths", "home:" + tempDir,
		"--paths", "temp:" + tempDir,
		"--restrict-filenames",
		"--trim-filenames", "120",
		"--output", sourceFilenameTemplate,
		"--print", "after_move:filepath",
		"--merge-output-format", platformMergeOutput,
		"-f", platformDownloadFormat,
		source,
	}
}

func runYTDLP(ctx context.Context, ytdlpPath string, args []string) ([]byte, string, error) {
	cmd := exec.CommandContext(ctx, ytdlpPath, args...)
	var stderr bytes.Buffer
	cmd.Stderr = &stderr
	output, err := cmd.Output()
	return output, strings.TrimSpace(stderr.String()), err
}

func formatYTDLPError(prefix string, err error, stderr string) error {
	var execErr *exec.Error
	if errors.As(err, &execErr) {
		return fmt.Errorf("%s: yt-dlp is required for platform URLs; install yt-dlp and try again", prefix)
	}
	if stderr != "" {
		return fmt.Errorf("%s: yt-dlp failed: %s", prefix, stderr)
	}
	return fmt.Errorf("%s: yt-dlp failed: %w", prefix, err)
}

func isHTTPPlatformSource(source string) bool {
	lower := strings.ToLower(source)
	return strings.HasPrefix(lower, "http://") || strings.HasPrefix(lower, "https://")
}
