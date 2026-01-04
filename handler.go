package main

import (
	"context"
	"fmt"
	"log"
	"strings"

	"cubik/api"
)

// APIHandler implements the ogen-generated Handler interface
type APIHandler struct{}

// Ensure APIHandler implements the interface at compile time
var _ api.Handler = (*APIHandler)(nil)

// GetDevices implements GET /api/devices
// Performs live discovery and transforms DeviceInfo to API response model
func (h *APIHandler) GetDevices(ctx context.Context) (api.GetDevicesRes, error) {
	// Call existing discovery function
	devices, err := DiscoverDevices()
	if err != nil {
		// Log error for debugging
		log.Printf("Discovery error: %v", err)

		// Return 500 error response
		return &api.Error{
			Error: err.Error(),
		}, nil
	}

	// Transform []*DeviceInfo to []api.Device
	apiDevices := make([]api.Device, 0, len(devices))
	for _, device := range devices {
		apiDevices = append(apiDevices, api.Device{
			ID:       device.ID,
			Name:     device.Name,
			Location: device.Location,
		})
	}

	// Return 200 success response
	return &api.GetDevicesOK{
		Devices: apiDevices,
	}, nil
}

// StartAnimation implements POST /animation/start
// Starts an animation on the specified device with the provided frames
func (h *APIHandler) StartAnimation(ctx context.Context, req *api.StartAnimationRequest) (api.StartAnimationRes, error) {
	// Validate device location format
	if !strings.HasPrefix(req.DeviceLocation, "yeelight://") {
		return &api.StartAnimationBadRequest{Error: "invalid device_location format"}, nil
	}

	// Validate frames array not empty
	if len(req.Frames) == 0 {
		return &api.StartAnimationBadRequest{Error: "frames array cannot be empty"}, nil
	}

	// Convert API frames to internal Color arrays
	// RGB validation is handled by ogen-generated code
	internalFrames := make([][]Color, len(req.Frames))
	for i, apiFrame := range req.Frames {
		internalFrames[i] = ConvertAPIFrameToColors(apiFrame)
	}

	// Start animation (stops existing if any)
	if err := StartDeviceAnimation(req.DeviceLocation, internalFrames); err != nil {
		return &api.StartAnimationInternalServerError{Error: fmt.Sprintf("failed to start animation: %v", err)}, nil
	}

	// Return success
	return &api.StartAnimationResponse{
		Message:    "Animation started successfully",
		FrameCount: len(req.Frames),
	}, nil
}

// StopAnimation implements POST /animation/stop
// Stops the animation on the specified device
func (h *APIHandler) StopAnimation(ctx context.Context, req *api.StopAnimationRequest) (api.StopAnimationRes, error) {
	// Validate device location format
	if !strings.HasPrefix(req.DeviceLocation, "yeelight://") {
		return &api.StopAnimationBadRequest{Error: "invalid device_location format"}, nil
	}

	// Stop animation (no-op if not running)
	StopDeviceAnimation(req.DeviceLocation)

	// Return success
	return &api.StopAnimationResponse{
		Message: "Animation stopped successfully",
	}, nil
}
