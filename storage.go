package main

import (
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
			jsonFrame[j] = FrameJSON{R: color.R, G: color.G, B: color.B}
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
				frame[j] = Color{R: pixel.R, G: pixel.G, B: pixel.B}
			}
			frames[i] = frame
		}
	}
	return frames, nil
}

func SaveAnimation(db *sql.DB, deviceID, name string, frames [][]Color) (*SavedAnimation, error) {
	id := uuid.New().String()
	framesJSON, err := serializeFrames(frames)
	if err != nil {
		return nil, err
	}

	now := time.Now().UTC()
	timestamp := now.Format(time.RFC3339)

	_, err = db.Exec(
		`INSERT INTO saved_animations (id, device_id, name, frames_json, created_at, updated_at)
		 VALUES (?, ?, ?, ?, ?, ?)`,
		id, deviceID, name, framesJSON, timestamp, timestamp,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to insert animation: %w", err)
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

func GetAnimation(db *sql.DB, id string) (*SavedAnimation, error) {
	var deviceID, name, framesJSON, createdAt, updatedAt string

	err := db.QueryRow(
		`SELECT device_id, name, frames_json, created_at, updated_at
		 FROM saved_animations WHERE id = ?`,
		id,
	).Scan(&deviceID, &name, &framesJSON, &createdAt, &updatedAt)

	if err == sql.ErrNoRows {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("failed to query animation: %w", err)
	}

	frames, err := deserializeFrames(framesJSON)
	if err != nil {
		return nil, err
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

func ListAnimationsByDevice(db *sql.DB, deviceID string) ([]*SavedAnimation, error) {
	rows, err := db.Query(
		`SELECT id, name, frames_json, created_at, updated_at
		 FROM saved_animations WHERE device_id = ? ORDER BY updated_at DESC`,
		deviceID,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to query animations: %w", err)
	}
	defer rows.Close()

	var animations []*SavedAnimation
	for rows.Next() {
		var id, name, framesJSON, createdAt, updatedAt string
		if err := rows.Scan(&id, &name, &framesJSON, &createdAt, &updatedAt); err != nil {
			return nil, fmt.Errorf("failed to scan row: %w", err)
		}

		frames, err := deserializeFrames(framesJSON)
		if err != nil {
			return nil, err
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

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating rows: %w", err)
	}
	return animations, nil
}

func UpdateAnimation(db *sql.DB, id, name string, frames [][]Color) (*SavedAnimation, error) {
	framesJSON, err := serializeFrames(frames)
	if err != nil {
		return nil, err
	}

	updatedAt := time.Now().UTC().Format(time.RFC3339)
	result, err := db.Exec(
		`UPDATE saved_animations SET name = ?, frames_json = ?, updated_at = ? WHERE id = ?`,
		name, framesJSON, updatedAt, id,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to update animation: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return nil, fmt.Errorf("failed to get rows affected: %w", err)
	}
	if rowsAffected == 0 {
		return nil, ErrNotFound
	}

	return GetAnimation(db, id)
}

func DeleteAnimation(db *sql.DB, id string) error {
	result, err := db.Exec(`DELETE FROM saved_animations WHERE id = ?`, id)
	if err != nil {
		return fmt.Errorf("failed to delete animation: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}
	if rowsAffected == 0 {
		return ErrNotFound
	}
	return nil
}
