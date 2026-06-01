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

func TestParseProbeCommand(t *testing.T) {
	cmd, err := Parse([]string{"probe", "clip.mp4"})
	if err != nil {
		t.Fatalf("Parse returned error: %v", err)
	}
	if cmd.Kind != ProbeCommand {
		t.Fatalf("Kind = %v, want %v", cmd.Kind, ProbeCommand)
	}
}

func TestParseMissingInput(t *testing.T) {
	_, err := Parse([]string{"play"})
	if err == nil {
		t.Fatal("Parse returned nil error for missing input")
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

func TestHelpTextMentionsCommands(t *testing.T) {
	help := HelpText()
	for _, want := range []string{
		"mojify play <video>",
		"Play a local video file in the terminal",
		"mojify probe <video>",
		"Print media and render metadata",
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
