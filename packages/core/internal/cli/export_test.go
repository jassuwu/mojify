package cli

import (
	"bytes"
	"testing"
)

func TestIsTerminalWriterReturnsFalseForPlainWriter(t *testing.T) {
	var writer bytes.Buffer

	if isTerminalWriter(&writer) {
		t.Fatal("isTerminalWriter returned true for bytes.Buffer")
	}
}

type fakeFDWriter struct{}

func (fakeFDWriter) Write(p []byte) (int, error) {
	return len(p), nil
}

func (fakeFDWriter) Fd() uintptr {
	return 2
}

func TestIsTerminalWriterReturnsFalseForNonFileWriterWithFD(t *testing.T) {
	if isTerminalWriter(fakeFDWriter{}) {
		t.Fatal("isTerminalWriter returned true for non-file writer")
	}
}
