package cli

import (
	"bytes"
	"context"
	"errors"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

func TestResolveSourceMediaLocalBypassesYTDLP(t *testing.T) {
	var stderr bytes.Buffer
	resolved, err := resolveSourceMediaWithOptions(context.Background(), "dist/clip.mp4", sourceResolverOptions{
		Stderr:    &stderr,
		YTDLPPath: "missing-yt-dlp-for-local-test",
	})
	if err != nil {
		t.Fatalf("resolveSourceMedia returned error: %v", err)
	}
	defer resolved.Cleanup()
	if resolved.Path != "dist/clip.mp4" {
		t.Fatalf("Path = %q, want local path", resolved.Path)
	}
	if resolved.Original != "dist/clip.mp4" {
		t.Fatalf("Original = %q, want local source", resolved.Original)
	}
	if resolved.Temporary {
		t.Fatal("Temporary = true, want false for local source")
	}
	if stderr.String() != "" {
		t.Fatalf("stderr = %q, want no local source status", stderr.String())
	}
}

func TestResolveSourceMediaClassifiesLocalStillSources(t *testing.T) {
	staticPNGPath := filepath.Join(t.TempDir(), "static.png")
	staticPNG := []byte{
		0x89, 'P', 'N', 'G', '\r', '\n', 0x1a, '\n',
		0, 0, 0, 0, 'I', 'E', 'N', 'D', 0, 0, 0, 0,
	}
	if err := os.WriteFile(staticPNGPath, staticPNG, 0o644); err != nil {
		t.Fatalf("write static PNG fixture: %v", err)
	}
	for _, source := range []string{
		staticPNGPath,
		"image.png",
		"image.jpg",
		"image.jpeg",
		"IMAGE.PNG",
		"IMAGE.JPG",
		"IMAGE.JPEG",
	} {
		resolved, err := resolveSourceMediaWithOptions(context.Background(), source, sourceResolverOptions{
			YTDLPPath: "missing-yt-dlp-for-local-test",
		})
		if err != nil {
			t.Fatalf("resolveSourceMediaWithOptions(%q) returned error: %v", source, err)
		}
		defer resolved.Cleanup()
		if resolved.Kind != sourceKindStill {
			t.Fatalf("Kind for %q = %s, want %s", source, resolved.Kind, sourceKindStill)
		}
	}
}

func TestResolveSourceMediaClassifiesLocalNonStillSourcesAsTimeBased(t *testing.T) {
	for _, source := range []string{
		"clip.mp4",
		"clip.mov",
		"clip",
	} {
		resolved, err := resolveSourceMediaWithOptions(context.Background(), source, sourceResolverOptions{
			YTDLPPath: "missing-yt-dlp-for-local-test",
		})
		if err != nil {
			t.Fatalf("resolveSourceMediaWithOptions(%q) returned error: %v", source, err)
		}
		defer resolved.Cleanup()
		if resolved.Kind != sourceKindTimeBased {
			t.Fatalf("Kind for %q = %s, want %s", source, resolved.Kind, sourceKindTimeBased)
		}
	}
}

func TestResolveSourceMediaRejectsDeferredImageSources(t *testing.T) {
	for _, tc := range []struct {
		source string
		want   string
	}{
		{"clip.gif", "animated image sources are not supported"},
		{"clip.apng", "animated image sources are not supported"},
		{"clip.webp", "webp source images are not supported"},
	} {
		t.Run(tc.source, func(t *testing.T) {
			_, err := resolveSourceMediaWithOptions(context.Background(), tc.source, sourceResolverOptions{
				YTDLPPath: "missing-yt-dlp-for-local-test",
			})
			if err == nil || !strings.Contains(err.Error(), tc.want) {
				t.Fatalf("error = %v, want %q", err, tc.want)
			}
		})
	}
}

func TestResolveSourceMediaRejectsAPNGContentWithPNGExtension(t *testing.T) {
	path := filepath.Join(t.TempDir(), "animated.png")
	data := []byte{
		0x89, 'P', 'N', 'G', '\r', '\n', 0x1a, '\n',
		0, 0, 0, 0, 'a', 'c', 'T', 'L', 0, 0, 0, 0,
	}
	if err := os.WriteFile(path, data, 0o644); err != nil {
		t.Fatalf("write APNG marker fixture: %v", err)
	}

	_, err := resolveSourceMediaWithOptions(context.Background(), path, sourceResolverOptions{
		YTDLPPath: "missing-yt-dlp-for-local-test",
	})
	if err == nil || !strings.Contains(err.Error(), "animated image sources are not supported") {
		t.Fatalf("error = %v, want animated image rejection", err)
	}
}

func TestResolveSourceMediaDownloadsHTTPSSource(t *testing.T) {
	fake := writeFakeYTDLP(t, fakeYTDLPOptions{})
	var stderr bytes.Buffer
	resolved, err := resolveSourceMediaWithOptions(context.Background(), "https://example.com/watch?v=demo", sourceResolverOptions{
		Stderr:    &stderr,
		YTDLPPath: fake.Path,
	})
	if err != nil {
		t.Fatalf("resolveSourceMedia returned error: %v", err)
	}
	downloadedPath := resolved.Path
	if !resolved.Temporary {
		t.Fatal("Temporary = false, want true for platform URL")
	}
	if resolved.Original != "https://example.com/watch?v=demo" {
		t.Fatalf("Original = %q, want original URL", resolved.Original)
	}
	if resolved.DisplayName != "Demo_Title [abc123].mp4" {
		t.Fatalf("DisplayName = %q, want final basename", resolved.DisplayName)
	}
	if resolved.Kind != sourceKindTimeBased {
		t.Fatalf("Kind = %s, want %s", resolved.Kind, sourceKindTimeBased)
	}
	if _, err := os.Stat(downloadedPath); err != nil {
		t.Fatalf("downloaded path does not exist before cleanup: %v", err)
	}
	if err := resolved.Cleanup(); err != nil {
		t.Fatalf("Cleanup returned error: %v", err)
	}
	if _, err := os.Stat(downloadedPath); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("downloaded path still exists after cleanup, stat err = %v", err)
	}
	got := stderr.String()
	for _, want := range []string{
		"resolving source media: https://example.com/watch?v=demo\n",
		"downloading source media...\n",
		"source media ready: Demo_Title [abc123].mp4\n",
	} {
		if !strings.Contains(got, want) {
			t.Fatalf("stderr missing %q in %q", want, got)
		}
	}
}

func TestResolveSourceMediaPassesExpectedYTDLPArgs(t *testing.T) {
	argsPath := filepath.Join(t.TempDir(), "args.txt")
	fake := writeFakeYTDLP(t, fakeYTDLPOptions{ArgsPath: argsPath})
	resolved, err := resolveSourceMediaWithOptions(context.Background(), "https://example.com/watch?v=demo", sourceResolverOptions{
		YTDLPPath: fake.Path,
	})
	if err != nil {
		t.Fatalf("resolveSourceMedia returned error: %v", err)
	}
	defer resolved.Cleanup()
	data, err := os.ReadFile(argsPath)
	if err != nil {
		t.Fatalf("read args: %v", err)
	}
	calls := splitFakeYTDLPCalls(string(data))
	if len(calls) != 2 {
		t.Fatalf("yt-dlp calls = %#v, want preflight and download calls", calls)
	}
	got := calls[1]
	if len(got) != 21 {
		t.Fatalf("download args = %#v, want 21 args", got)
	}
	want := []string{
		"--ignore-config",
		"--no-playlist",
		"--match-filters",
		"!is_live",
		"--no-progress",
		"--paths",
		got[6],
		"--paths",
		got[8],
		"--restrict-filenames",
		"--trim-filenames",
		"120",
		"--output",
		"%(title).120B [%(id)s].%(ext)s",
		"--print",
		"after_move:filepath",
		"--merge-output-format",
		"mp4",
		"-f",
		"bv*[ext=mp4]+ba[ext=m4a]/b[ext=mp4]/b",
		"https://example.com/watch?v=demo",
	}
	if !strings.HasPrefix(got[6], "home:") {
		t.Fatalf("home path arg = %q, want home:<tmp>", got[6])
	}
	if !strings.HasPrefix(got[8], "temp:") {
		t.Fatalf("temp path arg = %q, want temp:<tmp>", got[8])
	}
	if strings.TrimPrefix(got[6], "home:") != strings.TrimPrefix(got[8], "temp:") {
		t.Fatalf("home/temp paths differ: %q vs %q", got[6], got[8])
	}
	if strings.TrimPrefix(got[6], "home:") == "" {
		t.Fatal("home path arg is empty")
	}
	if strings.Join(got, "\n") != strings.Join(want, "\n") {
		t.Fatalf("download args mismatch:\ngot:\n%s\nwant:\n%s", strings.Join(got, "\n"), strings.Join(want, "\n"))
	}
}

func TestResolveSourceMediaPassesExpectedYTDLPPreflightArgs(t *testing.T) {
	argsPath := filepath.Join(t.TempDir(), "args.txt")
	fake := writeFakeYTDLP(t, fakeYTDLPOptions{ArgsPath: argsPath})
	resolved, err := resolveSourceMediaWithOptions(context.Background(), "https://example.com/watch?v=demo", sourceResolverOptions{
		YTDLPPath: fake.Path,
	})
	if err != nil {
		t.Fatalf("resolveSourceMedia returned error: %v", err)
	}
	defer resolved.Cleanup()
	data, err := os.ReadFile(argsPath)
	if err != nil {
		t.Fatalf("read args: %v", err)
	}
	calls := splitFakeYTDLPCalls(string(data))
	if len(calls) != 2 {
		t.Fatalf("yt-dlp calls = %#v, want preflight and download calls", calls)
	}
	got := calls[0]
	want := []string{
		"--ignore-config",
		"--no-playlist",
		"--no-progress",
		"--dump-single-json",
		"--skip-download",
		"https://example.com/watch?v=demo",
	}
	if strings.Join(got, "\n") != strings.Join(want, "\n") {
		t.Fatalf("preflight args mismatch:\ngot:\n%s\nwant:\n%s", strings.Join(got, "\n"), strings.Join(want, "\n"))
	}
}

func TestResolveSourceMediaRejectsPlaylistMetadata(t *testing.T) {
	fake := writeFakeYTDLP(t, fakeYTDLPOptions{MetadataJSON: `{"_type":"playlist","entries":[{"id":"one"}]}`})
	_, err := resolveSourceMediaWithOptions(context.Background(), "https://example.com/playlist", sourceResolverOptions{
		YTDLPPath: fake.Path,
	})
	if err == nil || !strings.Contains(err.Error(), "playlists are not supported") {
		t.Fatalf("error = %v, want playlist rejection", err)
	}
}

func TestResolveSourceMediaRejectsLiveMetadata(t *testing.T) {
	for _, metadata := range []string{
		`{"id":"live","is_live":true}`,
		`{"id":"live","is_live":false,"live_status":"is_live"}`,
		`{"id":"upcoming","is_live":false,"live_status":"is_upcoming"}`,
		`{"id":"post-live","is_live":false,"live_status":"post_live"}`,
	} {
		fake := writeFakeYTDLP(t, fakeYTDLPOptions{MetadataJSON: metadata})
		_, err := resolveSourceMediaWithOptions(context.Background(), "https://example.com/live", sourceResolverOptions{
			YTDLPPath: fake.Path,
		})
		if err == nil || !strings.Contains(err.Error(), "live streams are not supported") {
			t.Fatalf("metadata %s error = %v, want live stream rejection", metadata, err)
		}
	}
}

func TestResolveSourceMediaAcceptsFiniteLiveStatusMetadata(t *testing.T) {
	for _, metadata := range []string{
		`{"id":"abc123","title":"Demo Title","is_live":false,"live_status":"not_live"}`,
		`{"id":"abc123","title":"Demo Title","is_live":false,"live_status":"was_live"}`,
	} {
		fake := writeFakeYTDLP(t, fakeYTDLPOptions{MetadataJSON: metadata})
		resolved, err := resolveSourceMediaWithOptions(context.Background(), "https://example.com/watch?v=demo", sourceResolverOptions{
			YTDLPPath: fake.Path,
		})
		if err != nil {
			t.Fatalf("metadata %s resolveSourceMedia returned error: %v", metadata, err)
		}
		defer resolved.Cleanup()
		if resolved.Path == "" {
			t.Fatal("Path is empty, want downloaded source")
		}
	}
}

func TestResolveSourceMediaMissingYTDLPHasConciseError(t *testing.T) {
	_, err := resolveSourceMediaWithOptions(context.Background(), "https://example.com/watch?v=demo", sourceResolverOptions{
		YTDLPPath: "definitely-missing-yt-dlp",
	})
	if err == nil || !strings.Contains(err.Error(), "yt-dlp is required for platform URLs") {
		t.Fatalf("error = %v, want missing yt-dlp message", err)
	}
}

func TestResolveSourceMediaYTDLPFailureIncludesStderr(t *testing.T) {
	fake := writeFakeYTDLP(t, fakeYTDLPOptions{
		FailDownload: true,
		FailureText:  "unsupported url",
	})
	_, err := resolveSourceMediaWithOptions(context.Background(), "https://example.com/watch?v=demo", sourceResolverOptions{
		YTDLPPath: fake.Path,
	})
	if err == nil || !strings.Contains(err.Error(), "unsupported url") {
		t.Fatalf("error = %v, want yt-dlp stderr context", err)
	}
}

func TestResolveSourceMediaCleansTempDirOnYTDLPFailure(t *testing.T) {
	tempPath := filepath.Join(t.TempDir(), "temp.txt")
	fake := writeFakeYTDLP(t, fakeYTDLPOptions{
		FailDownload: true,
		FailureText:  "unsupported url",
		TempPath:     tempPath,
	})
	_, err := resolveSourceMediaWithOptions(context.Background(), "https://example.com/watch?v=demo", sourceResolverOptions{
		YTDLPPath: fake.Path,
	})
	if err == nil {
		t.Fatal("resolveSourceMedia returned nil error, want yt-dlp failure")
	}
	data, err := os.ReadFile(tempPath)
	if err != nil {
		t.Fatalf("read temp path: %v", err)
	}
	tempDir := strings.TrimSpace(string(data))
	if tempDir == "" {
		t.Fatal("fake yt-dlp did not record temp directory")
	}
	if _, err := os.Stat(tempDir); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("temp dir still exists after failure, stat err = %v", err)
	}
}

func TestResolveSourceMediaRejectsEmptyFinalPath(t *testing.T) {
	fake := writeFakeYTDLP(t, fakeYTDLPOptions{EmptyFinalPath: true})
	_, err := resolveSourceMediaWithOptions(context.Background(), "https://example.com/watch?v=demo", sourceResolverOptions{
		YTDLPPath: fake.Path,
	})
	if err == nil || !strings.Contains(err.Error(), "yt-dlp did not report a downloaded source path") {
		t.Fatalf("error = %v, want empty final path rejection", err)
	}
}

type fakeYTDLP struct {
	Path string
}

type fakeYTDLPOptions struct {
	ArgsPath       string
	TempPath       string
	MetadataJSON   string
	FailDownload   bool
	FailureText    string
	EmptyFinalPath bool
}

func writeFakeYTDLP(t *testing.T, options fakeYTDLPOptions) fakeYTDLP {
	t.Helper()
	dir := t.TempDir()
	path := filepath.Join(dir, "yt-dlp")
	if runtime.GOOS == "windows" {
		path += ".bat"
		t.Skip("fake yt-dlp bash script contract tests are Unix-only")
	}
	metadata := options.MetadataJSON
	if metadata == "" {
		metadata = `{"id":"abc123","title":"Demo Title","is_live":false}`
	}
	failure := options.FailureText
	if failure == "" {
		failure = "yt-dlp failed"
	}
	script := "#!/usr/bin/env bash\n" +
		"set -euo pipefail\n" +
		"args_path=" + shellQuote(options.ArgsPath) + "\n" +
		"temp_path=" + shellQuote(options.TempPath) + "\n" +
		"if [[ -n \"${args_path}\" ]]; then { printf 'CALL\\n'; for arg in \"$@\"; do printf '%s\\n' \"${arg}\"; done; } >> \"${args_path}\"; fi\n" +
		"for arg in \"$@\"; do\n" +
		"  if [[ \"${arg}\" == \"--dump-single-json\" ]]; then printf '%s\\n' " + shellQuote(metadata) + "; exit 0; fi\n" +
		"done\n" +
		"home=\"\"\n" +
		"prev=\"\"\n" +
		"for arg in \"$@\"; do\n" +
		"  if [[ \"${prev}\" == \"--paths\" && \"${arg}\" == home:* ]]; then home=\"${arg#home:}\"; fi\n" +
		"  prev=\"${arg}\"\n" +
		"done\n" +
		"if [[ -z \"${home}\" ]]; then echo 'missing home path' >&2; exit 7; fi\n" +
		"if [[ -n \"${temp_path}\" ]]; then printf '%s\\n' \"${home}\" > \"${temp_path}\"; fi\n" +
		"if [[ " + shellBool(options.FailDownload) + " == true ]]; then echo " + shellQuote(failure) + " >&2; exit 9; fi\n" +
		"out=\"${home}/Demo_Title [abc123].mp4\"\n" +
		"mkdir -p \"${home}\"\n" +
		"printf 'fake media' > \"${out}\"\n" +
		"if [[ " + shellBool(options.EmptyFinalPath) + " == false ]]; then printf '%s\\n' \"${out}\"; fi\n"
	if err := os.WriteFile(path, []byte(script), 0o755); err != nil {
		t.Fatalf("write fake yt-dlp: %v", err)
	}
	return fakeYTDLP{Path: path}
}

func shellQuote(value string) string {
	return "'" + strings.ReplaceAll(value, "'", "'\\''") + "'"
}

func shellBool(value bool) string {
	if value {
		return "true"
	}
	return "false"
}

func splitFakeYTDLPCalls(output string) [][]string {
	lines := strings.Split(strings.TrimSpace(output), "\n")
	var calls [][]string
	for _, line := range lines {
		if line == "CALL" {
			calls = append(calls, []string{})
			continue
		}
		if len(calls) == 0 {
			continue
		}
		calls[len(calls)-1] = append(calls[len(calls)-1], line)
	}
	return calls
}
