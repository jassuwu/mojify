package media

import (
	"reflect"
	"testing"
)

func TestMP4EncodeArgsMapsOptionalAudio(t *testing.T) {
	args, err := MP4EncodeArgs(MP4EncodeOptions{
		InputPath:  "source.mov",
		OutputPath: "out.mp4",
		Width:      640,
		Height:     360,
		FPS:        24,
		Bitrate:    "2M",
	})
	if err != nil {
		t.Fatalf("MP4EncodeArgs returned error: %v", err)
	}

	want := []string{
		"-v", "error",
		"-n",
		"-f", "rawvideo",
		"-pix_fmt", "rgb24",
		"-s", "640x360",
		"-r", "24",
		"-i", "pipe:0",
		"-i", "source.mov",
		"-map", "0:v:0",
		"-map", "1:a?",
		"-c:v", "libx264",
		"-preset", "veryfast",
		"-pix_fmt", "yuv420p",
		"-c:a", "aac",
		"-shortest",
		"-b:v", "2M",
		"out.mp4",
	}
	if !reflect.DeepEqual(args, want) {
		t.Fatalf("args = %#v, want %#v", args, want)
	}
}

func TestMP4EncodeArgsUsesVeryfastPreset(t *testing.T) {
	args, err := MP4EncodeArgs(MP4EncodeOptions{
		InputPath:  "source.mov",
		OutputPath: "out.mp4",
		Width:      640,
		Height:     360,
		FPS:        24,
	})
	if err != nil {
		t.Fatalf("MP4EncodeArgs returned error: %v", err)
	}
	if !containsAdjacent(args, "-preset", "veryfast") {
		t.Fatalf("args missing -preset veryfast: %#v", args)
	}
}

func TestMP4EncodeArgsUsesOverwriteFlag(t *testing.T) {
	args, err := MP4EncodeArgs(MP4EncodeOptions{
		InputPath:  "source.mov",
		OutputPath: "out.mp4",
		Width:      640,
		Height:     360,
		FPS:        24,
		Overwrite:  true,
	})
	if err != nil {
		t.Fatalf("MP4EncodeArgs returned error: %v", err)
	}

	want := []string{
		"-v", "error",
		"-y",
		"-f", "rawvideo",
		"-pix_fmt", "rgb24",
		"-s", "640x360",
		"-r", "24",
		"-i", "pipe:0",
		"-i", "source.mov",
		"-map", "0:v:0",
		"-map", "1:a?",
		"-c:v", "libx264",
		"-preset", "veryfast",
		"-pix_fmt", "yuv420p",
		"-c:a", "aac",
		"-shortest",
		"out.mp4",
	}
	if !reflect.DeepEqual(args, want) {
		t.Fatalf("args = %#v, want %#v", args, want)
	}
}

func TestMP4EncodeArgsRejectsInvalidDimensions(t *testing.T) {
	for _, tc := range []struct {
		name   string
		width  int
		height int
	}{
		{name: "zero width", width: 0, height: 360},
		{name: "zero height", width: 640, height: 0},
		{name: "negative width", width: -1, height: 360},
		{name: "negative height", width: 640, height: -1},
	} {
		t.Run(tc.name, func(t *testing.T) {
			_, err := MP4EncodeArgs(MP4EncodeOptions{
				InputPath:  "source.mov",
				OutputPath: "out.mp4",
				Width:      tc.width,
				Height:     tc.height,
				FPS:        24,
			})
			if err == nil {
				t.Fatal("MP4EncodeArgs returned nil error")
			}
		})
	}
}

func containsAdjacent(args []string, key, value string) bool {
	for i := 0; i+1 < len(args); i++ {
		if args[i] == key && args[i+1] == value {
			return true
		}
	}
	return false
}
