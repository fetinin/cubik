package main

import (
	"context"
	"cubik/api"
	"fmt"
	"log/slog"
	"sync"
	"time"
)

type AnimationState struct {
	DeviceLocation string
	// AnimationID links the playback to a saved animation; empty means unsaved.
	AnimationID string
	Frames      [][]Color
	StopFunc    func()
}

const (
	PowerOn      = "on"
	PowerOff     = "off"
	PowerUnknown = "unknown"
)

// The backend is the only source of truth for power and playback: the
// CubeLite can't be queried. Never send get_prop to it — it doesn't answer
// and triggers a temporary built-in animation. SSDP state fields are a fixed
// template and the device sends no push notifications. This state lives in
// memory and is lost on restart.
var (
	runningAnimations = make(map[string]*AnimationState)
	devicePower       = make(map[string]string)
	animationsMu      sync.RWMutex
)

func SetTrackedPower(deviceLocation, state string) {
	animationsMu.Lock()
	devicePower[deviceLocation] = state
	animationsMu.Unlock()
}

// TrackedPower returns the last power state set through this backend, or
// PowerUnknown if none was set since startup.
func TrackedPower(deviceLocation string) string {
	animationsMu.RLock()
	defer animationsMu.RUnlock()
	if state, ok := devicePower[deviceLocation]; ok {
		return state
	}
	return PowerUnknown
}

// PlaybackSnapshot reports what is looping on the device, if anything.
func PlaybackSnapshot(deviceLocation string) (string, [][]Color, bool) {
	animationsMu.RLock()
	defer animationsMu.RUnlock()
	state, ok := runningAnimations[deviceLocation]
	if !ok {
		return "", nil, false
	}
	return state.AnimationID, state.Frames, true
}

// DetachAnimation unlinks running playbacks from a saved animation that was
// changed or deleted. The frames keep playing and show up as unsaved.
func DetachAnimation(animationID string) {
	animationsMu.Lock()
	defer animationsMu.Unlock()
	for _, state := range runningAnimations {
		if state.AnimationID == animationID {
			state.AnimationID = ""
		}
	}
}

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

func PlayAnimation(ctx context.Context, state *AnimationState) error {
	deviceInfo := &DeviceInfo{Location: state.DeviceLocation}

	if err := ActivateFxMode(deviceInfo); err != nil {
		return fmt.Errorf("failed to activate fx mode: %w", err)
	}

	fb := NewFramebuffer(20, 5)
	ticker := time.NewTicker(1 * time.Second)
	defer ticker.Stop()

	frameIndex := 0
	for {
		select {
		case <-ctx.Done():
			return nil
		case <-ticker.C:
			if frameIndex >= len(state.Frames) {
				frameIndex = 0
			}

			copy(fb.Pixels, state.Frames[frameIndex])
			if err := UpdateLeds(deviceInfo, fb.Encode()); err != nil {
				slog.Error("Error updating LEDs", "device", state.DeviceLocation, "error", err)
			}

			frameIndex++
		}
	}
}

// StartDeviceAnimation also marks the device as powered on: the first
// update_leds frame wakes a cube that's off.
func StartDeviceAnimation(deviceLocation, animationID string, frames [][]Color) {
	StopDeviceAnimation(deviceLocation)

	ctx, cancelFunc := context.WithCancel(context.Background())
	done := make(chan struct{})
	state := &AnimationState{
		DeviceLocation: deviceLocation,
		AnimationID:    animationID,
		Frames:         frames,
		StopFunc: func() {
			cancelFunc()
			<-done
		},
	}

	animationsMu.Lock()
	runningAnimations[deviceLocation] = state
	devicePower[deviceLocation] = PowerOn
	animationsMu.Unlock()

	go func() {
		defer func() {
			animationsMu.Lock()
			// Don't remove a newer playback that has already taken this slot.
			if runningAnimations[deviceLocation] == state {
				delete(runningAnimations, deviceLocation)
			}
			animationsMu.Unlock()
			close(done)
		}()

		if err := PlayAnimation(ctx, state); err != nil {
			slog.Error("Animation error", "device", deviceLocation, "error", err)
		}
	}()
}

func StopDeviceAnimation(deviceLocation string) {
	animationsMu.Lock()
	state, exists := runningAnimations[deviceLocation]
	animationsMu.Unlock()
	if exists {
		state.StopFunc()
	}
}
