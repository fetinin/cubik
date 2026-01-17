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
	Frames         [][]Color
	StopFunc       func()
}

var (
	runningAnimations = make(map[string]*AnimationState)
	animationsMu      sync.RWMutex
)

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

func StartDeviceAnimation(deviceLocation string, frames [][]Color) {
	StopDeviceAnimation(deviceLocation)

	ctx, cancelFunc := context.WithCancel(context.Background())
	done := make(chan struct{})
	state := &AnimationState{
		DeviceLocation: deviceLocation,
		Frames:         frames,
		StopFunc: func() {
			cancelFunc()
			<-done
		},
	}

	animationsMu.Lock()
	runningAnimations[deviceLocation] = state
	animationsMu.Unlock()

	go func() {
		defer func() {
			animationsMu.Lock()
			delete(runningAnimations, deviceLocation)
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
