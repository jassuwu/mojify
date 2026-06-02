package exporter

import (
	"os"
	"path/filepath"
	"testing"
)

func TestCheckOutputPathRejectsExistingWithoutOverwrite(t *testing.T) {
	dir := t.TempDir()
	output := filepath.Join(dir, "out.mp4")
	if err := os.WriteFile(output, []byte("existing"), 0o644); err != nil {
		t.Fatalf("write output fixture: %v", err)
	}
	err := checkOutputPath(output, Options{})
	if err == nil {
		t.Fatal("checkOutputPath returned nil error for existing output")
	}
}

func TestCheckOutputPathAllowsOverwrite(t *testing.T) {
	dir := t.TempDir()
	output := filepath.Join(dir, "out.mp4")
	if err := os.WriteFile(output, []byte("existing"), 0o644); err != nil {
		t.Fatalf("write output fixture: %v", err)
	}
	err := checkOutputPath(output, Options{Overwrite: true})
	if err != nil {
		t.Fatalf("checkOutputPath returned error: %v", err)
	}
}
