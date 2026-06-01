package media

import "testing"

func TestParseProbeJSON(t *testing.T) {
	const input = `{
	  "streams": [
	    {
	      "codec_type": "video",
	      "width": 1920,
	      "height": 1080,
	      "avg_frame_rate": "30000/1001",
	      "nb_frames": "240"
	    }
	  ],
	  "format": { "duration": "8.008000" }
	}`

	info, err := ParseProbeJSON([]byte(input))
	if err != nil {
		t.Fatalf("ParseProbeJSON returned error: %v", err)
	}
	if info.Width != 1920 || info.Height != 1080 {
		t.Fatalf("size = %dx%d, want 1920x1080", info.Width, info.Height)
	}
	if info.FPS < 29.96 || info.FPS > 29.98 {
		t.Fatalf("FPS = %f, want about 29.97", info.FPS)
	}
	if info.FrameCount != 240 {
		t.Fatalf("FrameCount = %d, want 240", info.FrameCount)
	}
	if info.DurationSeconds != 8.008 {
		t.Fatalf("DurationSeconds = %f, want 8.008", info.DurationSeconds)
	}
}

func TestParseProbeJSONRejectsMissingVideo(t *testing.T) {
	_, err := ParseProbeJSON([]byte(`{"streams":[]}`))
	if err == nil {
		t.Fatal("ParseProbeJSON returned nil error for missing video stream")
	}
}

func TestParseProbeJSONRejectsMissingVideoDimensions(t *testing.T) {
	const input = `{
	  "streams": [
	    {
	      "codec_type": "video",
	      "avg_frame_rate": "24/1"
	    }
	  ],
	  "format": { "duration": "1.000000" }
	}`

	_, err := ParseProbeJSON([]byte(input))
	if err == nil {
		t.Fatal("ParseProbeJSON returned nil error for missing dimensions")
	}
}
