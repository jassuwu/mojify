package fonts

import "testing"

func TestDefaultFaceLoads(t *testing.T) {
	face, err := DefaultFace()
	if err != nil {
		t.Fatalf("DefaultFace returned error: %v", err)
	}
	if face == nil {
		t.Fatal("DefaultFace returned nil face")
	}
}
