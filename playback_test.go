package main

import (
	"context"
	"testing"
)

// putRunningAnimation registers a fake playback without starting a goroutine
// or touching the network.
func putRunningAnimation(t *testing.T, location, animationID string, frames [][]Color) {
	t.Helper()
	animationsMu.Lock()
	runningAnimations[location] = &AnimationState{
		DeviceLocation: location,
		AnimationID:    animationID,
		Frames:         frames,
		StopFunc:       func() {},
	}
	animationsMu.Unlock()
	t.Cleanup(func() {
		animationsMu.Lock()
		delete(runningAnimations, location)
		animationsMu.Unlock()
	})
}

// TestDetachAnimation_OnlyMatchingIDs verifies that only playbacks of the given animation are unlinked.
func TestDetachAnimation_OnlyMatchingIDs(t *testing.T) {
	putRunningAnimation(t, "yeelight://10.0.0.1:55443", "anim-a", nil)
	putRunningAnimation(t, "yeelight://10.0.0.2:55443", "anim-b", nil)
	putRunningAnimation(t, "yeelight://10.0.0.3:55443", "anim-a", nil)

	DetachAnimation("anim-a")

	for location, want := range map[string]string{
		"yeelight://10.0.0.1:55443": "",
		"yeelight://10.0.0.2:55443": "anim-b",
		"yeelight://10.0.0.3:55443": "",
	} {
		got, _, ok := PlaybackSnapshot(location)
		if !ok {
			t.Fatalf("PlaybackSnapshot(%s) found nothing", location)
		}
		if got != want {
			t.Errorf("AnimationID for %s = %q, want %q", location, got, want)
		}
	}
}

// TestTrackedPower_DefaultsToUnknown verifies the power state before and after it is set.
func TestTrackedPower_DefaultsToUnknown(t *testing.T) {
	location := "yeelight://10.0.0.9:55443"
	t.Cleanup(func() {
		animationsMu.Lock()
		delete(devicePower, location)
		animationsMu.Unlock()
	})

	if got := TrackedPower(location); got != PowerUnknown {
		t.Errorf("TrackedPower = %q, want %q", got, PowerUnknown)
	}
	SetTrackedPower(location, PowerOff)
	if got := TrackedPower(location); got != PowerOff {
		t.Errorf("TrackedPower = %q, want %q", got, PowerOff)
	}
}

// TestBuildPlayback_SavedAnimation verifies that a saved playback carries its name and no frames.
func TestBuildPlayback_SavedAnimation(t *testing.T) {
	db := setupTestDB(t)
	ctx := context.Background()
	frames := [][]Color{{{R: 255}}}
	anim, err := SaveAnimation(ctx, db, "device-1", "Palm Island", frames)
	if err != nil {
		t.Fatalf("SaveAnimation failed: %v", err)
	}
	location := "yeelight://10.0.0.4:55443"
	putRunningAnimation(t, location, anim.ID, frames)

	h := &APIHandler{db: db}
	pb, ok := h.buildPlayback(ctx, location)
	if !ok {
		t.Fatal("buildPlayback found nothing")
	}
	if pb.AnimationName.Value != "Palm Island" || pb.AnimationID.Value != anim.ID {
		t.Errorf("playback = %+v, want saved Palm Island", pb)
	}
	if pb.Frames != nil {
		t.Errorf("saved playback has %d frames, want none", len(pb.Frames))
	}
}

// TestBuildPlayback_Unsaved verifies that unsaved playback is returned with its frames.
func TestBuildPlayback_Unsaved(t *testing.T) {
	h := &APIHandler{db: setupTestDB(t)}
	location := "yeelight://10.0.0.5:55443"
	putRunningAnimation(t, location, "", [][]Color{{{G: 10}, {B: 20}}})

	pb, ok := h.buildPlayback(context.Background(), location)
	if !ok {
		t.Fatal("buildPlayback found nothing")
	}
	if pb.AnimationID.IsSet() {
		t.Errorf("unsaved playback has id %q", pb.AnimationID.Value)
	}
	if len(pb.Frames) != 1 || pb.Frames[0][1].B != 20 {
		t.Errorf("frames = %+v, want the running frames", pb.Frames)
	}
}

// TestBuildPlayback_DeletedRow verifies that a playback whose row is gone becomes unsaved and is detached.
func TestBuildPlayback_DeletedRow(t *testing.T) {
	db := setupTestDB(t)
	ctx := context.Background()
	frames := [][]Color{{{R: 1}}}
	anim, err := SaveAnimation(ctx, db, "device-1", "Gone", frames)
	if err != nil {
		t.Fatalf("SaveAnimation failed: %v", err)
	}
	if delErr := DeleteAnimation(ctx, db, anim.ID); delErr != nil {
		t.Fatalf("DeleteAnimation failed: %v", delErr)
	}
	location := "yeelight://10.0.0.6:55443"
	putRunningAnimation(t, location, anim.ID, frames)

	h := &APIHandler{db: db}
	pb, ok := h.buildPlayback(ctx, location)
	if !ok {
		t.Fatal("buildPlayback found nothing")
	}
	if pb.AnimationID.IsSet() || len(pb.Frames) != 1 {
		t.Errorf("playback = %+v, want unsaved with frames", pb)
	}
	if id, _, _ := PlaybackSnapshot(location); id != "" {
		t.Errorf("AnimationID after lookup = %q, want detached", id)
	}
}
