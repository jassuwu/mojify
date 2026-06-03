package exporter

import (
	"fmt"
	"path/filepath"
	"strings"
)

type OutputFamily string

const (
	OutputFamilyVideo      OutputFamily = "video"
	OutputFamilyAnimated   OutputFamily = "animated"
	OutputFamilyStillImage OutputFamily = "still image"
	OutputFamilyText       OutputFamily = "text"
)

type OutputFormat struct {
	Extension     string
	Family        OutputFamily
	TimeBased     bool
	SingleFrame   bool
	SupportsAudio bool
	Text          bool
}

var supportedOutputFormats = []OutputFormat{
	{Extension: ".mp4", Family: OutputFamilyVideo, TimeBased: true, SupportsAudio: true},
	{Extension: ".webm", Family: OutputFamilyVideo, TimeBased: true, SupportsAudio: true},
	{Extension: ".mov", Family: OutputFamilyVideo, TimeBased: true, SupportsAudio: true},
	{Extension: ".gif", Family: OutputFamilyAnimated, TimeBased: true},
	{Extension: ".apng", Family: OutputFamilyAnimated, TimeBased: true},
	{Extension: ".png", Family: OutputFamilyStillImage, SingleFrame: true},
	{Extension: ".jpg", Family: OutputFamilyStillImage, SingleFrame: true},
	{Extension: ".jpeg", Family: OutputFamilyStillImage, SingleFrame: true},
	{Extension: ".txt", Family: OutputFamilyText, SingleFrame: true, Text: true},
	{Extension: ".ansi", Family: OutputFamilyText, SingleFrame: true, Text: true},
}

func ResolveOutputFormat(outputPath string) (OutputFormat, error) {
	ext := strings.ToLower(filepath.Ext(outputPath))
	for _, format := range supportedOutputFormats {
		if ext == format.Extension {
			return format, nil
		}
	}
	return OutputFormat{}, fmt.Errorf("unsupported export output extension %q; supported extensions: %s", ext, SupportedOutputExtensionsText())
}

func SupportedOutputExtensionsText() string {
	extensions := make([]string, 0, len(supportedOutputFormats))
	for _, format := range supportedOutputFormats {
		extensions = append(extensions, format.Extension)
	}
	return strings.Join(extensions, ", ")
}
