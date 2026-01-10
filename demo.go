//nolint:unused // demo code for future use
package main

import (
	"fmt"
	"math"
	"math/rand/v2"
	"strings"
	"time"
)

func demoTestPatterns(device *DeviceInfo) {
	for {
		patterns := []struct {
			name string
			data string
		}{
			{"Checkerboard", createCheckerboard(100)},
			{"Solid Red", createSolidColor(255, 0, 0, 100)},
			{"Checkerboard", createCheckerboard(100)},
			// {"Solid Green", createSolidColor(0, 255, 0, 100)},
			{"Solid Blue", createSolidColor(0, 0, 255, 100)},
			// {"Rainbow Gradient", createGradient(100)},
		}

		for _, pattern := range patterns {
			fmt.Printf("\n  Displaying: %s\n", pattern.name)
			// fmt.Print("  Press Enter to continue...")
			// bufio.NewReader(os.Stdin).ReadString('\n')
			time.Sleep(1*time.Second + 100*time.Millisecond)

			err := UpdateLeds(device, pattern.data)
			if err != nil {
				fmt.Printf("  Error updating LEDs: %v\n", err)
			} else {
				fmt.Println("  ✓ Pattern displayed")
			}
		}
	}
}

func demoDigitDisplay(device *DeviceInfo) {
	fb := NewFramebuffer(20, 5)
	black := Color{R: 0, G: 0, B: 0}

	// Demo 1: Single digits 0-9 in green
	green := Color{R: 0, G: 255, B: 0}
	fmt.Println("\n  Demo: Single digits 0-9")
	for i := range 10 {
		fb.Clear(black)
		if err := DrawNumber(fb, i, 0, 0, AlignCenter, green, black); err != nil {
			fmt.Printf("  Error drawing number: %v\n", err)
			continue
		}
		if err := UpdateLeds(device, fb.Encode()); err != nil {
			fmt.Printf("  Error updating LEDs: %v\n", err)
			continue
		}
		time.Sleep(500 * time.Millisecond)
	}

	// Demo 2: Multi-digit numbers in blue
	blue := Color{R: 0, G: 0, B: 255}
	numbers := []int{42, 123, 999, 2025}
	fmt.Println("\n  Demo: Multi-digit numbers")
	for _, num := range numbers {
		fb.Clear(black)
		if err := DrawNumber(fb, num, 0, 1, AlignCenter, blue, black); err != nil {
			fmt.Printf("  Error drawing number: %v\n", err)
			continue
		}
		if err := UpdateLeds(device, fb.Encode()); err != nil {
			fmt.Printf("  Error updating LEDs: %v\n", err)
			continue
		}
		time.Sleep(1 * time.Second)
	}

	// Demo 3: Countdown in red
	red := Color{R: 255, G: 0, B: 0}
	fmt.Println("\n  Demo: Countdown 10 to 0")
	for i := 10; i >= 0; i-- {
		fb.Clear(black)
		if err := DrawNumber(fb, i, 0, 1, AlignCenter, red, black); err != nil {
			fmt.Printf("  Error drawing number: %v\n", err)
			continue
		}
		if err := UpdateLeds(device, fb.Encode()); err != nil {
			fmt.Printf("  Error updating LEDs: %v\n", err)
			continue
		}
		time.Sleep(500 * time.Millisecond)
	}

	// Demo 4: Alignment showcase
	white := Color{R: 255, G: 255, B: 255}
	fmt.Println("\n  Demo: Alignment (42 left/center/right)")
	alignments := []Alignment{AlignLeft, AlignCenter, AlignRight}
	for _, align := range alignments {
		fb.Clear(black)
		if err := DrawNumber(fb, 42, 0, 1, align, white, black); err != nil {
			fmt.Printf("  Error drawing number: %v\n", err)
			continue
		}
		if err := UpdateLeds(device, fb.Encode()); err != nil {
			fmt.Printf("  Error updating LEDs: %v\n", err)
			continue
		}
		time.Sleep(1 * time.Second)
	}

	fmt.Println("\n  Digit display demo complete!")
}

// ==========================================
// Christmas Tree Animation System
// ==========================================

// Animation modes for Christmas tree lights
const (
	AnimationTwinkle = iota
	AnimationChase
	AnimationPulse
	AnimationSparkle
	AnimationAll // Cycles through all modes
)

type AnimationMode int

// LightState tracks the state of a single Christmas light
type LightState struct {
	X, Y           int   // Position on matrix
	On             bool  // Current on/off state
	Color          Color // Current color
	BaseColor      Color // Original color for pulse effect
	NextToggleTick uint  // For twinkle effect (tick-based)
	SparkleEndTick uint  // When sparkle effect ends (tick-based)
}

// ChristmasTreeAnimator manages the animation state
type ChristmasTreeAnimator struct {
	Lights            []LightState
	Tick              uint
	Mode              AnimationMode
	LastActiveMode    AnimationMode
	ModeDurationTicks uint
}

// Color palette for Christmas tree
var (
	TreeGreen   = Color{R: 0, G: 180, B: 0}
	StarYellow  = Color{R: 255, G: 220, B: 0}
	TrunkBrown  = Color{R: 139, G: 69, B: 19}
	GiftRed     = Color{R: 200, G: 0, B: 0}
	GiftBlue    = Color{R: 0, G: 100, B: 200}
	LightRed    = Color{R: 255, G: 0, B: 0}
	LightBlue   = Color{R: 0, G: 100, B: 255}
	LightYellow = Color{R: 255, G: 200, B: 0}
	LightWhite  = Color{R: 255, G: 255, B: 255}
	LightOff    = Color{R: 0, G: 60, B: 0}
)

// Light positions on the tree (10 lights total)
var lightPositions = []struct{ X, Y int }{
	{10, 0},         // Row 0: 1 light (top)
	{9, 1}, {11, 1}, // Row 1: 2 lights
	{8, 2}, {10, 2}, {12, 2}, // Row 2: 3 lights
	{7, 3}, {9, 3}, {11, 3}, {13, 3}, // Row 3: 4 lights
}

// Tree foliage coordinates by row
var treePixels = map[int][]int{
	0: {10},                      // Row 0: 1 pixel
	1: {9, 10, 11},               // Row 1: 3 pixels
	2: {8, 9, 10, 11, 12},        // Row 2: 5 pixels
	3: {7, 8, 9, 10, 11, 12, 13}, // Row 3: 7 pixels
}

// demoChristmasTree runs the Christmas tree animation demo
func demoChristmasTree(device *DeviceInfo) {
	fmt.Println("\n  Christmas Tree Animation Demo")
	fmt.Println("  =============================")

	animator := NewChristmasTreeAnimator(AnimationAll)
	fb := NewFramebuffer(20, 5)

	for {
		// Update animation state
		animator.Update()

		// Render tree
		drawChristmasTree(fb, &animator)

		// Send to device
		err := UpdateLeds(device, fb.Encode())
		if err != nil {
			fmt.Printf("  Error updating LEDs: %v\n", err)
			return
		}

		// Pace updates to avoid device rate limiting
		time.Sleep(1 * time.Second)
	}
}

// NewChristmasTreeAnimator creates a new animator with initialized lights
func NewChristmasTreeAnimator(mode AnimationMode) ChristmasTreeAnimator {
	animator := ChristmasTreeAnimator{
		Lights:            make([]LightState, len(lightPositions)),
		Tick:              0,
		Mode:              mode,
		LastActiveMode:    AnimationTwinkle,
		ModeDurationTicks: 100,
	}

	// Initialize light states with different colors
	colors := []Color{LightRed, LightBlue, LightYellow, LightWhite}
	for i, pos := range lightPositions {
		animator.Lights[i] = LightState{
			X:              pos.X,
			Y:              pos.Y,
			On:             true,
			Color:          colors[i%len(colors)],
			BaseColor:      colors[i%len(colors)],
			NextToggleTick: uint(rand.IntN(10)), // initial twinkle offset in ticks
		}
	}

	return animator
}

// Update advances the animation state
func (a *ChristmasTreeAnimator) Update() {
	// Determine active mode based on tick (no wall-clock time).
	activeMode := a.Mode
	if a.Mode == AnimationAll {
		activeMode = AnimationMode((a.Tick / a.ModeDurationTicks) % uint(4))
	}

	if activeMode != a.LastActiveMode {
		fmt.Printf("  Switching to animation mode: %d\n", activeMode)
		a.LastActiveMode = activeMode
	}

	// Update based on current mode
	switch activeMode % 4 { // Use modulo to handle AnimationAll
	case AnimationTwinkle:
		a.updateTwinkle()
	case AnimationChase:
		a.updateChase()
	case AnimationPulse:
		a.updatePulse()
	case AnimationSparkle:
		a.updateSparkle()
	}

	// Advance one animation step per Update() call
	a.Tick++
}

// updateTwinkle implements random twinkling effect
func (a *ChristmasTreeAnimator) updateTwinkle() {
	for i := range a.Lights {
		if a.Tick >= a.Lights[i].NextToggleTick {
			a.Lights[i].On = !a.Lights[i].On
			// Random interval 2-10 ticks (was ~100-500ms at ~20 FPS)
			delayTicks := uint(2 + rand.IntN(9))
			a.Lights[i].NextToggleTick = a.Tick + delayTicks
		}
	}
}

// updateChase implements color chasing effect
func (a *ChristmasTreeAnimator) updateChase() {
	colors := []Color{LightRed, LightBlue, LightYellow, LightWhite}
	for i := range a.Lights {
		// Each light offset by 2 frames, color changes every 15 frames
		offset := (a.Tick + uint(i*2)) / uint(15)
		a.Lights[i].Color = colors[int(offset%uint(len(colors)))]
		a.Lights[i].On = true
	}
}

// updatePulse implements synchronized pulsing effect
func (a *ChristmasTreeAnimator) updatePulse() {
	// Tick-based sine wave: period = 40 ticks (was ~2s at ~20 FPS).
	const periodTicks uint = 40
	phase := 2 * math.Pi * float64(a.Tick%periodTicks) / float64(periodTicks)
	brightness := (math.Sin(phase) + 1) / 2

	for i := range a.Lights {
		base := a.Lights[i].BaseColor
		a.Lights[i].Color = Color{
			R: uint8(float64(base.R) * brightness),
			G: uint8(float64(base.G) * brightness),
			B: uint8(float64(base.B) * brightness),
		}
		a.Lights[i].On = true
	}
}

// updateSparkle implements random sparkle burst effect
func (a *ChristmasTreeAnimator) updateSparkle() {
	// Check for sparkles ending
	for i := range a.Lights {
		if a.Tick >= a.Lights[i].SparkleEndTick {
			a.Lights[i].Color = a.Lights[i].BaseColor
		}
		a.Lights[i].On = true
	}

	// Randomly trigger new sparkles (5% chance per frame)
	if rand.Float64() < 0.05 {
		sparkleIdx := rand.IntN(len(a.Lights))
		a.Lights[sparkleIdx].Color = LightWhite
		a.Lights[sparkleIdx].SparkleEndTick = a.Tick + uint(6)
	}
}

// drawChristmasTree renders the complete tree to the framebuffer
func drawChristmasTree(fb *Framebuffer, animator *ChristmasTreeAnimator) {
	// 1. Clear to black
	black := Color{R: 0, G: 0, B: 0}
	fb.Clear(black)

	// 2. Draw tree foliage (green background)
	for y, xCoords := range treePixels {
		for _, x := range xCoords {
			if err := fb.SetPixel(x, y, TreeGreen); err != nil {
				return
			}
		}
	}

	// 3. Draw trunk (row 4, pixels 9-11)
	for x := 9; x <= 11; x++ {
		if err := fb.SetPixel(x, 4, TrunkBrown); err != nil {
			return
		}
	}

	// 4. Draw lights (overlay on tree)
	for _, light := range animator.Lights {
		if light.On {
			if err := fb.SetPixel(light.X, light.Y, light.Color); err != nil {
				return
			}
		} else {
			if err := fb.SetPixel(light.X, light.Y, LightOff); err != nil {
				return
			}
		}
	}
}

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
	for range numLEDs {
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

	for i := range numLEDs {
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

	for i := range numLEDs {
		// Calculate position in gradient (0.0 to 1.0)
		pos := float64(i) / float64(numLEDs)

		var r, g, b uint8

		// Rainbow gradient using piecewise linear interpolation
		switch {
		case pos < 0.167: // Red to Yellow
			r = 255
			g = uint8(pos / 0.167 * 255)
			b = 0
		case pos < 0.333: // Yellow to Green
			r = uint8((0.333 - pos) / 0.166 * 255)
			g = 255
			b = 0
		case pos < 0.5: // Green to Cyan
			r = 0
			g = 255
			b = uint8((pos - 0.333) / 0.167 * 255)
		case pos < 0.667: // Cyan to Blue
			r = 0
			g = uint8((0.667 - pos) / 0.167 * 255)
			b = 255
		case pos < 0.833: // Blue to Magenta
			r = uint8((pos - 0.667) / 0.166 * 255)
			g = 0
			b = 255
		default: // Magenta to Red
			r = 255
			g = 0
			b = uint8((1.0 - pos) / 0.167 * 255)
		}

		builder.WriteString(encodeRGBColor(r, g, b))
	}

	return builder.String()
}
