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

func TestParseRejectsProtocolInputs(t *testing.T) {
	for _, command := range []string{"play", "probe"} {
		for _, input := range []string{
			"https://example.com/demo.mp4",
			"file:///tmp/demo.mp4",
			"pipe:0",
			"concat:part1.mp4|part2.mp4",
			"-",
		} {
			_, err := Parse([]string{command, input})
			if err == nil {
				t.Fatalf("Parse accepted non-local %s input %q", command, input)
			}
		}
	}
}

func TestParseExportRejectsProtocolInput(t *testing.T) {
	_, err := Parse([]string{"export", "https://example.com/demo.mp4", "clip.mp4"})
	if err == nil {
		t.Fatal("Parse returned nil error for protocol export input")
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
		"mojify play [--stats] <video>",
		"Play a local video file in the terminal",
		"mojify probe <video>",
		"Print media and render metadata",
		"mojify export [options] <video> <output.mp4>",
		"Export Mojify visuals to an MP4 file",
		"--width <px>",
		"--fps <n>",
		"--bitrate <value>",
		"--overwrite",
		"--stats",
		"--workers <n>",
		"FFmpeg and ffprobe",
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
