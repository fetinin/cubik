package main

import (
	"context"
	"cubik/api"
	"database/sql"
	"errors"
	"fmt"
	"strconv"
	"strings"
)

// The plan for T-codec specifies the constant names exactly as
// CURRENT_FORMAT_MAJOR / CURRENT_FORMAT_VERSION; tests reference them by
// those names, so we suppress the all-caps naming lint here.

// CURRENT_FORMAT_MAJOR is the major version of the Sparse JSON wire format
// supported by this codec. Decoding rejects payloads with a different major.
//
//nolint:revive,staticcheck // name fixed by T-codec plan
const CURRENT_FORMAT_MAJOR = 1

// CURRENT_FORMAT_VERSION is the full "<major>.<minor>" version stamp used
// when encoding a fresh payload.
//
//nolint:revive,staticcheck // name fixed by T-codec plan
const CURRENT_FORMAT_VERSION = "1.0"

// MatrixWidth and MatrixHeight are the LED dimensions of the only supported
// device today (Yeelight CubeLite). The encoder needs these to flatten the
// row-major frame buffer into Sparse {x, y} coordinates; they live here next
// to the other format-side constants so both the codec and any handler that
// orchestrates the codec can find them in one place.
const (
	MatrixWidth  = 20
	MatrixHeight = 5
)

// ImportMode controls the behaviour of PersistImported when the incoming
// animation collides with an existing record by name on the same device.
type ImportMode string

const (
	// ImportModeRename appends " (2)", " (3)", ... until a free name is found
	// and inserts a fresh record. This is the default conflict resolution.
	ImportModeRename ImportMode = "rename"
	// ImportModeOverwrite replaces the frames of the existing record (matched
	// by name) and bumps updated_at. The ID is preserved.
	ImportModeOverwrite ImportMode = "overwrite"
	// ImportModeCancel aborts the import on collision and returns a typed
	// *NameConflictError without touching the database.
	ImportModeCancel ImportMode = "cancel"
)

// UnknownMajorVersionError is returned by DecodeAnimation when the payload's
// major version does not match CURRENT_FORMAT_MAJOR.
type UnknownMajorVersionError struct {
	GotMajor  int
	WantMajor int
}

func (e *UnknownMajorVersionError) Error() string {
	return fmt.Sprintf("unknown major version %d; supported major: %d", e.GotMajor, e.WantMajor)
}

// MalformedVersionError is returned by DecodeAnimation when the version field
// is not in the expected "<int>.<int>" shape. The OpenAPI validator should
// catch this earlier in the HTTP path; the codec is defensive for callers
// that bypass validation.
type MalformedVersionError struct {
	Version string
}

func (e *MalformedVersionError) Error() string {
	return fmt.Sprintf("malformed version %q: expected <major>.<minor>", e.Version)
}

// NameConflictError is returned by PersistImported in ImportModeCancel when
// an animation with the same name already exists for the device.
type NameConflictError struct {
	Name       string
	ExistingID string
}

func (e *NameConflictError) Error() string {
	return fmt.Sprintf("animation name conflict: %q", e.Name)
}

// EncodeAnimation converts a SavedAnimation into the Sparse JSON wire form.
// Frames are flat row-major arrays of length width*height; only non-black
// pixels are emitted. The caller supplies width/height (CubeLite uses 20x5).
func EncodeAnimation(saved *SavedAnimation, width, height int) (api.SparseAnimation, error) {
	if saved == nil {
		return api.SparseAnimation{}, errors.New("encode: nil animation")
	}
	if width <= 0 || height <= 0 {
		return api.SparseAnimation{}, fmt.Errorf("encode: invalid dimensions %dx%d", width, height)
	}
	expected := width * height

	out := api.SparseAnimation{
		Version: CURRENT_FORMAT_VERSION,
		Name:    saved.Name,
		Width:   int32(width),
		Height:  int32(height),
		Frames:  make([]api.SparseFrame, len(saved.Frames)),
	}

	for fi, frame := range saved.Frames {
		if len(frame) != expected {
			return api.SparseAnimation{}, fmt.Errorf(
				"encode: frame %d has %d pixels, expected %d",
				fi, len(frame), expected,
			)
		}
		// Pre-allocate zero-length so an all-black frame round-trips as [].
		sparse := make(api.SparseFrame, 0)
		for idx, c := range frame {
			if c.R == 0 && c.G == 0 && c.B == 0 {
				continue
			}
			x := idx % width
			y := idx / width
			packed := (int32(c.R) << 16) | (int32(c.G) << 8) | int32(c.B)
			sparse = append(sparse, api.SparsePixel{
				X: int32(x),
				Y: int32(y),
				C: packed,
			})
		}
		out.Frames[fi] = sparse
	}
	return out, nil
}

// DecodeAnimation converts a Sparse JSON payload into the internal
// representation. It enforces major-version compatibility (rejects unknown
// majors with *UnknownMajorVersionError) and tolerates unknown fields within
// the same major (those are silently dropped by the ogen decoder upstream).
//
// The returned SavedAnimation has Name set; ID, DeviceID, CreatedAt,
// UpdatedAt are left zero — the persist step fills them in.
func DecodeAnimation(payload api.SparseAnimation) (*SavedAnimation, error) {
	major, err := parseMajor(payload.Version)
	if err != nil {
		return nil, err
	}
	if major != CURRENT_FORMAT_MAJOR {
		return nil, &UnknownMajorVersionError{GotMajor: major, WantMajor: CURRENT_FORMAT_MAJOR}
	}

	width := int(payload.Width)
	height := int(payload.Height)
	if width <= 0 || height <= 0 {
		return nil, fmt.Errorf("decode: invalid dimensions %dx%d", width, height)
	}
	total := width * height

	frames := make([][]Color, len(payload.Frames))
	for fi, sparse := range payload.Frames {
		frame := make([]Color, total)
		for _, px := range sparse {
			x := int(px.X)
			y := int(px.Y)
			if x < 0 || x >= width || y < 0 || y >= height {
				return nil, fmt.Errorf(
					"decode: frame %d pixel out of bounds (x=%d y=%d, dims=%dx%d)",
					fi,
					x,
					y,
					width,
					height,
				)
			}
			c := uint32(px.C)
			frame[y*width+x] = Color{
				R: uint8((c >> 16) & 0xFF),
				G: uint8((c >> 8) & 0xFF),
				B: uint8(c & 0xFF),
			}
		}
		frames[fi] = frame
	}

	return &SavedAnimation{
		Name:   payload.Name,
		Frames: frames,
	}, nil
}

// parseMajor splits "<major>.<minor>" and returns the integer major.
func parseMajor(version string) (int, error) {
	idx := strings.IndexByte(version, '.')
	if idx <= 0 || idx == len(version)-1 {
		return 0, &MalformedVersionError{Version: version}
	}
	majorStr := version[:idx]
	minorStr := version[idx+1:]
	major, err := strconv.Atoi(majorStr)
	if err != nil {
		return 0, &MalformedVersionError{Version: version}
	}
	if _, minorErr := strconv.Atoi(minorStr); minorErr != nil {
		return 0, &MalformedVersionError{Version: version}
	}
	return major, nil
}

// PersistImported inserts (or updates) an imported animation atomically.
// Concurrent writers can't slip a duplicate past the collision check: SQLite
// fails the loser's lock upgrade ("database table is locked"), and the
// UNIQUE(device_id, name) index is the backstop invariant (ErrNameTaken).
//
// Modes:
//   - ImportModeRename     append " (2)", " (3)", ... until free, then INSERT.
//   - ImportModeOverwrite  UPDATE the existing record (by name) in place.
//   - ImportModeCancel     return *NameConflictError; DB untouched.
//
// On success returns the persisted record (with ID and timestamps populated).
func PersistImported(
	ctx context.Context,
	db *sql.DB,
	deviceID string,
	anim *SavedAnimation,
	mode ImportMode,
) (*SavedAnimation, error) {
	if anim == nil {
		return nil, errors.New("persist: nil animation")
	}
	if mode == "" {
		mode = ImportModeRename
	}

	tx, err := db.BeginTx(ctx, nil)
	if err != nil {
		return nil, fmt.Errorf("persist: begin tx: %w", err)
	}
	// Roll back unless we explicitly commit. tx.Rollback after commit is a
	// no-op error we can swallow.
	committed := false
	defer func() {
		if !committed {
			_ = tx.Rollback()
		}
	}()

	existing, lookupErr := AnimationByName(ctx, tx, deviceID, anim.Name)
	hasCollision := lookupErr == nil
	if lookupErr != nil && !errors.Is(lookupErr, ErrNotFound) {
		return nil, lookupErr
	}

	var result *SavedAnimation
	switch {
	case !hasCollision:
		saved, saveErr := SaveAnimation(ctx, tx, deviceID, anim.Name, anim.Frames)
		if saveErr != nil {
			return nil, saveErr
		}
		result = saved

	case mode == ImportModeRename:
		freeName, findErr := findFreeName(ctx, tx, deviceID, anim.Name)
		if findErr != nil {
			return nil, findErr
		}
		saved, saveErr := SaveAnimation(ctx, tx, deviceID, freeName, anim.Frames)
		if saveErr != nil {
			return nil, saveErr
		}
		result = saved

	case mode == ImportModeOverwrite:
		updated, updErr := UpdateAnimation(ctx, tx, existing.ID, existing.Name, anim.Frames)
		if updErr != nil {
			return nil, updErr
		}
		result = updated

	case mode == ImportModeCancel:
		return nil, &NameConflictError{Name: existing.Name, ExistingID: existing.ID}

	default:
		return nil, fmt.Errorf("persist: unknown import mode %q", mode)
	}

	if commitErr := tx.Commit(); commitErr != nil {
		return nil, fmt.Errorf("persist: commit: %w", commitErr)
	}
	committed = true
	return result, nil
}

// findFreeName tries "<base> (2)", "<base> (3)", ... until it finds one not
// taken on the given device. Bounded loop to avoid pathological cases.
func findFreeName(ctx context.Context, db DBTX, deviceID, base string) (string, error) {
	const maxAttempts = 10000
	for n := 2; n < maxAttempts; n++ {
		candidate := fmt.Sprintf("%s (%d)", base, n)
		taken, err := NameExists(ctx, db, deviceID, candidate)
		if err != nil {
			return "", err
		}
		if !taken {
			return candidate, nil
		}
	}
	return "", fmt.Errorf("persist: could not find free rename suffix for %q after %d attempts", base, maxAttempts)
}
