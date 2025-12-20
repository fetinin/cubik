package main

import (
	"fmt"
	"math"
	"math/rand"
	"os"
	"time"
)

func main() {
	fmt.Println("Searching for Yeelight CubeLite devices...")
	fmt.Println("Note: For TCP control to work, enable 'LAN Control' in the Yeelight app settings.")
	fmt.Println()

	devices, err := DiscoverDevices()
	if err != nil {
		fmt.Printf("Error discovering devices: %v\n", err)
		os.Exit(1)
	}

	if len(devices) == 0 {
		fmt.Println("No Yeelight CubeLite devices found.")
		return
	}

	fmt.Printf("Found %d device(s):\n\n", len(devices))

	for i, device := range devices {
		fmt.Printf("Device #%d:\n", i+1)
		fmt.Printf("  Location:    %s\n", device.Location)
		fmt.Printf("  ID:          %s\n", device.ID)
		fmt.Printf("  Model:       %s\n", device.Model)
		fmt.Printf("  FW Version:  %s\n", device.FwVer)
		fmt.Printf("  Power:       %s\n", device.Power)
		fmt.Printf("  Brightness:  %s\n", device.Bright)
		fmt.Printf("  Color Mode:  %s\n", device.ColorMode)

		if device.CT != "" {
			fmt.Printf("  CT:          %s\n", device.CT)
		}
		if device.RGB != "" {
			fmt.Printf("  RGB:         %s\n", device.RGB)
		}
		if device.Hue != "" {
			fmt.Printf("  Hue:         %s\n", device.Hue)
		}
		if device.Sat != "" {
			fmt.Printf("  Saturation:  %s\n", device.Sat)
		}
		if device.Name != "" {
			fmt.Printf("  Name:        %s\n", device.Name)
		}
		// Always print Support field, even if empty
		fmt.Printf("  Support:     %s\n", device.Support)
		//if device.Support == "" {
		//	fmt.Println("  WARNING: Device reports no supported methods - may not support standard Yeelight protocol")
		//	continue
		//}

		// Query live properties using get_prop
		fmt.Println()
		//fmt.Println("  Querying live properties via TCP...")
		//
		//// Try with just one property first
		//props, err := GetProp(device, "music_on")
		//if err != nil {
		//	fmt.Printf("  Error querying properties: %v\n", err)
		//	fmt.Println("  Troubleshooting tips:")
		//	fmt.Println("    - Verify 'LAN Control' is enabled in Yeelight app")
		//	fmt.Println("    - Check if device is on the same network")
		//	fmt.Println("    - Some CubeLite models may have limited protocol support")
		//} else {
		//	fmt.Println("  Live Properties:")
		//	for key, value := range props {
		//		if value != "" {
		//			fmt.Printf("    %s: %s\n", key, value)
		//		}
		//	}
		//}

		// Demonstrate toggle functionality
		fmt.Println()
		fmt.Println("  Testing power toggle command...")
		fmt.Printf("  Current power state: %s\n", device.Power)

		err = TogglePower(device)
		if err != nil {
			fmt.Printf("  Error toggling power: %v\n", err)
		}

		// Matrix LED Control Demo
		fmt.Println()
		fmt.Println("  Matrix LED Control Demo")
		fmt.Println("  ========================")

		// Activate direct mode
		err = ActivateFxMode(device)
		if err != nil {
			fmt.Printf("  Error activating direct mode: %v\n", err)
			return
		}

		// Demo Christmas tree animation
		demoChristmasTree(device)
	}

	fmt.Println()
}

func demoTestPatterns(device *DeviceInfo) {
	for {
		patterns := []struct {
			name string
			data string
		}{
			{"Checkerboard", createCheckerboard(100)},
			{"Solid Red", createSolidColor(255, 0, 0, 100)},
			{"Checkerboard", createCheckerboard(100)},
			//{"Solid Green", createSolidColor(0, 255, 0, 100)},
			{"Solid Blue", createSolidColor(0, 0, 255, 100)},
			//{"Rainbow Gradient", createGradient(100)},
		}

		for _, pattern := range patterns {
			fmt.Printf("\n  Displaying: %s\n", pattern.name)
			//fmt.Print("  Press Enter to continue...")
			//bufio.NewReader(os.Stdin).ReadString('\n')
			time.Sleep(1*time.Second + 100*time.Millisecond)

			err := UpdateLeds(device, pattern.data)
			if err != nil {
				fmt.Printf("  Error updating LEDs: %v\n", err)
			} else {
				fmt.Println("  ✓ Pattern displayed")
			}
		}
	}

	fmt.Println()
	fmt.Println("  Matrix demo complete!")
}

func demoDigitDisplay(device *DeviceInfo) {
	fb := NewFramebuffer(20, 5)
	black := Color{R: 0, G: 0, B: 0}

	// Demo 1: Single digits 0-9 in green
	green := Color{R: 0, G: 255, B: 0}
	fmt.Println("\n  Demo: Single digits 0-9")
	for i := 0; i <= 9; i++ {
		fb.Clear(black)
		DrawNumber(fb, i, 0, 0, AlignCenter, green, black)
		UpdateLeds(device, fb.Encode())
		time.Sleep(500 * time.Millisecond)
	}

	// Demo 2: Multi-digit numbers in blue
	blue := Color{R: 0, G: 0, B: 255}
	numbers := []int{42, 123, 999, 2025}
	fmt.Println("\n  Demo: Multi-digit numbers")
	for _, num := range numbers {
		fb.Clear(black)
		DrawNumber(fb, num, 0, 1, AlignCenter, blue, black)
		UpdateLeds(device, fb.Encode())
		time.Sleep(1 * time.Second)
	}

	// Demo 3: Countdown in red
	red := Color{R: 255, G: 0, B: 0}
	fmt.Println("\n  Demo: Countdown 10 to 0")
	for i := 10; i >= 0; i-- {
		fb.Clear(black)
		DrawNumber(fb, i, 0, 1, AlignCenter, red, black)
		UpdateLeds(device, fb.Encode())
		time.Sleep(500 * time.Millisecond)
	}

	// Demo 4: Alignment showcase
	white := Color{R: 255, G: 255, B: 255}
	fmt.Println("\n  Demo: Alignment (42 left/center/right)")
	alignments := []Alignment{AlignLeft, AlignCenter, AlignRight}
	for _, align := range alignments {
		fb.Clear(black)
		DrawNumber(fb, 42, 0, 1, align, white, black)
		UpdateLeds(device, fb.Encode())
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
	X, Y       int       // Position on matrix
	On         bool      // Current on/off state
	Color      Color     // Current color
	NextToggle time.Time // For twinkle effect
	BaseColor  Color     // Original color for pulse effect
	SparkleEnd time.Time // When sparkle effect ends
}

// ChristmasTreeAnimator manages the animation state
type ChristmasTreeAnimator struct {
	Lights         []LightState
	Frame          int
	StartTime      time.Time
	Mode           AnimationMode
	LastModeSwitch time.Time
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
	{10, 0},                      // Row 0: 1 light (top)
	{9, 1}, {11, 1},              // Row 1: 2 lights
	{8, 2}, {10, 2}, {12, 2},     // Row 2: 3 lights
	{7, 3}, {9, 3}, {11, 3}, {13, 3}, // Row 3: 4 lights
}

// Tree foliage coordinates by row
var treePixels = map[int][]int{
	0: {10},                    // Row 0: 1 pixel
	1: {9, 10, 11},             // Row 1: 3 pixels
	2: {8, 9, 10, 11, 12},      // Row 2: 5 pixels
	3: {7, 8, 9, 10, 11, 12, 13}, // Row 3: 7 pixels
}

// demoChristmasTree runs the Christmas tree animation demo
func demoChristmasTree(device *DeviceInfo) {
	fmt.Println("\n  Christmas Tree Animation Demo")
	fmt.Println("  =============================")

	animator := NewChristmasTreeAnimator(AnimationAll)
	fb := NewFramebuffer(20, 5)
	frameDelay := 1 * time.Second // 20 FPS

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

		// Frame timing
		time.Sleep(frameDelay)
		animator.Frame++
	}
}

// NewChristmasTreeAnimator creates a new animator with initialized lights
func NewChristmasTreeAnimator(mode AnimationMode) ChristmasTreeAnimator {
	animator := ChristmasTreeAnimator{
		Lights:         make([]LightState, len(lightPositions)),
		Frame:          0,
		StartTime:      time.Now(),
		Mode:           mode,
		LastModeSwitch: time.Now(),
	}

	// Initialize light states with different colors
	colors := []Color{LightRed, LightBlue, LightYellow, LightWhite}
	for i, pos := range lightPositions {
		animator.Lights[i] = LightState{
			X:          pos.X,
			Y:          pos.Y,
			On:         true,
			Color:      colors[i%len(colors)],
			BaseColor:  colors[i%len(colors)],
			NextToggle: time.Now().Add(time.Duration(rand.Intn(500)) * time.Millisecond),
		}
	}

	return animator
}

// Update advances the animation state
func (a *ChristmasTreeAnimator) Update() {
	// Switch modes every 10 seconds if AnimationAll
	if a.Mode == AnimationAll {
		elapsed := time.Since(a.LastModeSwitch)
		if elapsed > 10*time.Second {
			// Cycle through modes: Twinkle → Chase → Pulse → Sparkle
			currentMode := (a.Frame / 200) % 4 // Switch every 10 seconds at 20 FPS
			a.Mode = AnimationMode(currentMode)
			a.LastModeSwitch = time.Now()
			fmt.Printf("  Switching to animation mode: %d\n", a.Mode)
		}
	}

	// Update based on current mode
	switch a.Mode % 4 { // Use modulo to handle AnimationAll
	case AnimationTwinkle:
		a.updateTwinkle()
	case AnimationChase:
		a.updateChase()
	case AnimationPulse:
		a.updatePulse()
	case AnimationSparkle:
		a.updateSparkle()
	}
}

// updateTwinkle implements random twinkling effect
func (a *ChristmasTreeAnimator) updateTwinkle() {
	now := time.Now()
	for i := range a.Lights {
		if now.After(a.Lights[i].NextToggle) {
			a.Lights[i].On = !a.Lights[i].On
			// Random interval 100-500ms
			delay := time.Duration(100+rand.Intn(400)) * time.Millisecond
			a.Lights[i].NextToggle = now.Add(delay)
		}
	}
}

// updateChase implements color chasing effect
func (a *ChristmasTreeAnimator) updateChase() {
	colors := []Color{LightRed, LightBlue, LightYellow, LightWhite}
	for i := range a.Lights {
		// Each light offset by 2 frames, color changes every 15 frames
		offset := (a.Frame + i*2) / 15
		a.Lights[i].Color = colors[offset%len(colors)]
		a.Lights[i].On = true
	}
}

// updatePulse implements synchronized pulsing effect
func (a *ChristmasTreeAnimator) updatePulse() {
	elapsed := time.Since(a.StartTime).Seconds()
	// Sine wave: period = 2 seconds
	brightness := (math.Sin(elapsed*math.Pi) + 1) / 2

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
	now := time.Now()

	// Check for sparkles ending
	for i := range a.Lights {
		if now.After(a.Lights[i].SparkleEnd) {
			a.Lights[i].Color = a.Lights[i].BaseColor
		}
		a.Lights[i].On = true
	}

	// Randomly trigger new sparkles (5% chance per frame)
	if rand.Float64() < 0.05 {
		sparkleIdx := rand.Intn(len(a.Lights))
		a.Lights[sparkleIdx].Color = LightWhite
		a.Lights[sparkleIdx].SparkleEnd = now.Add(300 * time.Millisecond)
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
			fb.SetPixel(x, y, TreeGreen)
		}
	}

	// 3. Draw trunk (row 4, pixels 9-11)
	for x := 9; x <= 11; x++ {
		fb.SetPixel(x, 4, TrunkBrown)
	}

	// 4. Draw lights (overlay on tree)
	for _, light := range animator.Lights {
		if light.On {
			fb.SetPixel(light.X, light.Y, light.Color)
		} else {
			fb.SetPixel(light.X, light.Y, LightOff)
		}
	}
}
