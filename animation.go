package main

import (
	"context"
	"fmt"
	"log"
	"sync"
	"time"

	"cubik/api"
)

// AnimationState tracks a running animation for a specific device
type AnimationState struct {
	DeviceLocation string
	Frames         [][]Color // Pre-converted frames (each frame is 100 Color values)
	CancelFunc     context.CancelFunc
}

// Global registry of running animations, keyed by device location
var (
	runningAnimations = make(map[string]*AnimationState)
	animationsMu      sync.RWMutex // Protects map access
)

// ConvertAPIFrameToColors converts an API frame (RGBPixel objects) to internal Color array
// RGB validation is handled by ogen-generated code based on OpenAPI schema constraints
func ConvertAPIFrameToColors(apiFrame []api.RGBPixel) []Color {
	colors := make([]Color, len(apiFrame))
	for i, pixel := range apiFrame {
		colors[i] = Color{
			R: uint8(pixel.R),
			G: uint8(pixel.G),
			B: uint8(pixel.B),
		}
	}

	return colors
}

// PlayAnimation runs the animation loop in a goroutine
// Activates direct mode, then sends frames to device every 1 second
// Loops frames infinitely until context is cancelled
func PlayAnimation(ctx context.Context, state *AnimationState) error {
	// Create device info for commands
	deviceInfo := &DeviceInfo{Location: state.DeviceLocation}

	// Activate direct mode on device (required before UpdateLeds)
	if err := ActivateFxMode(deviceInfo); err != nil {
		return fmt.Errorf("failed to activate fx mode: %w", err)
	}

	// Create framebuffer for encoding frames
	fb := NewFramebuffer(20, 5)

	// Setup ticker for 1-second frame timing
	ticker := time.NewTicker(1 * time.Second)
	defer ticker.Stop()

	frameIndex := 0

	// Animation loop
	for {
		select {
		case <-ctx.Done():
			// Context cancelled - clean exit
			return nil

		case <-ticker.C:
			// Loop frames infinitely
			if frameIndex >= len(state.Frames) {
				frameIndex = 0
			}

			// Copy frame pixels to framebuffer
			copy(fb.Pixels, state.Frames[frameIndex])

			// Encode and send to device (fire-and-forget)
			encoded := fb.Encode()
			if err := UpdateLeds(deviceInfo, encoded); err != nil {
				// Log error but continue - don't stop animation for network hiccups
				log.Printf("Error updating LEDs for %s: %v", state.DeviceLocation, err)
			}

			frameIndex++
		}
	}
}

// StartDeviceAnimation starts a new animation on the specified device
// Stops any existing animation for this device first
// Animation runs in a background goroutine
func StartDeviceAnimation(deviceLocation string, frames [][]Color) error {
	// Stop any existing animation for this device
	StopDeviceAnimation(deviceLocation)

	// Create cancellable context
	ctx, cancelFunc := context.WithCancel(context.Background())

	// Create animation state
	state := &AnimationState{
		DeviceLocation: deviceLocation,
		Frames:         frames,
		CancelFunc:     cancelFunc,
	}

	// Store in global registry
	animationsMu.Lock()
	runningAnimations[deviceLocation] = state
	animationsMu.Unlock()

	// Start animation goroutine
	go func() {
		// Cleanup on exit - remove from registry
		defer func() {
			animationsMu.Lock()
			delete(runningAnimations, deviceLocation)
			cancelFunc()
			animationsMu.Unlock()
		}()

		// Run animation loop
		if err := PlayAnimation(ctx, state); err != nil {
			log.Printf("Animation error for %s: %v", deviceLocation, err)
		}
	}()

	return nil
}

// StopDeviceAnimation stops the animation for the specified device
// No-op if no animation is currently running for this device
func StopDeviceAnimation(deviceLocation string) {
	// Lock and check if animation exists
	animationsMu.Lock()
	state, exists := runningAnimations[deviceLocation]
	if exists {
		state.CancelFunc()
		delete(runningAnimations, deviceLocation)
	}
	animationsMu.Unlock()
}
