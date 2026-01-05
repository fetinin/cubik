package main

import (
	"context"
	"database/sql"
	"fmt"
	"log"
	"strings"

	"cubik/api"
)

// APIHandler implements the ogen-generated Handler interface
type APIHandler struct {
	db *sql.DB
}

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
	// Convert API frames to internal Color arrays
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

// SaveAnimation implements POST /animation/save
// Saves animation frames to the database
func (h *APIHandler) SaveAnimation(ctx context.Context, req *api.SaveAnimationRequest) (api.SaveAnimationRes, error) {
	// TODO: Convert API frames to internal Color format
	// TODO: Save to database using storage layer
	// TODO: Return saved animation with generated UUID and timestamps

	return &api.SaveAnimationInternalServerError{
		Error: "not implemented yet",
	}, nil
}

// ListAnimations implements GET /animation/list/{device_id}
// Returns all saved animations for a device
func (h *APIHandler) ListAnimations(ctx context.Context, params api.ListAnimationsParams) (api.ListAnimationsRes, error) {
	// TODO: Query database for animations by device_id
	// TODO: Convert internal format to API format
	// TODO: Order by updated_at DESC

	return &api.ListAnimationsResponse{
		Animations: []api.SavedAnimation{},
	}, nil
}

// GetAnimation implements GET /animation/{id}
// Retrieves a specific saved animation by ID
func (h *APIHandler) GetAnimation(ctx context.Context, params api.GetAnimationParams) (api.GetAnimationRes, error) {
	// TODO: Query database for animation by ID (params.ID)
	// TODO: Return 404 if not found
	// TODO: Convert internal format to API format

	return &api.GetAnimationNotFound{
		Error: "not implemented yet",
	}, nil
}

// UpdateAnimation implements PUT /animation/{id}
// Updates an existing saved animation
func (h *APIHandler) UpdateAnimation(ctx context.Context, req *api.UpdateAnimationRequest, params api.UpdateAnimationParams) (api.UpdateAnimationRes, error) {
	// TODO: Convert API frames to internal Color format
	// TODO: Update animation in database by params.ID
	// TODO: Return 404 if not found
	// TODO: Update updated_at timestamp

	return &api.UpdateAnimationNotFound{
		Error: "not implemented yet",
	}, nil
}

// DeleteAnimation implements DELETE /animation/{id}
// Deletes a saved animation from the database
func (h *APIHandler) DeleteAnimation(ctx context.Context, params api.DeleteAnimationParams) (api.DeleteAnimationRes, error) {
	// TODO: Delete animation from database by params.ID
	// TODO: Return 404 if not found

	return &api.DeleteAnimationNotFound{
		Error: "not implemented yet",
	}, nil
}
