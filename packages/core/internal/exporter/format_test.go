package exporter

import "testing"

func TestResolveOutputFormatSupportsCuratedExtensions(t *testing.T) {
	tests := []struct {
		path   string
		ext    string
		family OutputFamily
	}{
		{"out.mp4", ".mp4", OutputFamilyVideo},
		{"out.webm", ".webm", OutputFamilyVideo},
		{"out.mov", ".mov", OutputFamilyVideo},
		{"out.gif", ".gif", OutputFamilyAnimated},
		{"out.apng", ".apng", OutputFamilyAnimated},
		{"out.png", ".png", OutputFamilyStillImage},
		{"out.jpg", ".jpg", OutputFamilyStillImage},
		{"out.jpeg", ".jpeg", OutputFamilyStillImage},
		{"out.txt", ".txt", OutputFamilyText},
		{"out.ansi", ".ansi", OutputFamilyText},
	}
	for _, tc := range tests {
		t.Run(tc.path, func(t *testing.T) {
			format, err := ResolveOutputFormat(tc.path)
			if err != nil {
				t.Fatalf("ResolveOutputFormat returned error: %v", err)
			}
			if format.Extension != tc.ext {
				t.Fatalf("Extension = %q, want %q", format.Extension, tc.ext)
			}
			if format.Family != tc.family {
				t.Fatalf("Family = %q, want %q", format.Family, tc.family)
			}
		})
	}
}

func TestResolveOutputFormatRejectsUnsupportedExtensions(t *testing.T) {
	for _, path := range []string{"out.webp", "out.bmp", "out", ".gitignore"} {
		_, err := ResolveOutputFormat(path)
		if err == nil {
			t.Fatalf("ResolveOutputFormat(%q) returned nil error", path)
		}
	}
}

func TestOutputFormatCapabilities(t *testing.T) {
	tests := []struct {
		path          string
		timeBased     bool
		singleFrame   bool
		supportsAudio bool
		text          bool
	}{
		{"out.mp4", true, false, true, false},
		{"out.webm", true, false, true, false},
		{"out.mov", true, false, true, false},
		{"out.gif", true, false, false, false},
		{"out.apng", true, false, false, false},
		{"out.png", false, true, false, false},
		{"out.jpg", false, true, false, false},
		{"out.txt", false, true, false, true},
		{"out.ansi", false, true, false, true},
	}
	for _, tc := range tests {
		t.Run(tc.path, func(t *testing.T) {
			format, err := ResolveOutputFormat(tc.path)
			if err != nil {
				t.Fatalf("ResolveOutputFormat returned error: %v", err)
			}
			if format.TimeBased != tc.timeBased {
				t.Fatalf("TimeBased = %v, want %v", format.TimeBased, tc.timeBased)
			}
			if format.SingleFrame != tc.singleFrame {
				t.Fatalf("SingleFrame = %v, want %v", format.SingleFrame, tc.singleFrame)
			}
			if format.SupportsAudio != tc.supportsAudio {
				t.Fatalf("SupportsAudio = %v, want %v", format.SupportsAudio, tc.supportsAudio)
			}
			if format.Text != tc.text {
				t.Fatalf("Text = %v, want %v", format.Text, tc.text)
			}
		})
	}
}

func TestSupportedOutputExtensionsText(t *testing.T) {
	got := SupportedOutputExtensionsText()
	want := ".mp4, .webm, .mov, .gif, .apng, .png, .jpg, .jpeg, .txt, .ansi"
	if got != want {
		t.Fatalf("SupportedOutputExtensionsText() = %q, want %q", got, want)
	}
}
