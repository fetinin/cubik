package main

import (
	"errors"
	"fmt"
	"strconv"
	"strings"
)

type Color struct {
	R, G, B uint8
}

type Framebuffer struct {
	Width  int
	Height int
	Pixels []Color
}

type Alignment int

const (
	AlignLeft Alignment = iota
	AlignCenter
	AlignRight
)

type DigitBitmap [5][5]bool

func NewFramebuffer(width, height int) *Framebuffer {
	return &Framebuffer{
		Width:  width,
		Height: height,
		Pixels: make([]Color, width*height),
	}
}

func (fb *Framebuffer) Clear(color Color) {
	for i := range fb.Pixels {
		fb.Pixels[i] = color
	}
}

func (fb *Framebuffer) SetPixel(x, y int, color Color) error {
	if x < 0 || x >= fb.Width || y < 0 || y >= fb.Height {
		return fmt.Errorf("pixel coordinates (%d, %d) out of bounds", x, y)
	}
	fb.Pixels[y*fb.Width+x] = color
	return nil
}

func (fb *Framebuffer) GetPixel(x, y int) (Color, error) {
	if x < 0 || x >= fb.Width || y < 0 || y >= fb.Height {
		return Color{}, fmt.Errorf("pixel coordinates (%d, %d) out of bounds", x, y)
	}
	return fb.Pixels[y*fb.Width+x], nil
}

// Encode converts the framebuffer to base64-encoded RGB string for UpdateLeds.
// X-axis is reversed because hardware addresses LEDs right-to-left.
func (fb *Framebuffer) Encode() string {
	var builder strings.Builder
	builder.Grow(len(fb.Pixels) * 4)

	for y := range fb.Height {
		for x := fb.Width - 1; x >= 0; x-- {
			pixel := fb.Pixels[y*fb.Width+x]
			builder.WriteString(encodeRGBColor(pixel.R, pixel.G, pixel.B))
		}
	}

	return builder.String()
}

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

func DrawDigit(fb *Framebuffer, digit rune, x, y int, color, background Color) error {
	bitmap, exists := digitFont[digit]
	if !exists {
		return fmt.Errorf("invalid digit character: '%c'", digit)
	}

	if x+5 > fb.Width || y+5 > fb.Height || x < 0 || y < 0 {
		return fmt.Errorf("digit at position (%d, %d) exceeds bounds", x, y)
	}

	for row := range 5 {
		for col := range 5 {
			pixelColor := background
			if bitmap[row][col] {
				pixelColor = color
			}
			if err := fb.SetPixel(x+col, y+row, pixelColor); err != nil {
				return fmt.Errorf("failed to set pixel: %w", err)
			}
		}
	}

	return nil
}

func DrawNumber(fb *Framebuffer, number int, y, spacing int, alignment Alignment, color, background Color) error {
	return DrawString(fb, strconv.Itoa(number), y, spacing, alignment, color, background)
}

func DrawString(fb *Framebuffer, str string, y, spacing int, alignment Alignment, color, background Color) error {
	if len(str) == 0 {
		return errors.New("cannot draw empty string")
	}

	totalWidth := len(str)*5 + (len(str)-1)*spacing
	if totalWidth > fb.Width {
		return fmt.Errorf("string '%s' too wide: needs %d pixels", str, totalWidth)
	}

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

	currentX := startX
	for _, digit := range str {
		if err := DrawDigit(fb, digit, currentX, y, color, background); err != nil {
			return fmt.Errorf("failed to draw digit '%c': %w", digit, err)
		}
		currentX += 5 + spacing
	}

	return nil
}
