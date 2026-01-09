package main

import (
	"context"
	"fmt"
	"log"
	"sync"
	"time"

	"cubik/api"
)

type AnimationState struct {
	DeviceLocation string
	Frames         [][]Color
	CancelFunc     context.CancelFunc
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
				log.Printf("Error updating LEDs for %s: %v", state.DeviceLocation, err)
			}

			frameIndex++
		}
	}
}

func StartDeviceAnimation(deviceLocation string, frames [][]Color) error {
	StopDeviceAnimation(deviceLocation)

	ctx, cancelFunc := context.WithCancel(context.Background())
	state := &AnimationState{
		DeviceLocation: deviceLocation,
		Frames:         frames,
		CancelFunc:     cancelFunc,
	}

	animationsMu.Lock()
	runningAnimations[deviceLocation] = state
	animationsMu.Unlock()

	go func() {
		defer func() {
			animationsMu.Lock()
			delete(runningAnimations, deviceLocation)
			animationsMu.Unlock()
			cancelFunc()
		}()

		if err := PlayAnimation(ctx, state); err != nil {
			log.Printf("Animation error for %s: %v", deviceLocation, err)
		}
	}()

	return nil
}

func StopDeviceAnimation(deviceLocation string) {
	animationsMu.Lock()
	state, exists := runningAnimations[deviceLocation]
	animationsMu.Unlock()
	if exists {
		state.CancelFunc()
	}
}
