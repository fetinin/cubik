package main

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
)

var ErrNotFound = errors.New("animation not found")

type SavedAnimation struct {
	ID        string
	DeviceID  string
	Name      string
	Frames    [][]Color
	CreatedAt time.Time
	UpdatedAt time.Time
}

type FrameJSON struct {
	R uint8 `json:"r"`
	G uint8 `json:"g"`
	B uint8 `json:"b"`
}

func serializeFrames(frames [][]Color) (string, error) {
	jsonFrames := make([][][]FrameJSON, len(frames))
	for i, frame := range frames {
		jsonFrame := make([]FrameJSON, len(frame))
		for j, color := range frame {
			jsonFrame[j] = FrameJSON(color)
		}
		jsonFrames[i] = [][]FrameJSON{jsonFrame}
	}

	data, err := json.Marshal(jsonFrames)
	if err != nil {
		return "", fmt.Errorf("failed to marshal frames: %w", err)
	}
	return string(data), nil
}

func deserializeFrames(jsonStr string) ([][]Color, error) {
	var jsonFrames [][][]FrameJSON
	if err := json.Unmarshal([]byte(jsonStr), &jsonFrames); err != nil {
		return nil, fmt.Errorf("failed to unmarshal frames: %w", err)
	}

	frames := make([][]Color, len(jsonFrames))
	for i, jsonFrameWrapper := range jsonFrames {
		if len(jsonFrameWrapper) > 0 {
			jsonFrame := jsonFrameWrapper[0]
			frame := make([]Color, len(jsonFrame))
			for j, pixel := range jsonFrame {
				frame[j] = Color(pixel)
			}
			frames[i] = frame
		}
	}
	return frames, nil
}

func SaveAnimation(ctx context.Context, db *sql.DB, deviceID, name string, frames [][]Color) (*SavedAnimation, error) {
	id := uuid.New().String()
	framesJSON, err := serializeFrames(frames)
	if err != nil {
		return nil, err
	}

	now := time.Now().UTC()
	timestamp := now.Format(time.RFC3339)

	_, execErr := db.ExecContext(
		ctx,
		`INSERT INTO saved_animations (id, device_id, name, frames_json, created_at, updated_at)
		 VALUES (?, ?, ?, ?, ?, ?)`,
		id, deviceID, name, framesJSON, timestamp, timestamp,
	)
	if execErr != nil {
		return nil, fmt.Errorf("failed to insert animation: %w", execErr)
	}

	return &SavedAnimation{
		ID:        id,
		DeviceID:  deviceID,
		Name:      name,
		Frames:    frames,
		CreatedAt: now,
		UpdatedAt: now,
	}, nil
}

func GetAnimation(ctx context.Context, db *sql.DB, id string) (*SavedAnimation, error) {
	var deviceID, name, framesJSON, createdAt, updatedAt string

	queryErr := db.QueryRowContext(
		ctx,
		`SELECT device_id, name, frames_json, created_at, updated_at
		 FROM saved_animations WHERE id = ?`,
		id,
	).Scan(&deviceID, &name, &framesJSON, &createdAt, &updatedAt)

	if errors.Is(queryErr, sql.ErrNoRows) {
		return nil, ErrNotFound
	}
	if queryErr != nil {
		return nil, fmt.Errorf("failed to query animation: %w", queryErr)
	}

	frames, deserializeErr := deserializeFrames(framesJSON)
	if deserializeErr != nil {
		return nil, deserializeErr
	}

	createdTime, _ := time.Parse(time.RFC3339, createdAt)
	updatedTime, _ := time.Parse(time.RFC3339, updatedAt)

	return &SavedAnimation{
		ID:        id,
		DeviceID:  deviceID,
		Name:      name,
		Frames:    frames,
		CreatedAt: createdTime,
		UpdatedAt: updatedTime,
	}, nil
}

func ListAnimationsByDevice(ctx context.Context, db *sql.DB, deviceID string) ([]*SavedAnimation, error) {
	rows, queryErr := db.QueryContext(
		ctx,
		`SELECT id, name, frames_json, created_at, updated_at
		 FROM saved_animations WHERE device_id = ? ORDER BY updated_at DESC`,
		deviceID,
	)
	if queryErr != nil {
		return nil, fmt.Errorf("failed to query animations: %w", queryErr)
	}
	defer rows.Close()

	var animations []*SavedAnimation
	for rows.Next() {
		var id, name, framesJSON, createdAt, updatedAt string
		if scanErr := rows.Scan(&id, &name, &framesJSON, &createdAt, &updatedAt); scanErr != nil {
			return nil, fmt.Errorf("failed to scan row: %w", scanErr)
		}

		frames, deserializeErr := deserializeFrames(framesJSON)
		if deserializeErr != nil {
			return nil, deserializeErr
		}

		createdTime, _ := time.Parse(time.RFC3339, createdAt)
		updatedTime, _ := time.Parse(time.RFC3339, updatedAt)

		animations = append(animations, &SavedAnimation{
			ID:        id,
			DeviceID:  deviceID,
			Name:      name,
			Frames:    frames,
			CreatedAt: createdTime,
			UpdatedAt: updatedTime,
		})
	}

	if iterErr := rows.Err(); iterErr != nil {
		return nil, fmt.Errorf("error iterating rows: %w", iterErr)
	}
	return animations, nil
}

func UpdateAnimation(ctx context.Context, db *sql.DB, id, name string, frames [][]Color) (*SavedAnimation, error) {
	framesJSON, err := serializeFrames(frames)
	if err != nil {
		return nil, err
	}

	updatedAt := time.Now().UTC().Format(time.RFC3339)
	result, execErr := db.ExecContext(
		ctx,
		`UPDATE saved_animations SET name = ?, frames_json = ?, updated_at = ? WHERE id = ?`,
		name, framesJSON, updatedAt, id,
	)
	if execErr != nil {
		return nil, fmt.Errorf("failed to update animation: %w", execErr)
	}

	rowsAffected, rowsErr := result.RowsAffected()
	if rowsErr != nil {
		return nil, fmt.Errorf("failed to get rows affected: %w", rowsErr)
	}
	if rowsAffected == 0 {
		return nil, ErrNotFound
	}

	return GetAnimation(ctx, db, id)
}

func DeleteAnimation(ctx context.Context, db *sql.DB, id string) error {
	result, execErr := db.ExecContext(ctx, `DELETE FROM saved_animations WHERE id = ?`, id)
	if execErr != nil {
		return fmt.Errorf("failed to delete animation: %w", execErr)
	}

	rowsAffected, rowsErr := result.RowsAffected()
	if rowsErr != nil {
		return fmt.Errorf("failed to get rows affected: %w", rowsErr)
	}
	if rowsAffected == 0 {
		return ErrNotFound
	}
	return nil
}
