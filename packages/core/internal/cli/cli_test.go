package cli

import "testing"

func TestParseBareCommandShowsHelp(t *testing.T) {
	cmd, err := Parse([]string{})
	if err != nil {
		t.Fatalf("Parse returned error: %v", err)
	}
	if cmd.Kind != HelpCommand {
		t.Fatalf("Kind = %v, want %v", cmd.Kind, HelpCommand)
	}
}

func TestParsePlayCommand(t *testing.T) {
	cmd, err := Parse([]string{"play", "clip.mp4"})
	if err != nil {
		t.Fatalf("Parse returned error: %v", err)
	}
	if cmd.Kind != PlayCommand {
		t.Fatalf("Kind = %v, want %v", cmd.Kind, PlayCommand)
	}
	if cmd.InputPath != "clip.mp4" {
		t.Fatalf("InputPath = %q, want clip.mp4", cmd.InputPath)
	}
}

func TestParsePlayStatsBeforeInput(t *testing.T) {
	cmd, err := Parse([]string{"play", "--stats", "clip.mp4"})
	if err != nil {
		t.Fatalf("Parse returned error: %v", err)
	}
	if cmd.Kind != PlayCommand {
		t.Fatalf("Kind = %v, want %v", cmd.Kind, PlayCommand)
	}
	if cmd.InputPath != "clip.mp4" {
		t.Fatalf("InputPath = %q, want clip.mp4", cmd.InputPath)
	}
	if !cmd.Stats {
		t.Fatal("Stats = false, want true")
	}
}

func TestParsePlayStatsAfterInput(t *testing.T) {
	cmd, err := Parse([]string{"play", "clip.mp4", "--stats"})
	if err != nil {
		t.Fatalf("Parse returned error: %v", err)
	}
	if cmd.InputPath != "clip.mp4" {
		t.Fatalf("InputPath = %q, want clip.mp4", cmd.InputPath)
	}
	if !cmd.Stats {
		t.Fatal("Stats = false, want true")
	}
}

func TestParsePlayNoAudioBeforeInput(t *testing.T) {
	cmd, err := Parse([]string{"play", "--no-audio", "clip.mp4"})
	if err != nil {
		t.Fatalf("Parse returned error: %v", err)
	}
	if cmd.Kind != PlayCommand {
		t.Fatalf("Kind = %v, want %v", cmd.Kind, PlayCommand)
	}
	if cmd.InputPath != "clip.mp4" {
		t.Fatalf("InputPath = %q, want clip.mp4", cmd.InputPath)
	}
	if !cmd.NoAudio {
		t.Fatal("NoAudio = false, want true")
	}
}

func TestParsePlayNoAudioAfterInput(t *testing.T) {
	cmd, err := Parse([]string{"play", "clip.mp4", "--no-audio"})
	if err != nil {
		t.Fatalf("Parse returned error: %v", err)
	}
	if !cmd.NoAudio {
		t.Fatal("NoAudio = false, want true")
	}
}

func TestParsePlayRejectsDuplicateNoAudio(t *testing.T) {
	_, err := Parse([]string{"play", "--no-audio", "clip.mp4", "--no-audio"})
	if err == nil {
		t.Fatal("Parse returned nil error for duplicate --no-audio")
	}
}

func TestParseRejectsNoAudioForProbe(t *testing.T) {
	_, err := Parse([]string{"probe", "--no-audio", "clip.mp4"})
	if err == nil {
		t.Fatal("Parse returned nil error for probe --no-audio")
	}
}

func TestParseRejectsStatsForProbe(t *testing.T) {
	_, err := Parse([]string{"probe", "--stats", "clip.mp4"})
	if err == nil {
		t.Fatal("Parse returned nil error for probe --stats")
	}
}

func TestParseRejectsDuplicateStats(t *testing.T) {
	_, err := Parse([]string{"play", "--stats", "clip.mp4", "--stats"})
	if err == nil {
		t.Fatal("Parse returned nil error for duplicate --stats")
	}
}

func TestParseProbeCommand(t *testing.T) {
	cmd, err := Parse([]string{"probe", "clip.mp4"})
	if err != nil {
		t.Fatalf("Parse returned error: %v", err)
	}
	if cmd.Kind != ProbeCommand {
		t.Fatalf("Kind = %v, want %v", cmd.Kind, ProbeCommand)
	}
}

func TestParseExportCommand(t *testing.T) {
	cmd, err := Parse([]string{"export", "clip.mov", "clip.mp4"})
	if err != nil {
		t.Fatalf("Parse returned error: %v", err)
	}
	if cmd.Kind != ExportCommand {
		t.Fatalf("Kind = %v, want %v", cmd.Kind, ExportCommand)
	}
	if cmd.InputPath != "clip.mov" {
		t.Fatalf("InputPath = %q, want clip.mov", cmd.InputPath)
	}
	if cmd.OutputPath != "clip.mp4" {
		t.Fatalf("OutputPath = %q, want clip.mp4", cmd.OutputPath)
	}
}

func TestParseExportFlags(t *testing.T) {
	cmd, err := Parse([]string{
		"export",
		"--width", "120",
		"clip.mov",
		"--fps", "29.97",
		"--bitrate", "4M",
		"--overwrite",
		"CLIP.MP4",
	})
	if err != nil {
		t.Fatalf("Parse returned error: %v", err)
	}
	if cmd.Export.Width != 120 {
		t.Fatalf("Export.Width = %d, want 120", cmd.Export.Width)
	}
	if cmd.Export.FPS != 29.97 {
		t.Fatalf("Export.FPS = %v, want 29.97", cmd.Export.FPS)
	}
	if cmd.Export.Bitrate != "4M" {
		t.Fatalf("Export.Bitrate = %q, want 4M", cmd.Export.Bitrate)
	}
	if !cmd.Export.Overwrite {
		t.Fatal("Export.Overwrite = false, want true")
	}
	if cmd.OutputPath != "CLIP.MP4" {
		t.Fatalf("OutputPath = %q, want CLIP.MP4", cmd.OutputPath)
	}
}

func TestParseExportStatsAndWorkers(t *testing.T) {
	cmd, err := Parse([]string{
		"export",
		"--stats",
		"--workers", "6",
		"clip.mov",
		"clip.mp4",
	})
	if err != nil {
		t.Fatalf("Parse returned error: %v", err)
	}
	if !cmd.Export.Stats {
		t.Fatal("Export.Stats = false, want true")
	}
	if cmd.Export.Workers != 6 {
		t.Fatalf("Export.Workers = %d, want 6", cmd.Export.Workers)
	}
}

func TestParseExportRejectsDuplicateStats(t *testing.T) {
	_, err := Parse([]string{"export", "--stats", "clip.mov", "--stats", "clip.mp4"})
	if err == nil {
		t.Fatal("Parse returned nil error for duplicate export --stats")
	}
}

func TestParseExportRejectsNoAudio(t *testing.T) {
	_, err := Parse([]string{"export", "--no-audio", "clip.mov", "clip.mp4"})
	if err == nil {
		t.Fatal("Parse returned nil error for export --no-audio")
	}
}

func TestParseExportRejectsInvalidWorkers(t *testing.T) {
	for _, workers := range []string{"0", "-1", "many"} {
		_, err := Parse([]string{"export", "--workers", workers, "clip.mov", "clip.mp4"})
		if err == nil {
			t.Fatalf("Parse returned nil error for invalid --workers %q", workers)
		}
	}
}

func TestParseMissingInput(t *testing.T) {
	_, err := Parse([]string{"play"})
	if err == nil {
		t.Fatal("Parse returned nil error for missing input")
	}
}

func TestParseExportMissingOutput(t *testing.T) {
	_, err := Parse([]string{"export", "clip.mov"})
	if err == nil {
		t.Fatal("Parse returned nil error for missing export output")
	}
}

func TestParseExportRejectsExtraInputs(t *testing.T) {
	_, err := Parse([]string{"export", "clip.mov", "clip.mp4", "extra.mp4"})
	if err == nil {
		t.Fatal("Parse returned nil error for extra export input")
	}
}

func TestParseAcceptsHTTPSources(t *testing.T) {
	for _, command := range []string{"play", "probe"} {
		for _, input := range []string{
			"https://example.com/watch?v=demo",
			"http://example.com/video",
		} {
			cmd, err := Parse([]string{command, input})
			if err != nil {
				t.Fatalf("Parse(%s %q) returned error: %v", command, input, err)
			}
			if cmd.InputPath != input {
				t.Fatalf("InputPath = %q, want %q", cmd.InputPath, input)
			}
		}
	}
}

func TestParseExportAcceptsHTTPSource(t *testing.T) {
	cmd, err := Parse([]string{"export", "https://example.com/watch?v=demo", "clip.mp4"})
	if err != nil {
		t.Fatalf("Parse returned error: %v", err)
	}
	if cmd.InputPath != "https://example.com/watch?v=demo" {
		t.Fatalf("InputPath = %q, want URL source", cmd.InputPath)
	}
	if cmd.OutputPath != "clip.mp4" {
		t.Fatalf("OutputPath = %q, want clip.mp4", cmd.OutputPath)
	}
}

func TestParseRejectsUnsupportedProtocolInputs(t *testing.T) {
	for _, command := range []string{"play", "probe"} {
		for _, input := range []string{
			"file:///tmp/demo.mp4",
			"pipe:0",
			"concat:part1.mp4|part2.mp4",
			"ytsearch:demo query",
			"-",
		} {
			_, err := Parse([]string{command, input})
			if err == nil {
				t.Fatalf("Parse accepted unsupported %s input %q", command, input)
			}
		}
	}
}

func TestParseExportRejectsUnsupportedProtocolInput(t *testing.T) {
	for _, input := range []string{
		"file:///tmp/demo.mp4",
		"pipe:0",
		"concat:part1.mp4|part2.mp4",
		"ytsearch:demo query",
		"-",
	} {
		_, err := Parse([]string{"export", input, "clip.mp4"})
		if err == nil {
			t.Fatalf("Parse returned nil error for unsupported export input %q", input)
		}
	}
}

func TestParseExportRejectsProtocolOutput(t *testing.T) {
	_, err := Parse([]string{"export", "clip.mov", "file:///tmp/clip.mp4"})
	if err == nil {
		t.Fatal("Parse returned nil error for protocol export output")
	}
}

func TestParseExportRejectsNonMP4Output(t *testing.T) {
	_, err := Parse([]string{"export", "clip.mov", "clip.mov"})
	if err == nil {
		t.Fatal("Parse returned nil error for non-MP4 export output")
	}
}

func TestParseExportRejectsInvalidWidth(t *testing.T) {
	_, err := Parse([]string{"export", "--width", "0", "clip.mov", "clip.mp4"})
	if err == nil {
		t.Fatal("Parse returned nil error for invalid export width")
	}
}

func TestParseExportRejectsInvalidFPS(t *testing.T) {
	_, err := Parse([]string{"export", "--fps", "0", "clip.mov", "clip.mp4"})
	if err == nil {
		t.Fatal("Parse returned nil error for invalid export FPS")
	}
}

func TestParseExportRejectsInvalidBitrate(t *testing.T) {
	_, err := Parse([]string{"export", "--bitrate", "4mbps", "clip.mov", "clip.mp4"})
	if err == nil {
		t.Fatal("Parse returned nil error for invalid export bitrate")
	}
}

func TestParseExportRejectsUnknownOption(t *testing.T) {
	_, err := Parse([]string{"export", "--height", "80", "clip.mov", "clip.mp4"})
	if err == nil {
		t.Fatal("Parse returned nil error for unknown export option")
	}
}

func TestHelpTextMentionsCommands(t *testing.T) {
	help := HelpText()
	for _, want := range []string{
		"mojify play [--stats] [--no-audio] <source>",
		"Play source media in the terminal",
		"mojify probe <source>",
		"Print source media and render metadata",
		"mojify export [options] <source> <output.mp4>",
		"Export Mojify visuals to an MP4 file",
		"<source> may be a local video file or an HTTP(S) platform URL",
		"yt-dlp is required for platform URL inputs",
		"--width <px>",
		"--fps <n>",
		"--bitrate <value>",
		"--overwrite",
		"--stats",
		"--no-audio",
		"--workers <n>",
		"FFmpeg and ffprobe",
		"ffplay is required for live playback audio unless --no-audio is used",
	} {
		if !contains(help, want) {
			t.Fatalf("HelpText() missing %q in:\n%s", want, help)
		}
	}
}

func contains(s string, needle string) bool {
	for i := 0; i+len(needle) <= len(s); i++ {
		if s[i:i+len(needle)] == needle {
			return true
		}
	}
	return false
}
