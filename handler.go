package main

import (
	"context"
	"cubik/api"
	"database/sql"
	"errors"
	"fmt"
	"log/slog"
)

type APIHandler struct {
	db *sql.DB
}

var _ api.Handler = (*APIHandler)(nil)

func (h *APIHandler) GetDevices(_ context.Context) (api.GetDevicesRes, error) {
	devices, err := DiscoverDevices()
	if err != nil {
		slog.Error("Discovery error", "error", err)
		return &api.Error{Error: err.Error()}, nil
	}

	apiDevices := make([]api.Device, 0, len(devices))
	for _, device := range devices {
		apiDevices = append(apiDevices, api.Device{
			ID:       device.ID,
			Name:     device.Name,
			Location: device.Location,
		})
	}

	return &api.GetDevicesOK{Devices: apiDevices}, nil
}

func (h *APIHandler) StartAnimation(
	_ context.Context,
	req *api.StartAnimationRequest,
) (api.StartAnimationRes, error) {
	internalFrames := make([][]Color, len(req.Frames))
	for i, apiFrame := range req.Frames {
		internalFrames[i] = ConvertAPIFrameToColors(apiFrame)
	}

	StartDeviceAnimation(req.DeviceLocation, internalFrames)

	return &api.StartAnimationResponse{
		Message:    "Animation started successfully",
		FrameCount: len(req.Frames),
	}, nil
}

func (h *APIHandler) StopAnimation(_ context.Context, req *api.StopAnimationRequest) (api.StopAnimationRes, error) {
	StopDeviceAnimation(req.DeviceLocation)
	return &api.StopAnimationResponse{Message: "Animation stopped successfully"}, nil
}

func (h *APIHandler) SaveAnimation(ctx context.Context, req *api.SaveAnimationRequest) (api.SaveAnimationRes, error) {
	frames := make([][]Color, len(req.Frames))
	for i, apiFrame := range req.Frames {
		frames[i] = ConvertAPIFrameToColors(apiFrame)
	}

	animation, err := SaveAnimation(ctx, h.db, req.DeviceID, req.Name, frames)
	if err != nil {
		return &api.SaveAnimationInternalServerError{
			Error: fmt.Sprintf("failed to save animation: %v", err),
		}, nil
	}

	return &api.SaveAnimationResponse{
		ID:        animation.ID,
		Message:   "Animation saved successfully",
		Animation: convertToAPIAnimation(animation),
	}, nil
}

func (h *APIHandler) ListAnimations(
	ctx context.Context,
	params api.ListAnimationsParams,
) (api.ListAnimationsRes, error) {
	animations, err := ListAnimationsByDevice(ctx, h.db, params.DeviceID)
	if err != nil {
		return &api.Error{Error: fmt.Sprintf("failed to list animations: %v", err)}, nil
	}

	apiAnimations := make([]api.SavedAnimation, len(animations))
	for i, anim := range animations {
		apiAnimations[i] = convertToAPIAnimation(anim)
	}

	return &api.ListAnimationsResponse{Animations: apiAnimations}, nil
}

func (h *APIHandler) GetAnimation(ctx context.Context, params api.GetAnimationParams) (api.GetAnimationRes, error) {
	animation, err := GetAnimation(ctx, h.db, params.ID)
	if errors.Is(err, ErrNotFound) {
		return &api.GetAnimationNotFound{Error: "animation not found"}, nil
	}
	if err != nil {
		return &api.GetAnimationInternalServerError{
			Error: fmt.Sprintf("failed to get animation: %v", err),
		}, nil
	}

	return &api.GetAnimationResponse{Animation: convertToAPIAnimation(animation)}, nil
}

func (h *APIHandler) UpdateAnimation(
	ctx context.Context,
	req *api.UpdateAnimationRequest,
	params api.UpdateAnimationParams,
) (api.UpdateAnimationRes, error) {
	frames := make([][]Color, len(req.Frames))
	for i, apiFrame := range req.Frames {
		frames[i] = ConvertAPIFrameToColors(apiFrame)
	}

	animation, err := UpdateAnimation(ctx, h.db, params.ID, req.Name, frames)
	if errors.Is(err, ErrNotFound) {
		return &api.UpdateAnimationNotFound{Error: "animation not found"}, nil
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

func (h *APIHandler) DeleteAnimation(
	ctx context.Context,
	params api.DeleteAnimationParams,
) (api.DeleteAnimationRes, error) {
	err := DeleteAnimation(ctx, h.db, params.ID)
	if errors.Is(err, ErrNotFound) {
		return &api.DeleteAnimationNotFound{Error: "animation not found"}, nil
	}
	if err != nil {
		return &api.DeleteAnimationInternalServerError{
			Error: fmt.Sprintf("failed to delete animation: %v", err),
		}, nil
	}

	return &api.DeleteAnimationResponse{Message: "Animation deleted successfully"}, nil
}

func convertToAPIAnimation(anim *SavedAnimation) api.SavedAnimation {
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
