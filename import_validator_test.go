package main

import (
	"fmt"
	"strings"
	"testing"
)

const validImportPayload = `{
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

// TestValidateImport_HappyPath confirms a well-formed payload decodes
// cleanly and the returned struct matches the source.
func TestValidateImport_HappyPath(t *testing.T) {
	anim, verr := ValidateImport([]byte(validImportPayload))
	if verr != nil {
		t.Fatalf("unexpected ValidationError: %v", verr)
	}
	if anim == nil {
		t.Fatal("expected non-nil SparseAnimation, got nil")
	}
	if anim.Version != "1.0" {
		t.Errorf("Version = %q, want \"1.0\"", anim.Version)
	}
	if anim.Width != 20 || anim.Height != 5 {
		t.Errorf("dims = %dx%d, want 20x5", anim.Width, anim.Height)
	}
	if len(anim.Frames) != 2 {
		t.Errorf("len(Frames) = %d, want 2", len(anim.Frames))
	}
}

// TestValidateImport_OversizePayload feeds bytes above the size cap and
// confirms the validator rejects them. The payload is deliberately
// unparseable JSON so a missed size check would fall through to the
// JSON-parse branch and produce a different error reason — that lets us
// assert the size check fires *first*.
func TestValidateImport_OversizePayload(t *testing.T) {
	junk := make([]byte, ImportPayloadMaxBytes+1)
	for i := range junk {
		junk[i] = 'x' // not JSON; would fail JSON parse if size check missed it
	}

	_, verr := ValidateImport(junk)
	if verr == nil {
		t.Fatal("expected ValidationError for oversize payload, got nil")
	}
	if !strings.Contains(verr.Reason, "size cap") {
		t.Errorf("expected reason to mention size cap; got: %s", verr.Reason)
	}
	if verr.Field != "" {
		t.Errorf("expected empty Field for size-cap failure, got %q", verr.Field)
	}
}

// TestValidateImport_MalformedJSON proves a syntactically broken payload is
// surfaced as a ValidationError without leaking raw payload bytes into the
// reason string.
func TestValidateImport_MalformedJSON(t *testing.T) {
	const secret = "SECRETMARKER12345"
	payload := []byte("{broken " + secret)

	_, verr := ValidateImport(payload)
	if verr == nil {
		t.Fatal("expected ValidationError for malformed JSON, got nil")
	}
	if strings.Contains(verr.Reason, secret) {
		t.Errorf("validator leaked raw payload bytes into reason: %s", verr.Reason)
	}
	if !strings.Contains(strings.ToLower(verr.Reason), "json") &&
		!strings.Contains(strings.ToLower(verr.Reason), "decode") &&
		!strings.Contains(strings.ToLower(verr.Reason), "invalid") {
		t.Errorf("expected reason to identify a JSON/decode problem; got: %s", verr.Reason)
	}
}

// TestValidateImport_MissingRequiredField confirms a payload that omits a
// required top-level field (here: "version") is rejected and the field name
// appears in the error.
func TestValidateImport_MissingRequiredField(t *testing.T) {
	const payload = `{
		"name": "No Version",
		"width": 20,
		"height": 5,
		"frames": [
			[{"x": 0, "y": 0, "c": 16711680}]
		]
	}`
	_, verr := ValidateImport([]byte(payload))
	if verr == nil {
		t.Fatal("expected ValidationError for missing 'version' field, got nil")
	}
	combined := verr.Field + " " + verr.Reason
	if !strings.Contains(strings.ToLower(combined), "version") {
		t.Errorf("expected error to mention 'version'; got field=%q reason=%q", verr.Field, verr.Reason)
	}
}

// TestValidateImport_VersionPatternViolation confirms the regex constraint
// on `version` (^[0-9]+\.[0-9]+$) fires through ogen's Validate() and is
// translated into a ValidationError carrying the field path.
func TestValidateImport_VersionPatternViolation(t *testing.T) {
	const payload = `{
		"version": "abc",
		"name": "Bad Version",
		"width": 20,
		"height": 5,
		"frames": [
			[{"x": 0, "y": 0, "c": 16711680}]
		]
	}`
	_, verr := ValidateImport([]byte(payload))
	if verr == nil {
		t.Fatal("expected ValidationError for version=\"abc\", got nil")
	}
	if !strings.Contains(verr.Field, "version") {
		t.Errorf("expected Field to contain 'version'; got %q (reason=%q)", verr.Field, verr.Reason)
	}
}

// TestValidateImport_PixelXOutOfDeclaredDims confirms the cross-field
// bounds check catches an x coordinate that fits the schema-level 0..254
// cap but exceeds the animation's declared width.
func TestValidateImport_PixelXOutOfDeclaredDims(t *testing.T) {
	const payload = `{
		"version": "1.0",
		"name": "Bad X",
		"width": 10,
		"height": 5,
		"frames": [
			[{"x": 12, "y": 0, "c": 100}]
		]
	}`
	_, verr := ValidateImport([]byte(payload))
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

// TestValidateImport_PixelYOutOfDeclaredDims is the symmetric check for y.
func TestValidateImport_PixelYOutOfDeclaredDims(t *testing.T) {
	const payload = `{
		"version": "1.0",
		"name": "Bad Y",
		"width": 10,
		"height": 5,
		"frames": [
			[{"x": 0, "y": 7, "c": 100}]
		]
	}`
	_, verr := ValidateImport([]byte(payload))
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

// TestValidateImport_EdgeOfRangeHappyPath proves the maximum-allowed pixel
// (x=width-1, y=height-1, c=0xFFFFFF) is accepted — i.e. the bounds checks
// use strict-greater-than-or-equal comparisons against the dims, not
// off-by-one.
func TestValidateImport_EdgeOfRangeHappyPath(t *testing.T) {
	payload := `{
		"version": "1.0",
		"name": "Edge",
		"width": 20,
		"height": 5,
		"frames": [
			[{"x": 19, "y": 4, "c": 16777215}]
		]
	}`
	anim, verr := ValidateImport([]byte(payload))
	if verr != nil {
		t.Fatalf("unexpected ValidationError on edge-of-range payload: %v", verr)
	}
	if anim == nil {
		t.Fatal("expected non-nil SparseAnimation on edge-of-range payload")
	}
}

// TestValidateImport_ColorOutOfRange confirms ogen's `c` 0..16777215 cap is
// surfaced through our ValidationError shape (not as an unwrapped ogen
// error), even though the cross-field check itself doesn't enforce color.
func TestValidateImport_ColorOutOfRange(t *testing.T) {
	const payload = `{
		"version": "1.0",
		"name": "Bad Color",
		"width": 20,
		"height": 5,
		"frames": [
			[{"x": 0, "y": 0, "c": 16777216}]
		]
	}`
	_, verr := ValidateImport([]byte(payload))
	if verr == nil {
		t.Fatal("expected ValidationError for c=16777216, got nil")
	}
	if !strings.Contains(verr.Field, "c") {
		t.Errorf("expected Field to reference 'c'; got %q (reason=%q)", verr.Field, verr.Reason)
	}
}

// Extra: feed a payload with both a structural problem (oversize) and
// nominal JSON validity, and confirm the size check fires before parsing.
// This is paranoia for the size-cap belt-and-suspenders contract.
func TestValidateImport_SizeCapBeforeParse(t *testing.T) {
	// Build a syntactically valid JSON object padded to just over the cap.
	pad := strings.Repeat("a", ImportPayloadMaxBytes)
	payload := fmt.Sprintf(`{"name": "%s"}`, pad)
	if len(payload) <= ImportPayloadMaxBytes {
		t.Fatalf("test setup: payload not over cap (len=%d cap=%d)", len(payload), ImportPayloadMaxBytes)
	}

	_, verr := ValidateImport([]byte(payload))
	if verr == nil {
		t.Fatal("expected ValidationError, got nil")
	}
	if !strings.Contains(verr.Reason, "size cap") {
		t.Errorf("expected size-cap reason; got %q", verr.Reason)
	}
}
