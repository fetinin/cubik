package main

import (
	"context"
	"database/sql"
	"fmt"
	"log"

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
	// Convert API frames to internal Color format
	frames := make([][]Color, len(req.Frames))
	for i, apiFrame := range req.Frames {
		frames[i] = ConvertAPIFrameToColors(apiFrame)
	}

	// Save to database
	animation, err := SaveAnimation(h.db, req.DeviceID, req.Name, frames)
	if err != nil {
		return &api.SaveAnimationInternalServerError{
			Error: fmt.Sprintf("failed to save animation: %v", err),
		}, nil
	}

	// Convert to API format and return
	return &api.SaveAnimationResponse{
		ID:        animation.ID,
		Message:   "Animation saved successfully",
		Animation: convertToAPIAnimation(animation),
	}, nil
}

// ListAnimations implements GET /animation/list/{device_id}
// Returns all saved animations for a device
func (h *APIHandler) ListAnimations(ctx context.Context, params api.ListAnimationsParams) (api.ListAnimationsRes, error) {
	// Query database for animations by device_id (ordered by updated_at DESC)
	animations, err := ListAnimationsByDevice(h.db, params.DeviceID)
	if err != nil {
		return &api.Error{
			Error: fmt.Sprintf("failed to list animations: %v", err),
		}, nil
	}

	// Convert to API format
	apiAnimations := make([]api.SavedAnimation, len(animations))
	for i, anim := range animations {
		apiAnimations[i] = convertToAPIAnimation(anim)
	}

	return &api.ListAnimationsResponse{
		Animations: apiAnimations,
	}, nil
}

// GetAnimation implements GET /animation/{id}
// Retrieves a specific saved animation by ID
func (h *APIHandler) GetAnimation(ctx context.Context, params api.GetAnimationParams) (api.GetAnimationRes, error) {
	// Query database for animation by ID
	animation, err := GetAnimation(h.db, params.ID)
	if err == ErrNotFound {
		return &api.GetAnimationNotFound{
			Error: "animation not found",
		}, nil
	}
	if err != nil {
		return &api.GetAnimationInternalServerError{
			Error: fmt.Sprintf("failed to get animation: %v", err),
		}, nil
	}

	return &api.GetAnimationResponse{
		Animation: convertToAPIAnimation(animation),
	}, nil
}

// UpdateAnimation implements PUT /animation/{id}
// Updates an existing saved animation
func (h *APIHandler) UpdateAnimation(ctx context.Context, req *api.UpdateAnimationRequest, params api.UpdateAnimationParams) (api.UpdateAnimationRes, error) {
	// Convert API frames to internal Color format
	frames := make([][]Color, len(req.Frames))
	for i, apiFrame := range req.Frames {
		frames[i] = ConvertAPIFrameToColors(apiFrame)
	}

	// Update animation in database
	animation, err := UpdateAnimation(h.db, params.ID, req.Name, frames)
	if err == ErrNotFound {
		return &api.UpdateAnimationNotFound{
			Error: "animation not found",
		}, nil
	}
	if err != nil {
		return &api.UpdateAnimationInternalServerError{
			Error: fmt.Sprintf("failed to update animation: %v", err),
		}, nil
	}

	return &api.UpdateAnimationResponse{
		Message:   "Animation updated successfully",
		Animation: convertToAPIAnimation(animation),
	}, nil
}

// DeleteAnimation implements DELETE /animation/{id}
// Deletes a saved animation from the database
func (h *APIHandler) DeleteAnimation(ctx context.Context, params api.DeleteAnimationParams) (api.DeleteAnimationRes, error) {
	// Delete animation from database
	err := DeleteAnimation(h.db, params.ID)
	if err == ErrNotFound {
		return &api.DeleteAnimationNotFound{
			Error: "animation not found",
		}, nil
	}
	if err != nil {
		return &api.DeleteAnimationInternalServerError{
			Error: fmt.Sprintf("failed to delete animation: %v", err),
		}, nil
	}

	return &api.DeleteAnimationResponse{
		Message: "Animation deleted successfully",
	}, nil
}

// convertToAPIAnimation converts internal SavedAnimation to API format
func convertToAPIAnimation(anim *SavedAnimation) api.SavedAnimation {
	// Convert [][]Color to []api.AnimationFrame
	apiFrames := make([]api.AnimationFrame, len(anim.Frames))
	for i, frame := range anim.Frames {
		apiFrames[i] = make(api.AnimationFrame, len(frame))
		for j, color := range frame {
			apiFrames[i][j] = api.RGBPixel{
				R: int32(color.R),
				G: int32(color.G),
				B: int32(color.B),
			}
		}
	}

	return api.SavedAnimation{
		ID:        anim.ID,
		DeviceID:  anim.DeviceID,
		Name:      anim.Name,
		Frames:    apiFrames,
		CreatedAt: anim.CreatedAt,
		UpdatedAt: anim.UpdatedAt,
	}
}
