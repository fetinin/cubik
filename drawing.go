package main

import (
	"fmt"
	"strings"
)

// Matrix Display - Digit Rendering System

// Color represents an RGB color value
type Color struct {
	R, G, B uint8
}

// Framebuffer represents the LED matrix state before encoding
type Framebuffer struct {
	Width  int     // 20 for CubeLite
	Height int     // 5 for CubeLite
	Pixels []Color // Length = Width * Height (100)
}

// Alignment specifies horizontal positioning for multi-digit numbers
type Alignment int

const (
	AlignLeft Alignment = iota
	AlignCenter
	AlignRight
)

// DigitBitmap represents a 5x5 pixel font for a single digit
// [row][column] where row=0 is top, column=0 is left
type DigitBitmap [5][5]bool

// NewFramebuffer creates a new framebuffer with all pixels set to black (off)
func NewFramebuffer(width, height int) *Framebuffer {
	fb := &Framebuffer{
		Width:  width,
		Height: height,
		Pixels: make([]Color, width*height),
	}
	// Pixels are already initialized to zero values (black)
	return fb
}

// Clear sets all pixels to the specified color
func (fb *Framebuffer) Clear(color Color) {
	for i := range fb.Pixels {
		fb.Pixels[i] = color
	}
}

// SetPixel sets a single pixel at (x, y) to the specified color
// Returns error if coordinates are out of bounds
func (fb *Framebuffer) SetPixel(x, y int, color Color) error {
	if x < 0 || x >= fb.Width || y < 0 || y >= fb.Height {
		return fmt.Errorf("pixel coordinates (%d, %d) out of bounds (width=%d, height=%d)", x, y, fb.Width, fb.Height)
	}
	index := y*fb.Width + x
	fb.Pixels[index] = color
	return nil
}

// GetPixel returns the color at (x, y)
func (fb *Framebuffer) GetPixel(x, y int) (Color, error) {
	if x < 0 || x >= fb.Width || y < 0 || y >= fb.Height {
		return Color{}, fmt.Errorf("pixel coordinates (%d, %d) out of bounds (width=%d, height=%d)", x, y, fb.Width, fb.Height)
	}
	index := y*fb.Width + x
	return fb.Pixels[index], nil
}

// Encode converts the framebuffer to base64-encoded RGB string for UpdateLeds
// Returns 400-character string for 20x5 matrix (100 LEDs * 4 chars each)
// Note: X-axis is reversed because hardware addresses LEDs right-to-left
func (fb *Framebuffer) Encode() string {
	var builder strings.Builder
	builder.Grow(len(fb.Pixels) * 4) // Pre-allocate: each LED = 4 base64 chars

	// Iterate through pixels in row-major order, but reverse X within each row
	// This compensates for the hardware addressing LEDs from right-to-left
	for y := 0; y < fb.Height; y++ {
		for x := fb.Width - 1; x >= 0; x-- {
			index := y*fb.Width + x
			pixel := fb.Pixels[index]
			encoded := encodeRGBColor(pixel.R, pixel.G, pixel.B)
			builder.WriteString(encoded)
		}
	}

	return builder.String()
}

// digitFont maps digit characters '0'-'9' to their 5x5 bitmap representations
// Each bitmap is organized as [row][column] where row=0 is top, column=0 is left
// true = pixel on (digit color), false = pixel off (background color)
var digitFont = map[rune]DigitBitmap{
	'0': {
		{true, true, true, true, true},
		{true, false, false, false, true},
		{true, false, false, false, true},
		{true, false, false, false, true},
		{true, true, true, true, true},
	},
	'1': {
		{false, false, true, false, false},
		{false, true, true, false, false},
		{false, false, true, false, false},
		{false, false, true, false, false},
		{false, true, true, true, false},
	},
	'2': {
		{true, true, true, true, true},
		{false, false, false, false, true},
		{true, true, true, true, true},
		{true, false, false, false, false},
		{true, true, true, true, true},
	},
	'3': {
		{true, true, true, true, true},
		{false, false, false, false, true},
		{true, true, true, true, true},
		{false, false, false, false, true},
		{true, true, true, true, true},
	},
	'4': {
		{true, false, false, false, true},
		{true, false, false, false, true},
		{true, true, true, true, true},
		{false, false, false, false, true},
		{false, false, false, false, true},
	},
	'5': {
		{true, true, true, true, true},
		{true, false, false, false, false},
		{true, true, true, true, true},
		{false, false, false, false, true},
		{true, true, true, true, true},
	},
	'6': {
		{true, true, true, true, true},
		{true, false, false, false, false},
		{true, true, true, true, true},
		{true, false, false, false, true},
		{true, true, true, true, true},
	},
	'7': {
		{true, true, true, true, true},
		{false, false, false, false, true},
		{false, false, false, true, false},
		{false, false, true, false, false},
		{false, true, false, false, false},
	},
	'8': {
		{true, true, true, true, true},
		{true, false, false, false, true},
		{true, true, true, true, true},
		{true, false, false, false, true},
		{true, true, true, true, true},
	},
	'9': {
		{true, true, true, true, true},
		{true, false, false, false, true},
		{true, true, true, true, true},
		{false, false, false, false, true},
		{true, true, true, true, true},
	},
}

// DrawDigit renders a single digit at the specified position
// Parameters:
//   - fb: target framebuffer
//   - digit: character '0'-'9'
//   - x, y: top-left corner position in framebuffer
//   - color: color for the digit pixels (lit pixels)
//   - background: color for non-digit pixels (typically black/off)
//
// Returns error if digit is invalid or position is out of bounds
func DrawDigit(fb *Framebuffer, digit rune, x, y int, color, background Color) error {
	// Look up digit bitmap
	bitmap, exists := digitFont[digit]
	if !exists {
		return fmt.Errorf("invalid digit character: '%c' (only '0'-'9' supported)", digit)
	}

	// Check bounds (digit is 5x5)
	if x+5 > fb.Width || y+5 > fb.Height {
		return fmt.Errorf("digit at position (%d, %d) would exceed framebuffer bounds (width=%d, height=%d)", x, y, fb.Width, fb.Height)
	}
	if x < 0 || y < 0 {
		return fmt.Errorf("digit position (%d, %d) cannot be negative", x, y)
	}

	// Iterate through 5x5 bitmap and set pixels
	for row := 0; row < 5; row++ {
		for col := 0; col < 5; col++ {
			pixelColor := background
			if bitmap[row][col] {
				pixelColor = color
			}
			fb.SetPixel(x+col, y+row, pixelColor)
		}
	}

	return nil
}

// DrawNumber renders a multi-digit number with alignment
// Parameters:
//   - fb: target framebuffer
//   - number: integer to display (e.g., 42, 123)
//   - y: vertical position (typically 0 for centered on 5-row display)
//   - spacing: pixels between digits (0 or 1 recommended)
//   - alignment: how to position the number horizontally (AlignLeft, AlignCenter, AlignRight)
//   - color: color for digit pixels
//   - background: color for background (black = off)
//
// Returns error if number doesn't fit or invalid parameters
func DrawNumber(fb *Framebuffer, number int, y, spacing int, alignment Alignment, color, background Color) error {
	// Convert number to string
	str := fmt.Sprintf("%d", number)
	return DrawString(fb, str, y, spacing, alignment, color, background)
}

// DrawString renders a string of digits with alignment
// Similar to DrawNumber but accepts string input for flexibility
// Parameters:
//   - fb: target framebuffer
//   - str: string of digits to display (e.g., "42", "123")
//   - y: vertical position (typically 0 for 5-row display)
//   - spacing: pixels between digits (0 or 1 recommended)
//   - alignment: how to position horizontally (AlignLeft, AlignCenter, AlignRight)
//   - color: color for digit pixels
//   - background: color for background (black = off)
//
// Returns error if string contains non-digit characters, doesn't fit, or invalid parameters
func DrawString(fb *Framebuffer, str string, y, spacing int, alignment Alignment, color, background Color) error {
	if len(str) == 0 {
		return fmt.Errorf("cannot draw empty string")
	}

	// Calculate total width needed
	// Each digit = 5 pixels, spacing between = spacing pixels
	// Total = (numDigits * 5) + ((numDigits - 1) * spacing)
	totalWidth := len(str)*5 + (len(str)-1)*spacing

	// Check if number fits in framebuffer
	if totalWidth > fb.Width {
		return fmt.Errorf("string '%s' too wide: needs %d pixels, framebuffer width is %d", str, totalWidth, fb.Width)
	}

	// Calculate starting X position based on alignment
	var startX int
	switch alignment {
	case AlignLeft:
		startX = 0
	case AlignCenter:
		startX = (fb.Width - totalWidth) / 2
	case AlignRight:
		startX = fb.Width - totalWidth
	default:
		return fmt.Errorf("invalid alignment value: %d", alignment)
	}

	// Draw each digit sequentially
	currentX := startX
	for _, digit := range str {
		err := DrawDigit(fb, digit, currentX, y, color, background)
		if err != nil {
			return fmt.Errorf("failed to draw digit '%c': %w", digit, err)
		}
		currentX += 5 + spacing // Move to next digit position
	}

	return nil
}
