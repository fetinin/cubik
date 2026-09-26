package main

import (
	"cubik/api"
	"encoding/json"
	"strings"
	"testing"
)

func mustDecodeSparse(t *testing.T, payload string) *api.SparseAnimation {
	t.Helper()
	var anim api.SparseAnimation
	if err := json.Unmarshal([]byte(payload), &anim); err != nil {
		t.Fatalf("decode fixture: %v", err)
	}
	return &anim
}

// TestValidatePixelBounds_XOutOfDeclaredDims confirms the cross-field
// bounds check catches an x coordinate that fits the schema-level 0..254
// cap but exceeds the animation's declared width.
func TestValidatePixelBounds_XOutOfDeclaredDims(t *testing.T) {
	anim := mustDecodeSparse(t, `{
		"version": "1.0",
		"name": "Bad X",
		"width": 10,
		"height": 5,
		"frames": [
			[{"x": 12, "y": 0, "c": 100}]
		]
	}`)
	verr := ValidatePixelBounds(anim)
	if verr == nil {
		t.Fatal("expected ValidationError for x=12 with width=10, got nil")
	}
	if got, want := verr.Field, "frames[0].pixels[0].x"; got != want {
		t.Errorf("Field = %q, want %q", got, want)
	}
	if !strings.Contains(verr.Reason, "width") {
		t.Errorf("expected reason to reference width bound; got: %s", verr.Reason)
	}
}

// TestValidatePixelBounds_YOutOfDeclaredDims is the symmetric check for y.
func TestValidatePixelBounds_YOutOfDeclaredDims(t *testing.T) {
	anim := mustDecodeSparse(t, `{
		"version": "1.0",
		"name": "Bad Y",
		"width": 10,
		"height": 5,
		"frames": [
			[{"x": 0, "y": 7, "c": 100}]
		]
	}`)
	verr := ValidatePixelBounds(anim)
	if verr == nil {
		t.Fatal("expected ValidationError for y=7 with height=5, got nil")
	}
	if got, want := verr.Field, "frames[0].pixels[0].y"; got != want {
		t.Errorf("Field = %q, want %q", got, want)
	}
	if !strings.Contains(verr.Reason, "height") {
		t.Errorf("expected reason to reference height bound; got: %s", verr.Reason)
	}
}

// TestValidatePixelBounds_EdgeOfRange proves the maximum-allowed pixel
// (x=width-1, y=height-1) is accepted — i.e. the bounds checks are not
// off-by-one.
func TestValidatePixelBounds_EdgeOfRange(t *testing.T) {
	anim := mustDecodeSparse(t, `{
		"version": "1.0",
		"name": "Edge",
		"width": 20,
		"height": 5,
		"frames": [
			[{"x": 19, "y": 4, "c": 16777215}]
		]
	}`)
	if verr := ValidatePixelBounds(anim); verr != nil {
		t.Fatalf("unexpected ValidationError on edge-of-range payload: %v", verr)
	}
}
