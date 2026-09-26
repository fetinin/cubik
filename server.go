package main

import (
	"context"
	"cubik/api"
	"database/sql"
	"embed"
	"encoding/json"
	"fmt"
	"io"
	"io/fs"
	"log/slog"
	"net/http"
	"strings"
)

//go:embed front/build/**
var frontendFS embed.FS

// importBodyLimitMiddleware enforces the import-payload size cap on the
// import endpoint specifically. Other API routes do not need a hard cap (they
// take small JSON bodies) so we only short-circuit oversize requests here.
//
// Two-stage check:
//  1. Content-Length header (cheap, deterministic). Some clients omit it; we
//     still want to reject before ogen consumes any bytes.
//  2. [http.MaxBytesReader] on the body (handles chunked/missing-Content-Length
//     uploads). When the cap is hit during decode, ogen surfaces the read
//     error and the configured ErrorHandler maps it to an ogen-internal 5xx;
//     the Content-Length pre-check above keeps us out of that path for any
//     well-behaved client. Streaming-without-Content-Length uploads above the
//     cap will get the 5xx ogen response — acceptable for a path nobody hits
//     by accident.
func importBodyLimitMiddleware(next http.Handler) http.Handler {
	const importPath = "/api/animation/import"
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == importPath {
			if r.ContentLength > ImportPayloadMaxBytes {
				writeImportTooLarge(w)
				return
			}
			if r.Body != nil {
				r.Body = http.MaxBytesReader(w, r.Body, ImportPayloadMaxBytes)
			}
		}
		next.ServeHTTP(w, r)
	})
}

// writeImportTooLarge emits the 413 response shape declared in spec.yml for
// the import operation. Body shape matches `Error` (the ogen-encoded variant
// for ImportAnimationRequestEntityTooLarge).
func writeImportTooLarge(w http.ResponseWriter) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(http.StatusRequestEntityTooLarge)
	body, _ := json.Marshal(map[string]string{
		"error": fmt.Sprintf("request body exceeds size cap of %d bytes", ImportPayloadMaxBytes),
	})
	_, _ = w.Write(body)
}

func corsMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type")

		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusOK)
			return
		}

		next.ServeHTTP(w, r)
	})
}

// buildAPIHandler composes the API middleware chain (size cap + CORS) around
// the ogen server, without mounting the SPA. Extracted so tests can drive
// the wire path with [httptest.NewServer].
//
// The export endpoint is special-cased via exportRouteOverride: ogen's
// typed-handler signature does not expose [http.ResponseWriter], so the
// Content-Disposition: attachment header — which is part of the export
// contract — is set by a tiny non-ogen route mounted directly on the chain.
// All other operations flow through the ogen server unchanged.
func buildAPIHandler(db *sql.DB) (http.Handler, error) {
	handler := &APIHandler{db: db}
	srv, srvErr := api.NewServer(handler)
	if srvErr != nil {
		return nil, fmt.Errorf("failed to create server: %w", srvErr)
	}
	return importBodyLimitMiddleware(corsMiddleware(exportRouteOverride(db, srv))), nil
}

func StartServer(ctx context.Context, db *sql.DB, port string) error {
	apiHandler, buildErr := buildAPIHandler(db)
	if buildErr != nil {
		return buildErr
	}

	mux := http.NewServeMux()
	mux.Handle("/api/", apiHandler)

	frontendSubFS, subErr := fs.Sub(frontendFS, "front/build")
	if subErr != nil {
		return fmt.Errorf("failed to create frontend sub-filesystem: %w", subErr)
	}

	spaHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		path := strings.TrimPrefix(r.URL.Path, "/")
		if path == "" {
			path = "index.html"
		}

		file, openErr := frontendSubFS.Open(path)
		if openErr != nil {
			http.NotFound(w, r)
			return
		}
		defer file.Close()

		stat, statErr := file.Stat()
		if statErr != nil {
			http.Error(w, "Error reading file", http.StatusInternalServerError)
			return
		}

		rs, ok := file.(io.ReadSeeker)
		if !ok {
			http.Error(w, "Error reading file", http.StatusInternalServerError)
			return
		}
		http.ServeContent(w, r, path, stat.ModTime(), rs)
	})
	mux.Handle("/", spaHandler)

	httpServer := &http.Server{
		Addr:    ":" + port,
		Handler: mux,
	}

	slog.Info("Starting Cubik server", "address", "http://localhost:"+port)

	go func() {
		<-ctx.Done()
		slog.Info("Shutting down server...")
		if shutdownErr := httpServer.Shutdown(context.Background()); shutdownErr != nil {
			slog.Error("Server shutdown error", "error", shutdownErr)
		}
	}()

	if listenErr := httpServer.ListenAndServe(); listenErr != nil && listenErr != http.ErrServerClosed {
		return fmt.Errorf("server error: %w", listenErr)
	}
	return nil
}
