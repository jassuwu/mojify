package media

import (
	"reflect"
	"testing"
)

func TestFFplayAudioArgs(t *testing.T) {
	args, err := FFplayAudioArgs("clip.mp4")
	if err != nil {
		t.Fatalf("FFplayAudioArgs returned error: %v", err)
	}
	want := []string{
		"-nodisp",
		"-autoexit",
		"-loglevel", "error",
		"-vn",
		"clip.mp4",
	}
	if !reflect.DeepEqual(args, want) {
		t.Fatalf("args = %#v, want %#v", args, want)
	}
}

func TestFFplayAudioArgsRejectsMissingInput(t *testing.T) {
	_, err := FFplayAudioArgs(" ")
	if err == nil {
		t.Fatal("FFplayAudioArgs returned nil error for missing input")
	}
}
