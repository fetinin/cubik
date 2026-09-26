package main

import (
	"cubik/api"
	"fmt"
)

// ImportPayloadMaxBytes is the size cap (1 MiB) for an animation import
// payload, enforced at the HTTP layer by importBodyLimitMiddleware.
const ImportPayloadMaxBytes = 1 << 20

// ValidationError is the user-facing validation failure type. Field is a
// dotted/indexed path to the offending value (e.g. "frames[0].pixels[2].x").
// Reason is a sanitized human-readable explanation that MUST NOT include
// raw payload bytes, file paths, or other server-internal details.
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

// ValidatePixelBounds enforces pixel.X < anim.Width and pixel.Y < anim.Height.
// ogen's Validate() only confirms 0..254 (the schema-level upper bound on
// SparsePixel.x/y), so we re-check against the animation's declared dims.
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
