package main

import (
	"bytes"
	"context"
	"cubik/api"
	"database/sql"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

// importTestServer wires the API middleware chain (size cap + CORS + ogen)
// against a fresh in-memory SQLite DB and returns an [httptest.Server]. It
// mirrors what StartServer mounts at /api/ — exercising the wire path means
// these tests cover validator, decoder, middleware, and ogen response
// encoding the same way a real client would.
func importTestServer(t *testing.T) (*httptest.Server, *sql.DB) {
	t.Helper()
	db := setupTestDB(t)
	apiHandler, err := buildAPIHandler(db)
	if err != nil {
		db.Close()
		t.Fatalf("buildAPIHandler failed: %v", err)
	}
	mux := http.NewServeMux()
	mux.Handle("/api/", apiHandler)
	srv := httptest.NewServer(mux)
	t.Cleanup(func() {
		srv.Close()
		db.Close()
	})
	return srv, db
}

func postImport(t *testing.T, srv *httptest.Server, body []byte, queryString string) (*http.Response, []byte) {
	t.Helper()
	url := srv.URL + "/api/animation/import"
	if queryString != "" {
		url += "?" + queryString
	}
	resp, err := http.Post(url, "application/json", bytes.NewReader(body))
	if err != nil {
		t.Fatalf("POST failed: %v", err)
	}
	respBody, readErr := io.ReadAll(resp.Body)
	resp.Body.Close()
	if readErr != nil {
		t.Fatalf("read response body failed: %v", readErr)
	}
	return resp, respBody
}

func countAnimations(t *testing.T, db *sql.DB) int {
	t.Helper()
	var n int
	err := db.QueryRowContext(context.Background(), `SELECT COUNT(*) FROM saved_animations`).Scan(&n)
	if err != nil {
		t.Fatalf("count rows failed: %v", err)
	}
	return n
}

// validImportRequestBody returns an ImportAnimationRequest body for a tiny,
// schema-clean SparseAnimation — used as the happy-path baseline.
func validImportRequestBody(deviceID, name string) []byte {
	body := map[string]any{
		"device_id": deviceID,
		"animation": map[string]any{
			"version": "1.0",
			"name":    name,
			"width":   20,
			"height":  5,
			"frames": []any{
				[]any{
					map[string]any{"x": 0, "y": 0, "c": 16711680},
					map[string]any{"x": 19, "y": 4, "c": 255},
				},
			},
		},
	}
	out, _ := json.Marshal(body)
	return out
}

// --- r1: R-import-file ----------------------------------------------------

// TestImport_HappyPath_RoundTripsThroughList verifies a valid import is
// accepted (200) and the new row shows up via ListAnimationsByDevice (the
// same query that backs GET /api/animation/list/{device_id}).
func TestImport_HappyPath_RoundTripsThroughList(t *testing.T) {
	srv, db := importTestServer(t)
	device := "0xdevice1234"

	resp, body := postImport(t, srv, validImportRequestBody(device, "Imported"), "")
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("status = %d, want 200; body=%s", resp.StatusCode, body)
	}

	var got api.ImportAnimationResponse
	if err := json.Unmarshal(body, &got); err != nil {
		t.Fatalf("decode response: %v; body=%s", err, body)
	}
	if got.Animation.Name != "Imported" {
		t.Errorf("Animation.Name = %q, want %q", got.Animation.Name, "Imported")
	}
	if got.RenamedFrom.IsSet() {
		t.Errorf("expected renamed_from absent on fresh import, got %q", got.RenamedFrom.Value)
	}

	all, listErr := ListAnimationsByDevice(context.Background(), db, device)
	if listErr != nil {
		t.Fatalf("ListAnimationsByDevice: %v", listErr)
	}
	if len(all) != 1 {
		t.Fatalf("expected 1 animation, got %d", len(all))
	}
	if all[0].Name != "Imported" {
		t.Errorf("listed name = %q, want %q", all[0].Name, "Imported")
	}
}

// --- r2: R-validate-import -----------------------------------------------

// TestImport_ValidationRejection_NoPersistence runs three malformed-input
// shapes through the wire path and asserts each is rejected with 4xx and
// the saved_animations table is untouched.
func TestImport_ValidationRejection_NoPersistence(t *testing.T) {
	cases := []struct {
		name    string
		body    []byte
		minCode int
		maxCode int
	}{
		{
			name:    "malformed JSON",
			body:    []byte(`{"device_id": "d1", "animation": {`),
			minCode: 400,
			maxCode: 499,
		},
		{
			name: "schema violation: missing frames",
			body: func() []byte {
				out, _ := json.Marshal(map[string]any{
					"device_id": "d1",
					"animation": map[string]any{
						"version": "1.0",
						"name":    "x",
						"width":   20,
						"height":  5,
						// frames intentionally omitted
					},
				})
				return out
			}(),
			minCode: 400,
			maxCode: 499,
		},
		{
			name: "oversized payload",
			body: func() []byte {
				huge := bytes.Repeat([]byte("x"), ImportPayloadMaxBytes+10)
				return huge
			}(),
			minCode: 400,
			maxCode: 499,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			srv, db := importTestServer(t)
			resp, body := postImport(t, srv, tc.body, "")
			if resp.StatusCode < tc.minCode || resp.StatusCode > tc.maxCode {
				t.Errorf("status = %d, want %d-%d; body=%s",
					resp.StatusCode, tc.minCode, tc.maxCode, body)
			}
			if got := countAnimations(t, db); got != 0 {
				t.Errorf("expected 0 rows after rejection, got %d", got)
			}
		})
	}
}

// --- r3: R-S1a-payload-size ----------------------------------------------

// TestImport_SizeCap_TwoMiBRejected confirms a 2 MiB request is rejected
// with 413 (or a 4xx from the size-cap middleware) and never produces a
// row in the database — i.e. ogen never gets to call the handler.
func TestImport_SizeCap_TwoMiBRejected(t *testing.T) {
	srv, db := importTestServer(t)

	// Build a syntactically-valid JSON document ~2 MiB by padding the name
	// field. Using JSON (not raw junk) rules out "the parser bailed early"
	// as the rejection cause; it must be the size cap that fires.
	pad := strings.Repeat("a", 2*1024*1024)
	body, _ := json.Marshal(map[string]any{
		"device_id": "device-1",
		"animation": map[string]any{
			"version": "1.0",
			"name":    pad,
			"width":   20,
			"height":  5,
			"frames":  []any{[]any{}},
		},
	})

	resp, respBody := postImport(t, srv, body, "")
	if resp.StatusCode != http.StatusRequestEntityTooLarge {
		t.Errorf("status = %d, want 413; body=%s", resp.StatusCode, respBody)
	}
	if got := countAnimations(t, db); got != 0 {
		t.Errorf("expected 0 rows, got %d", got)
	}
}

// --- r4: R-S1b-strict-schema ---------------------------------------------

// TestImport_MissingVersionField verifies the validator rejects an inner
// SparseAnimation that omits the required version field; the response
// must be a 400 with Field referencing "version".
func TestImport_MissingVersionField(t *testing.T) {
	srv, db := importTestServer(t)

	body, _ := json.Marshal(map[string]any{
		"device_id": "d1",
		"animation": map[string]any{
			// version intentionally omitted
			"name":   "x",
			"width":  20,
			"height": 5,
			"frames": []any{[]any{map[string]any{"x": 0, "y": 0, "c": 1}}},
		},
	})

	resp, respBody := postImport(t, srv, body, "")
	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("status = %d, want 400; body=%s", resp.StatusCode, respBody)
	}
	// The 400 body should be the ImportError shape OR the ogen default
	// decoder error envelope ({"error_message": "..."}). Either form must
	// mention "version" so the client can act on it.
	if !strings.Contains(strings.ToLower(string(respBody)), "version") {
		t.Errorf("expected response body to mention 'version'; got %s", respBody)
	}
	if got := countAnimations(t, db); got != 0 {
		t.Errorf("expected 0 rows, got %d", got)
	}
}

// --- r5: R-S1c-value-bounds ----------------------------------------------

// TestImport_ValueBounds_RejectsOutOfRangePixels covers x>=width, y>=height,
// and color outside 0..0xFFFFFF, plus a happy edge-of-range case that
// must succeed.
func TestImport_ValueBounds_RejectsOutOfRangePixels(t *testing.T) {
	mkAnim := func(x, y, c int32, w, h int32) []byte {
		out, _ := json.Marshal(map[string]any{
			"device_id": "d1",
			"animation": map[string]any{
				"version": "1.0",
				"name":    "bounds",
				"width":   w,
				"height":  h,
				"frames":  []any{[]any{map[string]any{"x": x, "y": y, "c": c}}},
			},
		})
		return out
	}

	cases := []struct {
		name             string
		body             []byte
		wantCode         int
		wantFieldHint    string
		expectAnyMessage string
	}{
		{
			name:          "x out of range",
			body:          mkAnim(20, 0, 1, 20, 5),
			wantCode:      400,
			wantFieldHint: "x",
		},
		{
			name:          "y out of range",
			body:          mkAnim(0, 5, 1, 20, 5),
			wantCode:      400,
			wantFieldHint: "y",
		},
		{
			name:     "color above 0xFFFFFF",
			body:     mkAnim(0, 0, 0xFFFFFF+1, 20, 5),
			wantCode: 400,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			srv, db := importTestServer(t)
			resp, body := postImport(t, srv, tc.body, "")
			if resp.StatusCode != tc.wantCode {
				t.Errorf("status = %d, want %d; body=%s", resp.StatusCode, tc.wantCode, body)
			}
			if tc.wantFieldHint != "" {
				if !strings.Contains(string(body), tc.wantFieldHint) {
					t.Errorf("expected body to mention %q; got %s", tc.wantFieldHint, body)
				}
			}
			if got := countAnimations(t, db); got != 0 {
				t.Errorf("expected 0 rows, got %d", got)
			}
		})
	}

	// Edge-of-range happy case: x=width-1, y=height-1, c=0xFFFFFF must accept.
	t.Run("edge of range accepted", func(t *testing.T) {
		srv, _ := importTestServer(t)
		body := mkAnim(19, 4, 0xFFFFFF, 20, 5)
		resp, respBody := postImport(t, srv, body, "")
		if resp.StatusCode != http.StatusOK {
			t.Errorf("edge-of-range case rejected: status=%d body=%s", resp.StatusCode, respBody)
		}
	})
}

// --- r6: R-S1d-actionable-errors -----------------------------------------

// TestImport_ErrorBodyShapeAndSanitization triggers a cross-field bounds
// failure (which goes through our own ImportError encoding, not ogen's
// fallback) and verifies the response decodes as ImportError without
// leaking server-internal artifacts.
func TestImport_ErrorBodyShapeAndSanitization(t *testing.T) {
	srv, _ := importTestServer(t)

	body, _ := json.Marshal(map[string]any{
		"device_id": "d1",
		"animation": map[string]any{
			"version": "1.0",
			"name":    "bounds",
			"width":   10,
			"height":  10,
			// pixel x=15 is past width=10 — handled by ValidatePixelBounds
			"frames": []any{[]any{map[string]any{"x": 15, "y": 0, "c": 1}}},
		},
	})

	resp, respBody := postImport(t, srv, body, "")
	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("status = %d, want 400; body=%s", resp.StatusCode, respBody)
	}

	var ie api.ImportError
	if err := json.Unmarshal(respBody, &ie); err != nil {
		t.Fatalf("response body not ImportError-shaped: %v; body=%s", err, respBody)
	}
	if ie.Field == "" {
		t.Errorf("Field should be populated for cross-field bounds failure; body=%s", respBody)
	}
	if ie.Reason == "" {
		t.Errorf("Reason should be populated; body=%s", respBody)
	}

	// Sanitization: no stack traces, no absolute file paths, no SQL.
	bodyStr := string(respBody)
	for _, leak := range []string{"/Users/", "goroutine ", "panic", "SELECT ", "INSERT ", "UPDATE ", "/cubik/"} {
		if strings.Contains(bodyStr, leak) {
			t.Errorf("response leaks server detail %q; body=%s", leak, respBody)
		}
	}
}

// --- r7: R-S2a-collision-detect ------------------------------------------

// TestImport_CancelMode_ReturnsConflict seeds an animation, posts an import
// with the same name and mode=cancel, and verifies a 409 NameConflict body
// pointing at the seeded row. The DB is left untouched.
func TestImport_CancelMode_ReturnsConflict(t *testing.T) {
	srv, db := importTestServer(t)
	device := "device-collide"

	seeded, seedErr := SaveAnimation(context.Background(), db, device, "Collide", [][]Color{{{R: 1}}})
	if seedErr != nil {
		t.Fatalf("seed failed: %v", seedErr)
	}

	resp, respBody := postImport(t, srv, validImportRequestBody(device, "Collide"), "mode=cancel")
	if resp.StatusCode != http.StatusConflict {
		t.Fatalf("status = %d, want 409; body=%s", resp.StatusCode, respBody)
	}

	var nc api.NameConflict
	if decErr := json.Unmarshal(respBody, &nc); decErr != nil {
		t.Fatalf("response not NameConflict-shaped: %v; body=%s", decErr, respBody)
	}
	if nc.ExistingID.String() != seeded.ID {
		t.Errorf("ExistingID = %q, want %q", nc.ExistingID, seeded.ID)
	}
	if nc.ExistingName != "Collide" {
		t.Errorf("ExistingName = %q, want %q", nc.ExistingName, "Collide")
	}
	if got := countAnimations(t, db); got != 1 {
		t.Errorf("expected 1 row (untouched), got %d", got)
	}
}

// --- r8: R-S2b-resolution-mode -------------------------------------------

// TestImport_ResolutionModes covers each mode against a pre-seeded
// collision plus the garbage-mode case.
func TestImport_ResolutionModes(t *testing.T) {
	t.Run("rename creates suffixed row", testResolutionRename)
	t.Run("overwrite preserves ID and updates frames", testResolutionOverwrite)
	t.Run("cancel returns 409 and leaves DB intact", testResolutionCancel)
	t.Run("garbage mode rejected with 400", testResolutionGarbage)
}

func testResolutionRename(t *testing.T) {
	srv, db := importTestServer(t)
	device := "device-rename"
	if _, err := SaveAnimation(context.Background(), db, device, "Wave", [][]Color{{{R: 1}}}); err != nil {
		t.Fatalf("seed: %v", err)
	}

	resp, body := postImport(t, srv, validImportRequestBody(device, "Wave"), "mode=rename")
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("status = %d, want 200; body=%s", resp.StatusCode, body)
	}
	var got api.ImportAnimationResponse
	if decErr := json.Unmarshal(body, &got); decErr != nil {
		t.Fatalf("decode: %v; body=%s", decErr, body)
	}
	if got.Animation.Name != "Wave (2)" {
		t.Errorf("Name = %q, want %q", got.Animation.Name, "Wave (2)")
	}
	if !got.RenamedFrom.IsSet() || got.RenamedFrom.Value != "Wave" {
		t.Errorf("RenamedFrom = %#v, want OptString{Wave}", got.RenamedFrom)
	}
	if n := countAnimations(t, db); n != 2 {
		t.Errorf("expected 2 rows, got %d", n)
	}
}

func testResolutionOverwrite(t *testing.T) {
	srv, db := importTestServer(t)
	device := "device-overwrite"
	original, seedErr := SaveAnimation(context.Background(), db, device, "Wave", [][]Color{{{R: 1}}})
	if seedErr != nil {
		t.Fatalf("seed: %v", seedErr)
	}

	resp, body := postImport(t, srv, validImportRequestBody(device, "Wave"), "mode=overwrite")
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("status = %d, want 200; body=%s", resp.StatusCode, body)
	}
	var got api.ImportAnimationResponse
	if decErr := json.Unmarshal(body, &got); decErr != nil {
		t.Fatalf("decode: %v; body=%s", decErr, body)
	}
	if got.Animation.ID != original.ID {
		t.Errorf("ID changed on overwrite: got %q, want %q", got.Animation.ID, original.ID)
	}
	if len(got.Animation.Frames) != 1 {
		t.Fatalf("expected 1 frame, got %d", len(got.Animation.Frames))
	}
	first := got.Animation.Frames[0]
	if len(first) != 100 {
		t.Fatalf("expected 100 pixels in frame, got %d", len(first))
	}
	if first[0].R != 255 {
		t.Errorf("expected pixel (0,0) red after overwrite, got %+v", first[0])
	}
	if n := countAnimations(t, db); n != 1 {
		t.Errorf("expected 1 row after overwrite, got %d", n)
	}
}

func testResolutionCancel(t *testing.T) {
	srv, db := importTestServer(t)
	device := "device-cancel"
	if _, err := SaveAnimation(context.Background(), db, device, "Wave", [][]Color{{{R: 1}}}); err != nil {
		t.Fatalf("seed: %v", err)
	}
	resp, _ := postImport(t, srv, validImportRequestBody(device, "Wave"), "mode=cancel")
	if resp.StatusCode != http.StatusConflict {
		t.Errorf("status = %d, want 409", resp.StatusCode)
	}
	if n := countAnimations(t, db); n != 1 {
		t.Errorf("expected 1 row after cancel, got %d", n)
	}
}

func testResolutionGarbage(t *testing.T) {
	srv, _ := importTestServer(t)
	resp, body := postImport(t, srv, validImportRequestBody("d1", "Junk"), "mode=garbage")
	if resp.StatusCode != http.StatusBadRequest {
		t.Errorf("status = %d, want 400; body=%s", resp.StatusCode, body)
	}
}

// --- r9: R-S2c-default-rename --------------------------------------------

// TestImport_DefaultModeIsRename omits the ?mode= query parameter; the
// service must default to rename, return a suffixed name, populate
// renamed_from, and leave the original row untouched.
func TestImport_DefaultModeIsRename(t *testing.T) {
	srv, db := importTestServer(t)
	device := "device-default"

	original, seedErr := SaveAnimation(context.Background(), db, device, "Wave", [][]Color{{{R: 1}}})
	if seedErr != nil {
		t.Fatalf("seed: %v", seedErr)
	}

	resp, body := postImport(t, srv, validImportRequestBody(device, "Wave"), "")
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("status = %d, want 200; body=%s", resp.StatusCode, body)
	}
	var got api.ImportAnimationResponse
	if decErr := json.Unmarshal(body, &got); decErr != nil {
		t.Fatalf("decode: %v; body=%s", decErr, body)
	}
	if got.Animation.Name == "Wave" {
		t.Errorf("expected suffixed name, got %q", got.Animation.Name)
	}
	if !strings.Contains(got.Animation.Name, "(") {
		t.Errorf("expected numeric suffix in name, got %q", got.Animation.Name)
	}
	if !got.RenamedFrom.IsSet() || got.RenamedFrom.Value != "Wave" {
		t.Errorf("RenamedFrom = %#v, want OptString{Wave}", got.RenamedFrom)
	}

	// Original must still exist with original frames.
	orig, err := GetAnimation(context.Background(), db, original.ID)
	if err != nil {
		t.Fatalf("GetAnimation original: %v", err)
	}
	if orig.Name != "Wave" {
		t.Errorf("original name changed: got %q", orig.Name)
	}
	if len(orig.Frames) != 1 || len(orig.Frames[0]) != 1 || orig.Frames[0][0].R != 1 {
		t.Errorf("original frames mutated: %+v", orig.Frames)
	}

	if n := countAnimations(t, db); n != 2 {
		t.Errorf("expected 2 rows, got %d", n)
	}
}
