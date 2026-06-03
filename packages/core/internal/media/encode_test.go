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

func TestRawVideoEncodeArgsForWebM(t *testing.T) {
	args, err := RawVideoEncodeArgs(RawVideoEncodeOptions{
		Format:       EncodeFormatWebM,
		InputPath:    "source.mov",
		OutputPath:   "out.webm",
		Width:        320,
		Height:       184,
		FPS:          12,
		Overwrite:    true,
		IncludeAudio: true,
	})
	if err != nil {
		t.Fatalf("RawVideoEncodeArgs returned error: %v", err)
	}
	for _, pair := range [][2]string{
		{"-c:v", "libvpx-vp9"},
		{"-pix_fmt", "yuv420p"},
		{"-c:a", "libopus"},
	} {
		if !containsAdjacent(args, pair[0], pair[1]) {
			t.Fatalf("args missing %s %s: %#v", pair[0], pair[1], args)
		}
	}
}

func TestRawVideoEncodeArgsForMOV(t *testing.T) {
	args, err := RawVideoEncodeArgs(RawVideoEncodeOptions{
		Format:       EncodeFormatMOV,
		InputPath:    "source.mov",
		OutputPath:   "out.mov",
		Width:        320,
		Height:       184,
		FPS:          12,
		IncludeAudio: true,
	})
	if err != nil {
		t.Fatalf("RawVideoEncodeArgs returned error: %v", err)
	}
	for _, pair := range [][2]string{
		{"-c:v", "libx264"},
		{"-pix_fmt", "yuv420p"},
		{"-c:a", "aac"},
	} {
		if !containsAdjacent(args, pair[0], pair[1]) {
			t.Fatalf("args missing %s %s: %#v", pair[0], pair[1], args)
		}
	}
}

func TestRawVideoEncodeArgsForAnimatedGIF(t *testing.T) {
	args, err := RawVideoEncodeArgs(RawVideoEncodeOptions{
		Format:     EncodeFormatGIF,
		OutputPath: "out.gif",
		Width:      320,
		Height:     184,
		FPS:        12,
		Overwrite:  true,
	})
	if err != nil {
		t.Fatalf("RawVideoEncodeArgs returned error: %v", err)
	}
	if !containsAdjacent(args, "-filter_complex", "split[s0][s1];[s0]palettegen[p];[s1][p]paletteuse") {
		t.Fatalf("args missing GIF palette filter: %#v", args)
	}
	if !containsAdjacent(args, "-loop", "0") {
		t.Fatalf("args missing -loop 0: %#v", args)
	}
}

func TestRawVideoEncodeArgsForAnimatedAPNG(t *testing.T) {
	args, err := RawVideoEncodeArgs(RawVideoEncodeOptions{
		Format:     EncodeFormatAPNG,
		OutputPath: "out.apng",
		Width:      320,
		Height:     184,
		FPS:        12,
	})
	if err != nil {
		t.Fatalf("RawVideoEncodeArgs returned error: %v", err)
	}
	if !containsAdjacent(args, "-c:v", "apng") {
		t.Fatalf("args missing -c:v apng: %#v", args)
	}
	if !containsAdjacent(args, "-plays", "0") {
		t.Fatalf("args missing -plays 0: %#v", args)
	}
}

func TestRawVideoEncodeArgsForStillPNG(t *testing.T) {
	args, err := RawVideoEncodeArgs(RawVideoEncodeOptions{
		Format:     EncodeFormatPNG,
		OutputPath: "out.png",
		Width:      320,
		Height:     184,
		FPS:        1,
		Overwrite:  true,
	})
	if err != nil {
		t.Fatalf("RawVideoEncodeArgs returned error: %v", err)
	}
	if !containsAdjacent(args, "-frames:v", "1") {
		t.Fatalf("args missing -frames:v 1: %#v", args)
	}
	if !containsAdjacent(args, "-c:v", "png") {
		t.Fatalf("args missing -c:v png: %#v", args)
	}
}

func TestRawVideoEncodeArgsForStillJPEG(t *testing.T) {
	args, err := RawVideoEncodeArgs(RawVideoEncodeOptions{
		Format:     EncodeFormatJPEG,
		OutputPath: "out.jpg",
		Width:      320,
		Height:     184,
		FPS:        1,
	})
	if err != nil {
		t.Fatalf("RawVideoEncodeArgs returned error: %v", err)
	}
	for _, pair := range [][2]string{
		{"-frames:v", "1"},
		{"-c:v", "mjpeg"},
		{"-q:v", "2"},
	} {
		if !containsAdjacent(args, pair[0], pair[1]) {
			t.Fatalf("args missing %s %s: %#v", pair[0], pair[1], args)
		}
	}
}

func TestRawVideoEncodeArgsRejectsAudioFormatWithoutInputPath(t *testing.T) {
	_, err := RawVideoEncodeArgs(RawVideoEncodeOptions{
		Format:       EncodeFormatWebM,
		OutputPath:   "out.webm",
		Width:        320,
		Height:       184,
		FPS:          12,
		IncludeAudio: true,
	})
	if err == nil {
		t.Fatal("RawVideoEncodeArgs returned nil error without input path for audio-capable format")
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
