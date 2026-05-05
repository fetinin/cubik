package main

import (
	"cubik/api"
	"encoding/json"
	"errors"
	"fmt"

	"github.com/ogen-go/ogen/validate"
)

// ImportPayloadMaxBytes is the size cap (1 MiB) for an animation import
// payload. Callers SHOULD enforce this at the HTTP layer using
// [net/http.MaxBytesReader] before calling ValidateImport — the validator
// re-checks defensively for non-HTTP callers (tests, internal use).
const ImportPayloadMaxBytes = 1 << 20

// ValidationError is the user-facing validation failure type. Field is a
// best-effort dotted/indexed path (e.g. "frames[0].pixels[2].x"); it may be
// empty when the failure is structural (oversize payload, malformed JSON).
// Reason is a sanitized human-readable explanation that MUST NOT include
// raw payload bytes, file paths, or other server-internal details.
//
// Design note: the validator returns a single *ValidationError on the first
// failure rather than collecting every problem. Multi-error reporting is
// nicer UX but requires more plumbing through ogen's nested validate.Error
// shape; defer that to a follow-up if/when product asks for it.
type ValidationError struct {
	Field  string
	Reason string
}

func (e *ValidationError) Error() string {
	if e.Field == "" {
		return e.Reason
	}
	return e.Field + ": " + e.Reason
}

// ValidateImport runs the input-validation gate that sits in front of the
// codec on every animation import. Steps:
//
//  1. Size cap (belt-and-suspenders for non-HTTP callers).
//  2. JSON shape via ogen's Decode (custom UnmarshalJSON).
//  3. Schema validation via the generated SparseAnimation.Validate().
//  4. Cross-field bounds: pixel x/y vs the parsed Width/Height (ogen only
//     validates pixel coords against the schema-level 0..254 cap, not
//     against the animation's own declared dimensions).
//
// Version-major compatibility is intentionally NOT checked here — the codec
// owns that downstream and returns *UnknownMajorVersionError on mismatch.
//
// On success returns the decoded animation and nil. On failure returns a
// *ValidationError with the first problem encountered.
func ValidateImport(payload []byte) (*api.SparseAnimation, *ValidationError) {
	if len(payload) > ImportPayloadMaxBytes {
		return nil, &ValidationError{
			Reason: fmt.Sprintf("payload exceeds size cap of %d bytes", ImportPayloadMaxBytes),
		}
	}

	var anim api.SparseAnimation
	if err := json.Unmarshal(payload, &anim); err != nil {
		// json/ogen decode errors can include offsets and field names but
		// never echo payload bytes; safe to surface verbatim.
		field, reason := translateOgenError(err)
		if field == "" {
			return nil, &ValidationError{Reason: "invalid JSON: " + reason}
		}
		return nil, &ValidationError{Field: field, Reason: reason}
	}

	if err := anim.Validate(); err != nil {
		field, reason := translateOgenError(err)
		return nil, &ValidationError{Field: field, Reason: reason}
	}

	if verr := ValidatePixelBounds(&anim); verr != nil {
		return nil, verr
	}

	return &anim, nil
}

// ValidatePixelBounds enforces pixel.X < anim.Width and pixel.Y < anim.Height.
// ogen's Validate() only confirms 0..254 (the schema-level upper bound on
// SparsePixel.x/y), so we re-check against the animation's declared dims.
// Exported so handlers that already have an ogen-decoded SparseAnimation can
// run the cross-field bounds check without repeating the JSON decode.
func ValidatePixelBounds(anim *api.SparseAnimation) *ValidationError {
	for fi, frame := range anim.Frames {
		for pi, px := range frame {
			if px.X >= anim.Width {
				return &ValidationError{
					Field:  fmt.Sprintf("frames[%d].pixels[%d].x", fi, pi),
					Reason: fmt.Sprintf("x %d out of range for declared width %d", px.X, anim.Width),
				}
			}
			if px.Y >= anim.Height {
				return &ValidationError{
					Field:  fmt.Sprintf("frames[%d].pixels[%d].y", fi, pi),
					Reason: fmt.Sprintf("y %d out of range for declared height %d", px.Y, anim.Height),
				}
			}
		}
	}
	return nil
}

// translateOgenError walks an ogen validate.Error tree to extract the
// deepest field path and a sanitized reason. ogen wraps nested failures in
// validate.Error{Fields: []FieldError{Name, Error}}, where Error may itself
// be wrapped via go-faster/errors.Wrap or another *validate.Error. We unwrap
// until we hit a leaf. If the error is not a *validate.Error at all (e.g. a
// raw JSON syntax error), the returned path is empty and the reason is the
// bare error string.
func translateOgenError(err error) (string, string) {
	const maxDepth = 32
	current := err
	path := ""
	for range maxDepth {
		var ve *validate.Error
		if !errors.As(current, &ve) || len(ve.Fields) == 0 {
			break
		}
		// First failure wins at every level — matches the rest of the
		// validator's first-failure semantics.
		f := ve.Fields[0]
		path = joinFieldPath(path, f.Name)
		current = f.Error
	}
	return path, current.Error()
}

// joinFieldPath concatenates a parent path with a child segment. Array
// indices arrive as "[N]" (no leading dot); object fields arrive as bare
// names and need a separating dot when there is already a parent.
func joinFieldPath(parent, child string) string {
	if parent == "" {
		return child
	}
	if len(child) > 0 && child[0] == '[' {
		return parent + child
	}
	return parent + "." + child
}
