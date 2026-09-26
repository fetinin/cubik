package main

import (
	"context"
	"database/sql"
	"errors"
	"testing"

	_ "modernc.org/sqlite"
)

// setupTestDB creates an in-memory SQLite database with the real migrations applied.
func setupTestDB(t *testing.T) *sql.DB {
	db, err := sql.Open("sqlite", ":memory:")
	if err != nil {
		t.Fatalf("failed to open test database: %v", err)
	}
	// Every new connection to ":memory:" is a separate empty database.
	db.SetMaxOpenConns(1)
	t.Cleanup(func() { db.Close() })

	if migrateErr := RunMigrations(db); migrateErr != nil {
		t.Fatalf("failed to run migrations: %v", migrateErr)
	}
	return db
}

// TestNameExists_NoAnimations verifies that NameExists returns false when no animations exist for the device.
func TestNameExists_NoAnimations(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	ctx := context.Background()
	exists, err := NameExists(ctx, db, "device-1", "my-animation")
	if err != nil {
		t.Fatalf("NameExists failed: %v", err)
	}
	if exists {
		t.Errorf("NameExists = true, want false")
	}
}

// TestNameExists_AfterSave verifies that NameExists returns true after SaveAnimation.
func TestNameExists_AfterSave(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	ctx := context.Background()
	deviceID := "device-1"
	animName := "my-animation"
	frames := [][]Color{
		{{R: 255, G: 0, B: 0}, {R: 0, G: 255, B: 0}},
	}

	_, err := SaveAnimation(ctx, db, deviceID, animName, frames)
	if err != nil {
		t.Fatalf("SaveAnimation failed: %v", err)
	}

	exists, err := NameExists(ctx, db, deviceID, animName)
	if err != nil {
		t.Fatalf("NameExists failed: %v", err)
	}
	if !exists {
		t.Errorf("NameExists = false, want true")
	}
}

// TestNameExists_PerDevice verifies that NameExists is scoped per device.
func TestNameExists_PerDevice(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	ctx := context.Background()
	deviceID1 := "device-1"
	deviceID2 := "device-2"
	animName := "my-animation"
	frames := [][]Color{
		{{R: 255, G: 0, B: 0}},
	}

	// Save animation on device-1
	_, err := SaveAnimation(ctx, db, deviceID1, animName, frames)
	if err != nil {
		t.Fatalf("SaveAnimation on device-1 failed: %v", err)
	}

	// Check that it exists on device-1
	exists, err := NameExists(ctx, db, deviceID1, animName)
	if err != nil {
		t.Fatalf("NameExists on device-1 failed: %v", err)
	}
	if !exists {
		t.Errorf("NameExists on device-1 = false, want true")
	}

	// Check that it does NOT exist on device-2
	exists, err = NameExists(ctx, db, deviceID2, animName)
	if err != nil {
		t.Fatalf("NameExists on device-2 failed: %v", err)
	}
	if exists {
		t.Errorf("NameExists on device-2 = true, want false (same name, different device)")
	}
}

// TestAnimationByName_NotFound verifies that AnimationByName returns ErrNotFound for missing (deviceID, name).
func TestAnimationByName_NotFound(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	ctx := context.Background()
	anim, err := AnimationByName(ctx, db, "device-1", "missing-animation")
	if err == nil {
		t.Fatalf("AnimationByName = nil error, want ErrNotFound")
	}
	if !errors.Is(err, ErrNotFound) {
		t.Errorf("AnimationByName error = %v, want ErrNotFound", err)
	}
	if anim != nil {
		t.Errorf("AnimationByName = %+v, want nil", anim)
	}
}

// TestAnimationByName_FullRecord verifies that AnimationByName returns the full SavedAnimation record.
func TestAnimationByName_FullRecord(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	ctx := context.Background()
	deviceID := "device-1"
	animName := "test-animation"
	frames := [][]Color{
		{{R: 255, G: 0, B: 0}, {R: 0, G: 255, B: 0}},
		{{R: 0, G: 0, B: 255}, {R: 255, G: 255, B: 0}},
	}

	// Save the animation
	saved, err := SaveAnimation(ctx, db, deviceID, animName, frames)
	if err != nil {
		t.Fatalf("SaveAnimation failed: %v", err)
	}

	// Fetch by name
	retrieved, err := AnimationByName(ctx, db, deviceID, animName)
	if err != nil {
		t.Fatalf("AnimationByName failed: %v", err)
	}
	if retrieved == nil {
		t.Fatalf("AnimationByName returned nil")
	}

	// Verify all fields match
	if retrieved.ID != saved.ID {
		t.Errorf("ID = %q, want %q", retrieved.ID, saved.ID)
	}
	if retrieved.DeviceID != deviceID {
		t.Errorf("DeviceID = %q, want %q", retrieved.DeviceID, deviceID)
	}
	if retrieved.Name != animName {
		t.Errorf("Name = %q, want %q", retrieved.Name, animName)
	}
	if len(retrieved.Frames) != len(frames) {
		t.Errorf("len(Frames) = %d, want %d", len(retrieved.Frames), len(frames))
	}
	// Note: timestamps are stored as strings and may lose nanosecond precision
	// when round-tripped, so we only compare up to the second
	if retrieved.CreatedAt.Unix() != saved.CreatedAt.Unix() {
		t.Errorf("CreatedAt = %v, want %v", retrieved.CreatedAt, saved.CreatedAt)
	}
	if retrieved.UpdatedAt.Unix() != saved.UpdatedAt.Unix() {
		t.Errorf("UpdatedAt = %v, want %v", retrieved.UpdatedAt, saved.UpdatedAt)
	}

	// Spot-check frame contents
	for i, frame := range retrieved.Frames {
		if len(frame) != len(frames[i]) {
			t.Errorf("frame[%d] length = %d, want %d", i, len(frame), len(frames[i]))
		}
		for j, pixel := range frame {
			if pixel != frames[i][j] {
				t.Errorf("frame[%d][%d] = {R:%d G:%d B:%d}, want {R:%d G:%d B:%d}",
					i, j, pixel.R, pixel.G, pixel.B,
					frames[i][j].R, frames[i][j].G, frames[i][j].B)
			}
		}
	}
}

// TestAnimationByName_PerDevice verifies that AnimationByName is scoped per device.
func TestAnimationByName_PerDevice(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	ctx := context.Background()
	deviceID1 := "device-1"
	deviceID2 := "device-2"
	animName := "shared-name"
	frames1 := [][]Color{{{R: 255, G: 0, B: 0}}}
	frames2 := [][]Color{{{R: 0, G: 255, B: 0}}}

	// Save animations with the same name on different devices
	anim1, err := SaveAnimation(ctx, db, deviceID1, animName, frames1)
	if err != nil {
		t.Fatalf("SaveAnimation on device-1 failed: %v", err)
	}

	anim2, err := SaveAnimation(ctx, db, deviceID2, animName, frames2)
	if err != nil {
		t.Fatalf("SaveAnimation on device-2 failed: %v", err)
	}

	// Fetch from device-1 and verify it's the right one
	retrieved1, err := AnimationByName(ctx, db, deviceID1, animName)
	if err != nil {
		t.Fatalf("AnimationByName on device-1 failed: %v", err)
	}
	if retrieved1.ID != anim1.ID {
		t.Errorf("device-1: ID = %q, want %q", retrieved1.ID, anim1.ID)
	}

	// Fetch from device-2 and verify it's the right one
	retrieved2, err := AnimationByName(ctx, db, deviceID2, animName)
	if err != nil {
		t.Fatalf("AnimationByName on device-2 failed: %v", err)
	}
	if retrieved2.ID != anim2.ID {
		t.Errorf("device-2: ID = %q, want %q", retrieved2.ID, anim2.ID)
	}

	// IDs should be different even though names are the same
	if retrieved1.ID == retrieved2.ID {
		t.Errorf("device-1 and device-2 animations have the same ID, want different")
	}
}
