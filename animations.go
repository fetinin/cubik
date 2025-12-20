package main

import (
	"strings"
)

// Matrix LED Control Functions - Patterns and Animations

// createSolidColor generates base64-encoded RGB data for a solid color pattern.
// All LEDs will be set to the same color specified by (r, g, b).
// numLEDs specifies the total number of LEDs (100 for a 20x5 matrix).
// Returns a string of length numLEDs * 4 characters.
func createSolidColor(r, g, b uint8, numLEDs int) string {
	// Encode single pixel
	encoded := encodeRGBColor(r, g, b)

	// Use strings.Builder for efficient concatenation
	var builder strings.Builder
	builder.Grow(numLEDs * 4) // Pre-allocate capacity

	// Repeat for all LEDs
	for i := 0; i < numLEDs; i++ {
		builder.WriteString(encoded)
	}

	return builder.String()
}

// createCheckerboard creates a checkerboard pattern alternating red and blue.
// The pattern alternates based on LED position in a row-major order.
// For a 20x5 matrix (100 LEDs), this creates an alternating pattern across all LEDs.
// Returns base64-encoded RGB data for numLEDs.
func createCheckerboard(numLEDs int) string {
	// Pre-encode colors
	red := encodeRGBColor(255, 0, 0)
	blue := encodeRGBColor(0, 0, 255)

	var builder strings.Builder
	builder.Grow(numLEDs * 4)

	for i := 0; i < numLEDs; i++ {
		if i%2 == 0 {
			builder.WriteString(red)
		} else {
			builder.WriteString(blue)
		}
	}

	return builder.String()
}

// createGradient creates a rainbow gradient across all LEDs.
// The gradient transitions through red → yellow → green → cyan → blue → magenta → red.
// Each LED gets a color based on its position, creating a smooth rainbow effect.
// For a 20x5 matrix, the gradient flows across all 100 LEDs in row-major order.
func createGradient(numLEDs int) string {
	var builder strings.Builder
	builder.Grow(numLEDs * 4)

	for i := 0; i < numLEDs; i++ {
		// Calculate position in gradient (0.0 to 1.0)
		pos := float64(i) / float64(numLEDs)

		var r, g, b uint8

		// Rainbow gradient using piecewise linear interpolation
		if pos < 0.167 { // Red to Yellow
			r = 255
			g = uint8(pos / 0.167 * 255)
			b = 0
		} else if pos < 0.333 { // Yellow to Green
			r = uint8((0.333 - pos) / 0.166 * 255)
			g = 255
			b = 0
		} else if pos < 0.5 { // Green to Cyan
			r = 0
			g = 255
			b = uint8((pos - 0.333) / 0.167 * 255)
		} else if pos < 0.667 { // Cyan to Blue
			r = 0
			g = uint8((0.667 - pos) / 0.167 * 255)
			b = 255
		} else if pos < 0.833 { // Blue to Magenta
			r = uint8((pos - 0.667) / 0.166 * 255)
			g = 0
			b = 255
		} else { // Magenta to Red
			r = 255
			g = 0
			b = uint8((1.0 - pos) / 0.167 * 255)
		}

		builder.WriteString(encodeRGBColor(r, g, b))
	}

	return builder.String()
}

