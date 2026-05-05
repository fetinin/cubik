package main

import (
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

// exportTestServer wires the same middleware chain StartServer mounts at
// /api/ against an in-memory DB. Mirroring importTestServer keeps both
// endpoint integration tests speaking the same dialect.
func exportTestServer(t *testing.T) (*httptest.Server, *sql.DB) {
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

func getExport(t *testing.T, srv *httptest.Server, id string) (*http.Response, []byte) {
	t.Helper()
	resp, err := http.Get(srv.URL + "/api/animation/" + id + "/export")
	if err != nil {
		t.Fatalf("GET failed: %v", err)
	}
	body, readErr := io.ReadAll(resp.Body)
	resp.Body.Close()
	if readErr != nil {
		t.Fatalf("read response body failed: %v", readErr)
	}
	return resp, body
}

// blankFrame returns a 20x5 all-black frame buffer that the encoder will
// accept (the codec rejects frames whose length doesn't equal width*height).
func blankFrame() []Color {
	return make([]Color, MatrixWidth*MatrixHeight)
}

// litFrame returns a 20x5 frame with a couple of non-black pixels at known
// positions; the export must preserve them as Sparse pixels in the response.
func litFrame() []Color {
	frame := blankFrame()
	// (0,0) red.
	frame[0] = Color{R: 255}
	// (19,4) blue — last cell, exercises the row-major flattening path.
	frame[MatrixWidth*MatrixHeight-1] = Color{B: 255}
	return frame
}

// --- r1: R-export-file (happy path) ---------------------------------------

// TestExport_HappyPath_ReturnsSparseAnimationWithHeader seeds an animation,
// hits GET /api/animation/{id}/export, and asserts the full contract: 200
// status, Content-Type JSON, Content-Disposition attachment with the
// animation's name, and a body whose decoded SparseAnimation matches the
// seed (version, name, dimensions, frame count, and at least one non-black
// pixel surviving end-to-end).
func TestExport_HappyPath_ReturnsSparseAnimationWithHeader(t *testing.T) {
	srv, db := exportTestServer(t)
	device := "0xdevice-export-1"
	name := "Rainbow Wave"

	seeded, seedErr := SaveAnimation(
		context.Background(),
		db,
		device,
		name,
		[][]Color{litFrame(), blankFrame()},
	)
	if seedErr != nil {
		t.Fatalf("seed failed: %v", seedErr)
	}

	resp, body := getExport(t, srv, seeded.ID)

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("status = %d, want 200; body=%s", resp.StatusCode, body)
	}
	if ct := resp.Header.Get("Content-Type"); !strings.Contains(ct, "application/json") {
		t.Errorf("Content-Type = %q, want application/json...", ct)
	}

	cd := resp.Header.Get("Content-Disposition")
	wantCD := `attachment; filename="Rainbow Wave.cubik.json"`
	if cd != wantCD {
		t.Errorf("Content-Disposition = %q, want %q", cd, wantCD)
	}

	var got api.SparseAnimation
	if err := json.Unmarshal(body, &got); err != nil {
		t.Fatalf("decode response: %v; body=%s", err, body)
	}
	if got.Version != "1.0" {
		t.Errorf("Version = %q, want %q", got.Version, "1.0")
	}
	if got.Name != name {
		t.Errorf("Name = %q, want %q", got.Name, name)
	}
	if got.Width != int32(MatrixWidth) || got.Height != int32(MatrixHeight) {
		t.Errorf("Dimensions = %dx%d, want %dx%d",
			got.Width, got.Height, MatrixWidth, MatrixHeight)
	}
	if len(got.Frames) != 2 {
		t.Fatalf("Frames length = %d, want 2", len(got.Frames))
	}

	// First frame had two lit pixels; the all-black second frame should be
	// emitted as an empty sparse array.
	if len(got.Frames[0]) != 2 {
		t.Errorf("Frames[0] sparse pixel count = %d, want 2", len(got.Frames[0]))
	}
	if len(got.Frames[1]) != 0 {
		t.Errorf("Frames[1] (all-black) sparse pixel count = %d, want 0", len(got.Frames[1]))
	}

	// At least one non-black pixel must survive — test r1 acceptance.
	foundLit := false
	for _, px := range got.Frames[0] {
		if px.C != 0 {
			foundLit = true
			break
		}
	}
	if !foundLit {
		t.Errorf("expected at least one non-black pixel in Frames[0], got %+v", got.Frames[0])
	}
}

// TestExport_RoundTripPixelEqualsSeed verifies the codec's encode-side
// invariant on the wire: lit pixels at (0,0)=red and (19,4)=blue from the
// seed must reappear in the response with matching coordinates and packed
// colour values. Catches any drift between the codec and the export route.
func TestExport_RoundTripPixelEqualsSeed(t *testing.T) {
	srv, db := exportTestServer(t)
	seeded, seedErr := SaveAnimation(
		context.Background(),
		db,
		"0xdevice-roundtrip",
		"Round Trip",
		[][]Color{litFrame()},
	)
	if seedErr != nil {
		t.Fatalf("seed failed: %v", seedErr)
	}

	resp, body := getExport(t, srv, seeded.ID)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("status = %d, want 200; body=%s", resp.StatusCode, body)
	}
	var got api.SparseAnimation
	if err := json.Unmarshal(body, &got); err != nil {
		t.Fatalf("decode response: %v; body=%s", err, body)
	}

	// Index sparse pixels by (x,y) for assertion.
	type key struct{ X, Y int32 }
	seen := make(map[key]int32, len(got.Frames[0]))
	for _, px := range got.Frames[0] {
		seen[key{px.X, px.Y}] = px.C
	}
	if c := seen[key{0, 0}]; c != 0xFF0000 {
		t.Errorf("pixel (0,0) c = 0x%x, want 0xFF0000 (red)", c)
	}
	if c := seen[key{19, 4}]; c != 0x0000FF {
		t.Errorf("pixel (19,4) c = 0x%x, want 0x0000FF (blue)", c)
	}
}

// --- r1: R-export-file (404 branch) ---------------------------------------

// TestExport_MissingID_Returns404 hits an export URL whose ID does not exist
// in the DB. Asserts 404 + an Error-shaped JSON body. The id slot uses a
// well-formed UUID so route matching succeeds and we exercise the
// not-found branch (not a 400 path-validation error).
func TestExport_MissingID_Returns404(t *testing.T) {
	srv, _ := exportTestServer(t)

	resp, body := getExport(t, srv, "11111111-2222-3333-4444-555555555555")
	if resp.StatusCode != http.StatusNotFound {
		t.Fatalf("status = %d, want 404; body=%s", resp.StatusCode, body)
	}
	var ge api.Error
	if err := json.Unmarshal(body, &ge); err != nil {
		t.Fatalf("decode error response: %v; body=%s", err, body)
	}
	if !strings.Contains(strings.ToLower(ge.Error), "not found") {
		t.Errorf("Error = %q, expected to mention 'not found'", ge.Error)
	}
}

// --- Filename sanitization ------------------------------------------------

// TestExport_FilenameSanitization seeds animations with hostile names and
// asserts the Content-Disposition filename has no path separators or
// control bytes and always ends with .cubik.json.
func TestExport_FilenameSanitization(t *testing.T) {
	cases := []struct {
		seedName     string
		wantFilename string
	}{
		{seedName: "../../etc/passwd", wantFilename: "_._etc_passwd.cubik.json"},
		{seedName: "Bad\x00Name\x1f", wantFilename: "Bad_Name_.cubik.json"},
		{seedName: "with/slash\\and:colon*?", wantFilename: "with_slash_and_colon__.cubik.json"},
		{seedName: "....hidden", wantFilename: "hidden.cubik.json"},
		{seedName: "   ", wantFilename: "animation.cubik.json"},
	}

	for _, tc := range cases {
		t.Run(tc.seedName, func(t *testing.T) {
			runFilenameSanitizationCase(t, tc.seedName, tc.wantFilename)
		})
	}
}

// runFilenameSanitizationCase seeds one animation, exports it, and asserts
// the Content-Disposition filename matches the expected sanitized form.
// Extracted from the table loop to keep TestExport_FilenameSanitization's
// cognitive complexity below the gocognit threshold.
func runFilenameSanitizationCase(t *testing.T, seedName, wantFilename string) {
	t.Helper()
	srv, db := exportTestServer(t)
	seeded, seedErr := SaveAnimation(
		context.Background(), db, "0xdev-sanitize", seedName,
		[][]Color{blankFrame()},
	)
	if seedErr != nil {
		t.Fatalf("seed failed: %v", seedErr)
	}

	resp, body := getExport(t, srv, seeded.ID)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("status = %d, want 200; body=%s", resp.StatusCode, body)
	}

	cd := resp.Header.Get("Content-Disposition")
	wantCD := `attachment; filename="` + wantFilename + `"`
	if cd != wantCD {
		t.Errorf("Content-Disposition = %q, want %q", cd, wantCD)
	}

	assertFilenameSafe(t, wantFilename)
}

// assertFilenameSafe checks the table-supplied filename has no path
// separators, no control bytes, and ends with .cubik.json. Tripping any of
// these means the test table itself is wrong.
func assertFilenameSafe(t *testing.T, filename string) {
	t.Helper()
	if strings.ContainsAny(filename, `/\`) {
		t.Errorf("wantFilename %q contains path separator — test bug", filename)
	}
	for _, r := range filename {
		if r < 0x20 || r == 0x7f {
			t.Errorf("filename %q contains control byte 0x%x", filename, r)
		}
	}
	if !strings.HasSuffix(filename, ".cubik.json") {
		t.Errorf("filename %q missing .cubik.json suffix", filename)
	}
}
