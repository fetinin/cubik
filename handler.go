package main

import (
	"context"
	"cubik/api"
	"database/sql"
	"errors"
	"fmt"
	"log/slog"

	"github.com/google/uuid"
)

const msgAnimationNotFound = "animation not found"

type APIHandler struct {
	db *sql.DB
}

var _ api.Handler = (*APIHandler)(nil)

func (h *APIHandler) GetDevices(ctx context.Context) (api.GetDevicesRes, error) {
	devices, err := DiscoverDevices()
	if err != nil {
		slog.Error("Discovery error", "error", err)
		return &api.Error{Error: err.Error()}, nil
	}

	apiDevices := make([]api.Device, 0, len(devices))
	for _, device := range devices {
		playback := api.OptNilPlayback{Set: true, Null: true}
		if pb, ok := h.buildPlayback(ctx, device.Location); ok {
			playback = api.NewOptNilPlayback(pb)
		}
		apiDevices = append(apiDevices, api.Device{
			ID:       device.ID,
			Name:     device.Name,
			Location: device.Location,
			Power:    api.DevicePower(TrackedPower(device.Location)),
			Playback: playback,
		})
	}

	return &api.GetDevicesOK{Devices: apiDevices}, nil
}

// buildPlayback describes what is looping on the device. A saved animation
// whose row has disappeared is detached and reported as unsaved.
func (h *APIHandler) buildPlayback(ctx context.Context, location string) (api.Playback, bool) {
	animationID, frames, ok := PlaybackSnapshot(location)
	if !ok {
		return api.Playback{}, false
	}
	if animationID != "" {
		anim, err := GetAnimation(ctx, h.db, animationID)
		switch {
		case err == nil:
			return savedPlayback(anim), true
		case errors.Is(err, ErrNotFound):
			DetachAnimation(animationID)
		default:
			slog.Error("Playback lookup failed", "animation_id", animationID, "error", err)
		}
	}
	return api.Playback{Frames: convertToAPIFrames(frames)}, true
}

func savedPlayback(anim *SavedAnimation) api.Playback {
	return api.Playback{
		AnimationID:   api.NewOptString(anim.ID),
		AnimationName: api.NewOptString(anim.Name),
	}
}

func (h *APIHandler) StartAnimation(
	_ context.Context,
	req *api.StartAnimationRequest,
) (api.StartAnimationRes, error) {
	internalFrames := make([][]Color, len(req.Frames))
	for i, apiFrame := range req.Frames {
		internalFrames[i] = ConvertAPIFrameToColors(apiFrame)
	}

	StartDeviceAnimation(req.DeviceLocation, "", internalFrames)

	return &api.StartAnimationResponse{
		Message:    "Animation started successfully",
		FrameCount: len(req.Frames),
		Playback:   api.NewOptPlayback(api.Playback{Frames: req.Frames}),
	}, nil
}

func (h *APIHandler) PlayAnimation(
	ctx context.Context,
	req *api.PlayAnimationRequest,
	params api.PlayAnimationParams,
) (api.PlayAnimationRes, error) {
	anim, err := GetAnimation(ctx, h.db, params.ID)
	if errors.Is(err, ErrNotFound) {
		return &api.PlayAnimationNotFound{Error: msgAnimationNotFound}, nil
	}
	if err != nil {
		return &api.PlayAnimationInternalServerError{
			Error: fmt.Sprintf("failed to get animation: %v", err),
		}, nil
	}

	StartDeviceAnimation(req.DeviceLocation, anim.ID, anim.Frames)

	return &api.PlayAnimationResponse{
		Message:  "Animation started successfully",
		Playback: savedPlayback(anim),
	}, nil
}

func (h *APIHandler) StopAnimation(_ context.Context, req *api.StopAnimationRequest) (api.StopAnimationRes, error) {
	StopDeviceAnimation(req.DeviceLocation)
	return &api.StopAnimationResponse{
		Message:  "Animation stopped successfully",
		Playback: api.OptNilPlayback{Set: true, Null: true},
	}, nil
}

func (h *APIHandler) PowerOn(_ context.Context, req *api.PowerOnRequest) (api.PowerOnRes, error) {
	device := &DeviceInfo{Location: req.DeviceLocation}
	if err := SetPower(device, "on"); err != nil {
		return &api.PowerOnInternalServerError{
			Error: fmt.Sprintf("failed to power on the device: %v", err),
		}, nil
	}
	SetTrackedPower(req.DeviceLocation, PowerOn)
	return &api.PowerOnNoContent{}, nil
}

func (h *APIHandler) PowerOff(_ context.Context, req *api.PowerOffRequest) (api.PowerOffRes, error) {
	// Stop first: the next animation frame would wake the cube right back up.
	StopDeviceAnimation(req.DeviceLocation)
	device := &DeviceInfo{Location: req.DeviceLocation}
	if err := SetPower(device, "off"); err != nil {
		return &api.PowerOffInternalServerError{
			Error: fmt.Sprintf("failed to power off the device: %v", err),
		}, nil
	}
	SetTrackedPower(req.DeviceLocation, PowerOff)
	return &api.PowerOffNoContent{}, nil
}

func (h *APIHandler) SaveAnimation(ctx context.Context, req *api.SaveAnimationRequest) (api.SaveAnimationRes, error) {
	frames := make([][]Color, len(req.Frames))
	for i, apiFrame := range req.Frames {
		frames[i] = ConvertAPIFrameToColors(apiFrame)
	}

	animation, err := SaveAnimation(ctx, h.db, req.DeviceID, req.Name, frames)
	if errors.Is(err, ErrNameTaken) {
		return &api.SaveAnimationConflict{Error: ErrNameTaken.Error()}, nil
	}
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
		return &api.GetAnimationNotFound{Error: msgAnimationNotFound}, nil
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
		return &api.UpdateAnimationNotFound{Error: msgAnimationNotFound}, nil
	}
	if errors.Is(err, ErrNameTaken) {
		return &api.UpdateAnimationConflict{Error: ErrNameTaken.Error()}, nil
	}
	if err != nil {
		return &api.UpdateAnimationInternalServerError{
			Error: fmt.Sprintf("failed to update animation: %v", err),
		}, nil
	}

	DetachAnimation(params.ID)

	return &api.UpdateAnimationResponse{
		Message:   "Animation updated successfully",
		Animation: convertToAPIAnimation(animation),
	}, nil
}

// ImportAnimation accepts a Sparse JSON payload (already shape-validated by
// ogen) and persists it via the codec. The size cap is enforced upstream by
// importBodyLimitMiddleware in server.go; oversize requests are mapped to
// 413 there and never reach this handler.
func (h *APIHandler) ImportAnimation(
	ctx context.Context,
	req *api.ImportAnimationRequest,
	params api.ImportAnimationParams,
) (api.ImportAnimationRes, error) {
	if res := classifyImportInput(&req.Animation); res != nil {
		return res, nil
	}

	decoded, decErr := DecodeAnimation(req.Animation)
	if decErr != nil {
		return classifyDecodeError(decErr), nil
	}

	mode := mapImportMode(params.Mode)
	persisted, persistErr := PersistImported(ctx, h.db, req.DeviceID, decoded, mode)
	if persistErr != nil {
		return classifyPersistError(persistErr), nil
	}
	if mode == ImportModeOverwrite {
		// Overwrite keeps the existing row's id but replaces its frames.
		DetachAnimation(persisted.ID)
	}

	response := &api.ImportAnimationResponse{
		Animation: convertToAPIAnimation(persisted),
	}
	if persisted.Name != decoded.Name {
		response.RenamedFrom = api.NewOptString(decoded.Name)
	}
	return response, nil
}

// classifyImportInput runs the cross-field bounds check the ogen schema
// can't express. Returns nil when the payload is acceptable, or an
// *api.ImportError ready to be returned to the client.
func classifyImportInput(anim *api.SparseAnimation) api.ImportAnimationRes {
	verr := ValidatePixelBounds(anim)
	if verr == nil {
		return nil
	}
	return &api.ImportError{
		Field:  verr.Field,
		Reason: verr.Reason,
	}
}

// classifyDecodeError maps a codec-decode failure to a sanitized
// ImportError. Only version-shape problems carry through their detail; any
// other decode failure is reported with a generic reason to avoid leaking
// internal context.
func classifyDecodeError(err error) api.ImportAnimationRes {
	if unkVer, ok := errors.AsType[*UnknownMajorVersionError](err); ok {
		return &api.ImportError{
			Field:  "animation.version",
			Reason: unkVer.Error(),
		}
	}
	if malformedVer, ok := errors.AsType[*MalformedVersionError](err); ok {
		return &api.ImportError{
			Field:  "animation.version",
			Reason: malformedVer.Error(),
		}
	}
	return &api.ImportError{
		Field:  "",
		Reason: "decode failed",
	}
}

// classifyPersistError handles codec.PersistImported errors: name conflicts
// become 409 NameConflict bodies; everything else is logged and surfaced as
// a 500 with a sanitized message.
func classifyPersistError(err error) api.ImportAnimationRes {
	if conflict, ok := errors.AsType[*NameConflictError](err); ok {
		existingUUID, _ := uuid.Parse(conflict.ExistingID)
		return &api.NameConflict{
			ExistingID:   existingUUID,
			ExistingName: conflict.Name,
		}
	}
	slog.Error("import persist failed", "error", err)
	return &api.ImportAnimationInternalServerError{
		Error: fmt.Sprintf("failed to import animation: %v", err),
	}
}

// mapImportMode maps the optional ogen-decoded mode parameter to the
// internal ImportMode enum, defaulting to ImportModeRename when unset.
func mapImportMode(opt api.OptImportAnimationMode) ImportMode {
	if !opt.IsSet() {
		return ImportModeRename
	}
	switch opt.Value {
	case api.ImportAnimationModeRename:
		return ImportModeRename
	case api.ImportAnimationModeOverwrite:
		return ImportModeOverwrite
	case api.ImportAnimationModeCancel:
		return ImportModeCancel
	default:
		return ImportModeRename
	}
}

// ExportAnimation is wired into ogen's Handler interface but is not the live
// path: GET /api/animation/{id}/export is served by a dedicated, non-ogen
// HTTP route mounted in server.go (see exportAnimationRoute) so the response
// can carry a Content-Disposition: attachment header keyed off the
// animation's name. Ogen's typed-handler signature does not expose the raw
// [http.ResponseWriter] to the handler, and the simplest "set the header from
// the typed handler" workarounds (ResponseWriter-via-context, custom encoder)
// produced more glue than the entire feature does. The non-ogen route owns
// the whole request; this method exists only to satisfy api.Handler at
// compile time and returns 500 if it is ever reached, which would indicate
// the export route override regressed.
func (h *APIHandler) ExportAnimation(
	_ context.Context,
	_ api.ExportAnimationParams,
) (api.ExportAnimationRes, error) {
	slog.Error("ExportAnimation reached the ogen handler; the non-ogen route override is missing")
	return &api.ExportAnimationInternalServerError{
		Error: "export route misconfigured",
	}, nil
}

func (h *APIHandler) DeleteAnimation(
	ctx context.Context,
	params api.DeleteAnimationParams,
) (api.DeleteAnimationRes, error) {
	err := DeleteAnimation(ctx, h.db, params.ID)
	if errors.Is(err, ErrNotFound) {
		return &api.DeleteAnimationNotFound{Error: msgAnimationNotFound}, nil
	}
	if err != nil {
		return &api.DeleteAnimationInternalServerError{
			Error: fmt.Sprintf("failed to delete animation: %v", err),
		}, nil
	}

	DetachAnimation(params.ID)

	return &api.DeleteAnimationResponse{Message: "Animation deleted successfully"}, nil
}

func convertToAPIFrames(frames [][]Color) []api.AnimationFrame {
	apiFrames := make([]api.AnimationFrame, len(frames))
	for i, frame := range frames {
		apiFrames[i] = make(api.AnimationFrame, len(frame))
		for j, color := range frame {
			apiFrames[i][j] = api.RGBPixel{
				R: int32(color.R),
				G: int32(color.G),
				B: int32(color.B),
			}
		}
	}
	return apiFrames
}

func convertToAPIAnimation(anim *SavedAnimation) api.SavedAnimation {
	return api.SavedAnimation{
		ID:        anim.ID,
		DeviceID:  anim.DeviceID,
		Name:      anim.Name,
		Frames:    convertToAPIFrames(anim.Frames),
		CreatedAt: anim.CreatedAt,
		UpdatedAt: anim.UpdatedAt,
	}
}
