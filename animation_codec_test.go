package main

import (
	"context"
	"cubik/api"
	"encoding/json"
	"errors"
	"reflect"
	"strings"
	"testing"
)

// makeFrame builds a flat width*height frame of black pixels, then applies
// the supplied (x, y, color) overrides.
type pixelOverride struct {
	x, y int
	c    Color
}

func makeFrame(width, height int, overrides ...pixelOverride) []Color {
	frame := make([]Color, width*height)
	for _, o := range overrides {
		frame[o.y*width+o.x] = o.c
	}
	return frame
}

// assertRoundTrip encodes then decodes the supplied frames and verifies the
// result is deep-equal to the original.
func assertRoundTrip(t *testing.T, name string, w, h int, frames [][]Color) {
	t.Helper()
	orig := &SavedAnimation{Name: "Round Trip " + name, Frames: frames}

	encoded, err := EncodeAnimation(orig, w, h)
	if err != nil {
		t.Fatalf("EncodeAnimation failed: %v", err)
	}
	if encoded.Version != CURRENT_FORMAT_VERSION {
		t.Errorf("Version = %q, want %q", encoded.Version, CURRENT_FORMAT_VERSION)
	}
	if int(encoded.Width) != w || int(encoded.Height) != h {
		t.Errorf("dims = %dx%d, want %dx%d", encoded.Width, encoded.Height, w, h)
	}
	if encoded.Name != orig.Name {
		t.Errorf("Name = %q, want %q", encoded.Name, orig.Name)
	}

	decoded, err := DecodeAnimation(encoded)
	if err != nil {
		t.Fatalf("DecodeAnimation failed: %v", err)
	}
	if decoded.Name != orig.Name {
		t.Errorf("decoded.Name = %q, want %q", decoded.Name, orig.Name)
	}
	if !reflect.DeepEqual(decoded.Frames, orig.Frames) {
		t.Errorf("frames mismatch after round-trip\n got: %#v\nwant: %#v", decoded.Frames, orig.Frames)
	}
}

// TestEncodeDecode_RoundTrip is the table-driven proof for R-roundtrip:
// for several sample animations EncodeAnimation followed by DecodeAnimation
// returns frames deep-equal to the original.
func TestEncodeDecode_RoundTrip(t *testing.T) {
	const w, h = 20, 5
	dense := make([]Color, w*h)
	for i := range dense {
		dense[i] = Color{R: uint8(i + 1), G: uint8(i + 2), B: uint8(i + 3)}
	}

	cases := []struct {
		name   string
		w, h   int
		frames [][]Color
	}{
		{
			name: "single sparse frame", w: w, h: h,
			frames: [][]Color{
				makeFrame(w, h,
					pixelOverride{x: 0, y: 0, c: Color{R: 255}},
					pixelOverride{x: 19, y: 4, c: Color{B: 255}},
				),
			},
		},
		{
			name: "all-black frame stays black", w: w, h: h,
			frames: [][]Color{makeFrame(w, h)},
		},
		{
			name: "mixed frames including all-black", w: w, h: h,
			frames: [][]Color{
				makeFrame(w, h, pixelOverride{x: 5, y: 2, c: Color{R: 10, G: 20, B: 30}}),
				makeFrame(w, h),
				makeFrame(w, h,
					pixelOverride{x: 0, y: 0, c: Color{R: 255, G: 255, B: 255}},
					pixelOverride{x: 9, y: 2, c: Color{G: 255}},
					pixelOverride{x: 19, y: 4, c: Color{R: 1, G: 2, B: 3}},
				),
			},
		},
		{
			name: "dense (every pixel lit)", w: w, h: h,
			frames: [][]Color{dense},
		},
		{
			// Different dimensions exercises the index math.
			name: "non-square 8x3 with two frames", w: 8, h: 3,
			frames: [][]Color{
				makeFrame(8, 3, pixelOverride{x: 7, y: 2, c: Color{G: 128}}),
				makeFrame(8, 3, pixelOverride{x: 0, y: 0, c: Color{R: 64}}),
			},
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			assertRoundTrip(t, tc.name, tc.w, tc.h, tc.frames)
		})
	}
}

// TestDecodeAnimation_RejectUnknownMajor covers R-S4b-major-reject:
// a payload with version "2.0" must be rejected with *UnknownMajorVersionError
// retrievable via [errors.As], and the error message must mention the version.
func TestDecodeAnimation_RejectUnknownMajor(t *testing.T) {
	payload := api.SparseAnimation{
		Version: "2.0",
		Name:    "Future",
		Width:   20,
		Height:  5,
		Frames: []api.SparseFrame{
			{{X: 0, Y: 0, C: 0xFF0000}},
		},
	}

	_, err := DecodeAnimation(payload)
	if err == nil {
		t.Fatalf("DecodeAnimation = nil error, want UnknownMajorVersionError")
	}
	var typed *UnknownMajorVersionError
	if !errors.As(err, &typed) {
		t.Fatalf("errors.As failed: got %T (%v), want *UnknownMajorVersionError", err, err)
	}
	if typed.GotMajor != 2 {
		t.Errorf("GotMajor = %d, want 2", typed.GotMajor)
	}
	if typed.WantMajor != CURRENT_FORMAT_MAJOR {
		t.Errorf("WantMajor = %d, want %d", typed.WantMajor, CURRENT_FORMAT_MAJOR)
	}
	// Sanity: error message includes the unsupported major number.
	if msg := typed.Error(); msg == "" || !strings.Contains(msg, "2") {
		t.Errorf("Error() = %q, expected to contain version number", msg)
	}
}

// TestDecodeAnimation_AcceptUnknownFieldsSameMajor covers R-S4c-minor-additive:
// decoding a JSON payload with version "1.1" plus an extra unknown field on
// both the envelope and a pixel succeeds; the resulting SavedAnimation has
// only known fields populated.
func TestDecodeAnimation_AcceptUnknownFieldsSameMajor(t *testing.T) {
	const raw = `{
		"version": "1.1",
		"name": "Future Minor",
		"width": 4,
		"height": 2,
		"some_future_envelope_field": {"a": 1, "b": [1,2,3]},
		"frames": [
			[
				{"x": 0, "y": 0, "c": 16711680, "future_pixel_field": "ignore me"},
				{"x": 3, "y": 1, "c": 255}
			]
		]
	}`

	var payload api.SparseAnimation
	if err := json.Unmarshal([]byte(raw), &payload); err != nil {
		t.Fatalf("json.Unmarshal failed: %v", err)
	}

	decoded, err := DecodeAnimation(payload)
	if err != nil {
		t.Fatalf("DecodeAnimation failed: %v", err)
	}
	if decoded.Name != "Future Minor" {
		t.Errorf("Name = %q, want %q", decoded.Name, "Future Minor")
	}
	if len(decoded.Frames) != 1 {
		t.Fatalf("len(Frames) = %d, want 1", len(decoded.Frames))
	}
	frame := decoded.Frames[0]
	if len(frame) != 4*2 {
		t.Fatalf("len(frame) = %d, want %d", len(frame), 4*2)
	}
	// y=0, x=0 -> red
	if got, want := frame[0], (Color{R: 0xFF}); got != want {
		t.Errorf("frame[0] = %+v, want %+v", got, want)
	}
	// y=1, x=3 -> blue, index = 1*4+3 = 7
	if got, want := frame[7], (Color{B: 0xFF}); got != want {
		t.Errorf("frame[7] = %+v, want %+v", got, want)
	}
	// Spot-check that other slots are zero (unspecified -> black).
	for i, c := range frame {
		if i == 0 || i == 7 {
			continue
		}
		if c != (Color{}) {
			t.Errorf("frame[%d] = %+v, want zero", i, c)
		}
	}
}

// TestDecodeAnimation_MalformedVersion exercises the defensive guard that
// runs even if the ogen pattern validator hasn't.
func TestDecodeAnimation_MalformedVersion(t *testing.T) {
	payload := api.SparseAnimation{
		Version: "abc",
		Name:    "Bad",
		Width:   2, Height: 2,
		Frames: []api.SparseFrame{{}},
	}
	_, err := DecodeAnimation(payload)
	if err == nil {
		t.Fatalf("expected error for malformed version, got nil")
	}
	var mve *MalformedVersionError
	if !errors.As(err, &mve) {
		t.Fatalf("errors.As failed: got %T (%v), want *MalformedVersionError", err, err)
	}
}

// TestPersistImported_NoCollision verifies a fresh import inserts cleanly.
func TestPersistImported_NoCollision(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()
	ctx := context.Background()

	anim := &SavedAnimation{
		Name:   "fresh",
		Frames: [][]Color{{{R: 1}, {G: 2}}},
	}
	got, err := PersistImported(ctx, db, "device-1", anim, ImportModeRename)
	if err != nil {
		t.Fatalf("PersistImported failed: %v", err)
	}
	if got.ID == "" {
		t.Errorf("expected non-empty ID")
	}
	if got.Name != "fresh" {
		t.Errorf("Name = %q, want %q", got.Name, "fresh")
	}
}

// TestPersistImported_Rename verifies the suffix flow on collision.
func TestPersistImported_Rename(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()
	ctx := context.Background()
	device := "device-1"

	// Seed an existing animation called "Wave".
	if _, err := SaveAnimation(ctx, db, device, "Wave", [][]Color{{{R: 1}}}); err != nil {
		t.Fatalf("SaveAnimation seed failed: %v", err)
	}

	anim := &SavedAnimation{Name: "Wave", Frames: [][]Color{{{G: 99}}}}
	got, err := PersistImported(ctx, db, device, anim, ImportModeRename)
	if err != nil {
		t.Fatalf("PersistImported failed: %v", err)
	}
	if got.Name != "Wave (2)" {
		t.Errorf("Name = %q, want %q", got.Name, "Wave (2)")
	}

	// Importing again should bump to (3).
	got2, err := PersistImported(ctx, db, device, anim, ImportModeRename)
	if err != nil {
		t.Fatalf("PersistImported second call failed: %v", err)
	}
	if got2.Name != "Wave (3)" {
		t.Errorf("second Name = %q, want %q", got2.Name, "Wave (3)")
	}
}

// TestPersistImported_Overwrite verifies UPDATE-in-place by name.
func TestPersistImported_Overwrite(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()
	ctx := context.Background()
	device := "device-1"

	original, err := SaveAnimation(ctx, db, device, "Wave", [][]Color{{{R: 1}}})
	if err != nil {
		t.Fatalf("SaveAnimation seed failed: %v", err)
	}

	newFrames := [][]Color{{{B: 200}}, {{G: 50}}}
	anim := &SavedAnimation{Name: "Wave", Frames: newFrames}
	got, err := PersistImported(ctx, db, device, anim, ImportModeOverwrite)
	if err != nil {
		t.Fatalf("PersistImported failed: %v", err)
	}
	if got.ID != original.ID {
		t.Errorf("ID changed on overwrite: got %q, want %q", got.ID, original.ID)
	}
	if got.Name != "Wave" {
		t.Errorf("Name = %q, want %q", got.Name, "Wave")
	}
	if !reflect.DeepEqual(got.Frames, newFrames) {
		t.Errorf("frames not updated: got %#v, want %#v", got.Frames, newFrames)
	}

	// Confirm there's still only one row for the device (no rename).
	all, err := ListAnimationsByDevice(ctx, db, device)
	if err != nil {
		t.Fatalf("ListAnimationsByDevice failed: %v", err)
	}
	if len(all) != 1 {
		t.Errorf("expected 1 animation for device, got %d", len(all))
	}
}

// TestPersistImported_Cancel verifies that ImportModeCancel returns a typed
// error (retrievable via [errors.As]) and leaves the DB untouched.
func TestPersistImported_Cancel(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()
	ctx := context.Background()
	device := "device-1"

	existing, err := SaveAnimation(ctx, db, device, "Wave", [][]Color{{{R: 1}}})
	if err != nil {
		t.Fatalf("SaveAnimation seed failed: %v", err)
	}

	anim := &SavedAnimation{Name: "Wave", Frames: [][]Color{{{B: 200}}}}
	_, err = PersistImported(ctx, db, device, anim, ImportModeCancel)
	if err == nil {
		t.Fatalf("expected NameConflictError, got nil")
	}
	var nce *NameConflictError
	if !errors.As(err, &nce) {
		t.Fatalf("errors.As failed: got %T (%v), want *NameConflictError", err, err)
	}
	if nce.Name != "Wave" {
		t.Errorf("Name = %q, want %q", nce.Name, "Wave")
	}
	if nce.ExistingID != existing.ID {
		t.Errorf("ExistingID = %q, want %q", nce.ExistingID, existing.ID)
	}

	// DB should still hold exactly the original record, unchanged.
	all, err := ListAnimationsByDevice(ctx, db, device)
	if err != nil {
		t.Fatalf("ListAnimationsByDevice failed: %v", err)
	}
	if len(all) != 1 {
		t.Errorf("expected 1 animation for device, got %d", len(all))
	}
	if all[0].ID != existing.ID {
		t.Errorf("ID = %q, want %q (unchanged)", all[0].ID, existing.ID)
	}
}

// TestEncodeAnimation_RejectsBadDimensions ensures we don't silently pad/truncate.
func TestEncodeAnimation_RejectsBadDimensions(t *testing.T) {
	saved := &SavedAnimation{
		Name:   "x",
		Frames: [][]Color{{{R: 1}}}, // 1 pixel, but we'll claim 20x5 = 100
	}
	if _, err := EncodeAnimation(saved, 20, 5); err == nil {
		t.Errorf("expected encode error on dimension mismatch, got nil")
	}
}
