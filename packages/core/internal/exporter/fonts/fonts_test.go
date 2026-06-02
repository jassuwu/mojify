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

func TestDefaultFaceReturnsDistinctFaces(t *testing.T) {
	first, err := DefaultFace()
	if err != nil {
		t.Fatalf("DefaultFace first call returned error: %v", err)
	}
	second, err := DefaultFace()
	if err != nil {
		t.Fatalf("DefaultFace second call returned error: %v", err)
	}
	if first == nil || second == nil {
		t.Fatal("DefaultFace returned nil face")
	}
	if first == second {
		t.Fatal("DefaultFace returned the same face instance twice")
	}
}
