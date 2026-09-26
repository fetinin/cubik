package main

import (
	"cubik/api"
	"encoding/json"
	"strings"
	"testing"
)

// TestSparseAnimation_DecodeRoundTrip verifies that a hand-written
// SparseAnimation JSON sample decodes into the ogen-generated api.SparseAnimation
// type with all fields preserved. This is the proof that the wire format is
// self-describing and round-trip-decodable from raw JSON
// (covers R-format-sparse + R-format-self-describing).
func TestSparseAnimation_DecodeRoundTrip(t *testing.T) {
	const payload = `{
		"version": "1.0",
		"name": "Tiny Sample",
		"width": 20,
		"height": 5,
		"frames": [
			[
				{"x": 0, "y": 0, "c": 16711680},
				{"x": 19, "y": 4, "c": 255}
			],
			[
				{"x": 9, "y": 2, "c": 65280}
			]
		]
	}`

	var anim api.SparseAnimation
	if err := json.Unmarshal([]byte(payload), &anim); err != nil {
		t.Fatalf("unexpected decode error: %v", err)
	}

	if got, want := anim.Version, "1.0"; got != want {
		t.Errorf("Version = %q, want %q", got, want)
	}
	if got, want := anim.Name, "Tiny Sample"; got != want {
		t.Errorf("Name = %q, want %q", got, want)
	}
	if got, want := anim.Width, int32(20); got != want {
		t.Errorf("Width = %d, want %d", got, want)
	}
	if got, want := anim.Height, int32(5); got != want {
		t.Errorf("Height = %d, want %d", got, want)
	}
	if got, want := len(anim.Frames), 2; got != want {
		t.Fatalf("len(Frames) = %d, want %d", got, want)
	}

	frame0 := anim.Frames[0]
	if got, want := len(frame0), 2; got != want {
		t.Fatalf("len(Frames[0]) = %d, want %d", got, want)
	}
	if px := frame0[0]; px.X != 0 || px.Y != 0 || px.C != 16711680 {
		t.Errorf("Frames[0][0] = (x=%d y=%d c=%d), want (0, 0, 16711680)",
			px.X, px.Y, px.C)
	}
	if px := frame0[1]; px.X != 19 || px.Y != 4 || px.C != 255 {
		t.Errorf("Frames[0][1] = (x=%d y=%d c=%d), want (19, 4, 255)",
			px.X, px.Y, px.C)
	}

	frame1 := anim.Frames[1]
	if got, want := len(frame1), 1; got != want {
		t.Fatalf("len(Frames[1]) = %d, want %d", got, want)
	}
	if px := frame1[0]; px.X != 9 || px.Y != 2 || px.C != 65280 {
		t.Errorf("Frames[1][0] = (x=%d y=%d c=%d), want (9, 2, 65280)",
			px.X, px.Y, px.C)
	}

	// Sanity check: validation against spec constraints should also pass for
	// a well-formed payload, ensuring the round-trip really is end-to-end clean.
	if err := anim.Validate(); err != nil {
		t.Errorf("unexpected Validate() error on well-formed payload: %v", err)
	}
}

// TestSparseAnimation_MissingVersion verifies that a payload without the
// required "version" field is rejected by the ogen-generated decoder
// (covers R-S4a-version-field).
func TestSparseAnimation_MissingVersion(t *testing.T) {
	const payload = `{
		"name": "No Version",
		"width": 20,
		"height": 5,
		"frames": [
			[{"x": 0, "y": 0, "c": 16711680}]
		]
	}`

	var anim api.SparseAnimation
	err := json.Unmarshal([]byte(payload), &anim)
	if err == nil {
		t.Fatalf("expected decode error for payload missing 'version', got nil")
	}
	// Sanity-check the error mentions "version" so we do not accidentally
	// pass on some unrelated decoding failure.
	if !strings.Contains(strings.ToLower(err.Error()), "version") {
		t.Errorf("expected decode error to mention 'version', got: %v", err)
	}
}

// TestSparseAnimation_PatternViolation verifies that the spec-declared regex
// constraint on the "version" field (^[0-9]+\.[0-9]+$) is enforced by the
// generated Validate() method.
//
// Note: ogen splits decoding from validation — UnmarshalJSON only checks
// shape and required-field presence; pattern/range/length constraints live
// in the generated Validate() method that the HTTP server invokes after
// decode. We therefore call Validate() explicitly here.
func TestSparseAnimation_PatternViolation(t *testing.T) {
	const payload = `{
		"version": "abc",
		"name": "Bad Version",
		"width": 20,
		"height": 5,
		"frames": [
			[{"x": 0, "y": 0, "c": 16711680}]
		]
	}`

	var anim api.SparseAnimation
	if err := json.Unmarshal([]byte(payload), &anim); err != nil {
		t.Fatalf("unexpected decode error (pattern is checked in Validate, not Unmarshal): %v", err)
	}

	err := anim.Validate()
	if err == nil {
		t.Fatalf("expected Validate() to reject version=\"abc\", got nil")
	}
	if !strings.Contains(strings.ToLower(err.Error()), "version") {
		t.Errorf("expected validation error to mention 'version', got: %v", err)
	}
}
