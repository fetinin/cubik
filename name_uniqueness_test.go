package main

import (
	"bytes"
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"net/http"
	"testing"

	"github.com/golang-migrate/migrate/v4"
	"github.com/golang-migrate/migrate/v4/database/sqlite"
	"github.com/golang-migrate/migrate/v4/source/iofs"
)

func TestSaveAnimation_DuplicateNameReturnsErrNameTaken(t *testing.T) {
	db := setupTestDB(t)
	ctx := context.Background()
	frames := [][]Color{{{R: 1}}}

	if _, err := SaveAnimation(ctx, db, "dev", "Dup", frames); err != nil {
		t.Fatalf("first save failed: %v", err)
	}
	if _, err := SaveAnimation(ctx, db, "dev", "Dup", frames); !errors.Is(err, ErrNameTaken) {
		t.Fatalf("second save err = %v, want ErrNameTaken", err)
	}
	if _, err := SaveAnimation(ctx, db, "other-dev", "Dup", frames); err != nil {
		t.Fatalf("same name on another device should succeed, got: %v", err)
	}
}

func TestUpdateAnimation_RenameToTakenNameReturnsErrNameTaken(t *testing.T) {
	db := setupTestDB(t)
	ctx := context.Background()
	frames := [][]Color{{{R: 1}}}

	if _, err := SaveAnimation(ctx, db, "dev", "Taken", frames); err != nil {
		t.Fatalf("seed failed: %v", err)
	}
	other, err := SaveAnimation(ctx, db, "dev", "Other", frames)
	if err != nil {
		t.Fatalf("seed failed: %v", err)
	}

	if _, updErr := UpdateAnimation(ctx, db, other.ID, "Taken", frames); !errors.Is(updErr, ErrNameTaken) {
		t.Fatalf("update err = %v, want ErrNameTaken", updErr)
	}
	// Keeping its own name must still work.
	if _, updErr := UpdateAnimation(ctx, db, other.ID, "Other", frames); updErr != nil {
		t.Fatalf("update with unchanged name failed: %v", updErr)
	}
}

// TestUniqueNameMigration_RenamesExistingDuplicates applies migration 1,
// seeds duplicates the old schema allowed, then applies migration 2 and
// checks the oldest row keeps its name while the rest get a suffix.
func TestUniqueNameMigration_RenamesExistingDuplicates(t *testing.T) {
	db, err := sql.Open("sqlite", ":memory:")
	if err != nil {
		t.Fatalf("open: %v", err)
	}
	db.SetMaxOpenConns(1)
	t.Cleanup(func() { db.Close() })

	src, err := iofs.New(migrationsFS, "migrations")
	if err != nil {
		t.Fatalf("migration source: %v", err)
	}
	drv, err := sqlite.WithInstance(db, &sqlite.Config{})
	if err != nil {
		t.Fatalf("migration driver: %v", err)
	}
	m, err := migrate.NewWithInstance("iofs", src, "sqlite", drv)
	if err != nil {
		t.Fatalf("migrate instance: %v", err)
	}
	if stepErr := m.Steps(1); stepErr != nil {
		t.Fatalf("migration 1: %v", stepErr)
	}

	ctx := context.Background()
	for _, id := range []string{"aaaaaaaa-1", "bbbbbbbb-2", "cccccccc-3"} {
		_, execErr := db.ExecContext(ctx,
			`INSERT INTO saved_animations (id, device_id, name, frames_json, created_at, updated_at)
			 VALUES (?, 'dev', 'Dup', '[]', '2026-01-01T00:00:00Z', '2026-01-01T00:00:00Z')`, id)
		if execErr != nil {
			t.Fatalf("seed %s: %v", id, execErr)
		}
	}

	if stepErr := m.Steps(1); stepErr != nil {
		t.Fatalf("migration 2: %v", stepErr)
	}

	want := map[string]string{
		"aaaaaaaa-1": "Dup",
		"bbbbbbbb-2": "Dup (bbbbbbbb)",
		"cccccccc-3": "Dup (cccccccc)",
	}
	for id, wantName := range want {
		var name string
		if scanErr := db.QueryRowContext(ctx, `SELECT name FROM saved_animations WHERE id = ?`, id).
			Scan(&name); scanErr != nil {
			t.Fatalf("query %s: %v", id, scanErr)
		}
		if name != wantName {
			t.Errorf("row %s name = %q, want %q", id, name, wantName)
		}
	}
}

func TestSaveAndUpdateEndpoints_DuplicateNameReturns409(t *testing.T) {
	srv, db := importTestServer(t)
	ctx := context.Background()
	frames := [][]Color{{{R: 1}}}

	if _, err := SaveAnimation(ctx, db, "dev", "Taken", frames); err != nil {
		t.Fatalf("seed failed: %v", err)
	}
	other, err := SaveAnimation(ctx, db, "dev", "Other", frames)
	if err != nil {
		t.Fatalf("seed failed: %v", err)
	}

	apiFrames := []any{[]any{map[string]any{"r": 1, "g": 0, "b": 0}}}

	saveBody, _ := json.Marshal(map[string]any{"device_id": "dev", "name": "Taken", "frames": apiFrames})
	resp, err := http.Post(srv.URL+"/api/animation/save", "application/json", bytes.NewReader(saveBody))
	if err != nil {
		t.Fatalf("save POST failed: %v", err)
	}
	resp.Body.Close()
	if resp.StatusCode != http.StatusConflict {
		t.Errorf("save status = %d, want 409", resp.StatusCode)
	}

	updBody, _ := json.Marshal(map[string]any{"name": "Taken", "frames": apiFrames})
	req, err := http.NewRequestWithContext(
		ctx,
		http.MethodPut,
		srv.URL+"/api/animation/"+other.ID,
		bytes.NewReader(updBody),
	)
	if err != nil {
		t.Fatalf("build PUT: %v", err)
	}
	req.Header.Set("Content-Type", "application/json")
	resp, err = http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("update PUT failed: %v", err)
	}
	resp.Body.Close()
	if resp.StatusCode != http.StatusConflict {
		t.Errorf("update status = %d, want 409", resp.StatusCode)
	}

	if got := countAnimations(t, db); got != 2 {
		t.Errorf("expected 2 rows, got %d", got)
	}
}
