package media

import (
	"bytes"
	"io"
	"reflect"
	"testing"
)

func TestDecodeArgs(t *testing.T) {
	args := DecodeArgs("clip.mp4", 320, 180)
	want := []string{
		"-v", "error",
		"-i", "clip.mp4",
		"-vf", "scale=320:180:force_original_aspect_ratio=decrease,pad=320:180:(ow-iw)/2:(oh-ih)/2",
		"-f", "rawvideo",
		"-pix_fmt", "rgb24",
		"-",
	}
	if !reflect.DeepEqual(args, want) {
		t.Fatalf("args = %#v, want %#v", args, want)
	}
}

func TestExportDecodeArgsWithoutFPS(t *testing.T) {
	args := ExportDecodeArgs("clip.mp4", 640, 360, 0)
	want := []string{
		"-v", "error",
		"-i", "clip.mp4",
		"-vf", "scale=640:360:force_original_aspect_ratio=decrease,pad=640:360:(ow-iw)/2:(oh-ih)/2",
		"-f", "rawvideo",
		"-pix_fmt", "rgb24",
		"-",
	}
	if !reflect.DeepEqual(args, want) {
		t.Fatalf("args = %#v, want %#v", args, want)
	}
}

func TestExportDecodeArgsWithFPS(t *testing.T) {
	args := ExportDecodeArgs("clip.mp4", 640, 360, 24)
	want := []string{
		"-v", "error",
		"-i", "clip.mp4",
		"-vf", "scale=640:360:force_original_aspect_ratio=decrease,pad=640:360:(ow-iw)/2:(oh-ih)/2,fps=24",
		"-f", "rawvideo",
		"-pix_fmt", "rgb24",
		"-",
	}
	if !reflect.DeepEqual(args, want) {
		t.Fatalf("args = %#v, want %#v", args, want)
	}
}

func TestReadRawFrame(t *testing.T) {
	data := []byte{1, 2, 3, 4, 5, 6}
	frame, err := ReadRawFrame(bytes.NewReader(data), 1, 2)
	if err != nil {
		t.Fatalf("ReadRawFrame returned error: %v", err)
	}
	if frame.Width != 1 || frame.Height != 2 {
		t.Fatalf("size = %dx%d, want 1x2", frame.Width, frame.Height)
	}
	if !reflect.DeepEqual(frame.Data, data) {
		t.Fatalf("data = %#v, want %#v", frame.Data, data)
	}
}

func TestReadRawFrameEOF(t *testing.T) {
	_, err := ReadRawFrame(bytes.NewReader(nil), 1, 1)
	if err != io.EOF {
		t.Fatalf("err = %v, want io.EOF", err)
	}
}

func TestReadRawFrameUnexpectedEOF(t *testing.T) {
	_, err := ReadRawFrame(bytes.NewReader([]byte{1, 2}), 1, 1)
	if err != io.ErrUnexpectedEOF {
		t.Fatalf("err = %v, want io.ErrUnexpectedEOF", err)
	}
}

func TestReadRawFrameRejectsInvalidDimensions(t *testing.T) {
	for _, tc := range []struct {
		width  int
		height int
	}{
		{0, 1},
		{1, 0},
		{-1, 1},
		{1, -1},
	} {
		_, err := ReadRawFrame(bytes.NewReader(nil), tc.width, tc.height)
		if err == nil {
			t.Fatalf("ReadRawFrame(%d, %d) returned nil error", tc.width, tc.height)
		}
	}
}
