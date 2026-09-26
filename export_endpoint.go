package main

import (
	"cubik/api"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"regexp"
	"strings"

	"github.com/go-faster/jx"
)

// exportFilenameSuffix is the file extension stamped onto every exported
// animation. Mirrors the frontend's FILE_SUFFIX in fileChannel.ts so files
// produced by either side are interchangeable.
const exportFilenameSuffix = ".cubik.json"

// exportFilenameDefault is the fallback stem used when the animation name
// sanitises down to the empty string. Mirrors the frontend's DEFAULT_NAME.
const exportFilenameDefault = "animation"

// exportPathPattern matches GET /api/animation/{id}/export. We pin both ends
// (^...$) so other animation routes (e.g. /api/animation/{id}, the import
// endpoint, or any future sub-route) cannot accidentally divert into the
// export handler.
var exportPathPattern = regexp.MustCompile(`^/api/animation/([^/]+)/export$`)

// exportRouteOverride wraps the ogen-served HTTP handler. If the incoming
// request is GET /api/animation/{id}/export, the wrapper serves it directly
// — fetching the animation, encoding via the codec, and writing JSON with a
// Content-Disposition: attachment header. All other requests fall through to
// the wrapped handler unchanged. Mounted in server.go between the size-cap
// middleware and the ogen server.
func exportRouteOverride(db *sql.DB, next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			next.ServeHTTP(w, r)
			return
		}
		match := exportPathPattern.FindStringSubmatch(r.URL.Path)
		if match == nil {
			next.ServeHTTP(w, r)
			return
		}
		serveExport(w, r, db, match[1])
	})
}

// serveExport is the actual export handler: fetch, encode, set header, write.
// On a not-found id it writes a 404 in the same shape ogen would (matches the
// Error schema); on encode failure it writes a 500 in the same shape.
func serveExport(w http.ResponseWriter, r *http.Request, db *sql.DB, id string) {
	saved, err := GetAnimation(r.Context(), db, id)
	if errors.Is(err, ErrNotFound) {
		writeExportError(w, http.StatusNotFound, msgAnimationNotFound)
		return
	}
	if err != nil {
		slog.Error("export fetch failed", "id", id, "error", err)
		writeExportError(w, http.StatusInternalServerError,
			fmt.Sprintf("failed to fetch animation: %v", err))
		return
	}

	payload, encErr := EncodeAnimation(saved, MatrixWidth, MatrixHeight)
	if encErr != nil {
		slog.Error("export encode failed", "id", id, "error", encErr)
		writeExportError(w, http.StatusInternalServerError,
			fmt.Sprintf("failed to encode animation: %v", encErr))
		return
	}

	filename := buildExportFilename(saved.Name)
	// Set headers BEFORE WriteHeader; once status is written the header map
	// is committed and further mutations are no-ops.
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.Header().Set("Content-Disposition", fmt.Sprintf(`attachment; filename=%q`, filename))
	w.WriteHeader(http.StatusOK)

	// Use the same JSON encoder ogen uses on the success path so the body
	// shape and field-ordering match exactly what a SparseAnimation would
	// look like when served via ogen — keeps the wire response consistent
	// across endpoints and lets generated clients parse it identically.
	enc := new(jx.Encoder)
	payload.Encode(enc)
	if _, writeErr := enc.WriteTo(w); writeErr != nil {
		slog.Error("export response write failed", "id", id, "error", writeErr)
	}
}

// writeExportError emits an Error-shaped JSON body. Kept independent of
// ogen's encoders so the non-ogen route doesn't need an ogen response object.
// The shape (`{"error": "..."}`) matches the Error schema from spec.yml.
func writeExportError(w http.ResponseWriter, status int, msg string) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)
	body, _ := json.Marshal(api.Error{Error: msg})
	_, _ = w.Write(body)
}

// buildExportFilename produces a downloadable file name from a user-provided
// animation name. Mirrors the rules in front/src/lib/io/fileChannel.ts so a
// file downloaded via the browser matches one downloaded via direct curl:
//   - replace path separators, control bytes, and Windows-reserved chars with `_`
//   - collapse runs of two-or-more dots into one (kills `..` traversal patterns)
//   - strip leading dots (no `.hidden` filenames)
//   - fall back to the default stem when the result is empty
//   - append `.cubik.json` unless the stem already ends with it
func buildExportFilename(name string) string {
	stem := sanitizeExportFilename(name)
	if strings.HasSuffix(strings.ToLower(stem), exportFilenameSuffix) {
		return stem
	}
	return stem + exportFilenameSuffix
}

var (
	// forbidden = control bytes 0x00-0x1F, DEL (0x7F), and the cross-platform
	// reserved set: / \ : * ? " < > |.
	forbiddenFilenameChars = regexp.MustCompile(`[\x00-\x1f\x7f/\\:*?"<>|]`)
	// twoOrMoreDots collapses sequences like "..", "..." into a single ".".
	twoOrMoreDots = regexp.MustCompile(`\.{2,}`)
	// leadingDots strips any number of leading "." characters.
	leadingDots = regexp.MustCompile(`^\.+`)
)

func sanitizeExportFilename(name string) string {
	trimmed := strings.TrimSpace(name)
	cleaned := forbiddenFilenameChars.ReplaceAllString(trimmed, "_")
	dotsCollapsed := twoOrMoreDots.ReplaceAllString(cleaned, ".")
	withoutLeadingDots := leadingDots.ReplaceAllString(dotsCollapsed, "")
	stem := strings.TrimSpace(withoutLeadingDots)
	if stem == "" {
		return exportFilenameDefault
	}
	return stem
}
