package main

import (
	"context"
	"database/sql"
	"embed"
	"fmt"
	"io"
	"io/fs"
	"log"
	"net/http"
	"strings"

	"cubik/api"
)

//go:embed front/build/**
var frontendFS embed.FS

// corsMiddleware adds CORS headers for frontend access
func corsMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Allow all origins for development
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type")

		// Handle preflight requests
		if r.Method == "OPTIONS" {
			w.WriteHeader(http.StatusOK)
			return
		}

		next.ServeHTTP(w, r)
	})
}

// StartServer initializes and runs the HTTP API server on port 9080
func StartServer(ctx context.Context, db *sql.DB) error {
	// Create handler implementation with DB dependency
	handler := &APIHandler{
		db: db,
	}

	// Create ogen server with handler
	srv, err := api.NewServer(handler)
	if err != nil {
		return fmt.Errorf("failed to create server: %w", err)
	}

	// Create a mux to handle both API and static files
	mux := http.NewServeMux()

	// API routes under /api/* with CORS middleware
	mux.Handle("/api/", corsMiddleware(srv))

	// Serve static frontend files
	frontendSubFS, err := fs.Sub(frontendFS, "front/build")
	if err != nil {
		return fmt.Errorf("failed to create frontend sub-filesystem: %w", err)
	}

	// Create a custom file server with SPA fallback
	spaHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Prepare the path for file system lookup
		path := strings.TrimPrefix(r.URL.Path, "/")
		if path == "" {
			path = "index.html"
		}

		// Try to open the file
		file, err := frontendSubFS.Open(path)
		if err != nil {
			http.NotFound(w, r)
			return
		}
		defer file.Close()

		// File exists, serve it with proper content type
		stat, err := file.Stat()
		if err != nil {
			http.Error(w, "Error reading file", http.StatusInternalServerError)
			return
		}

		// Let http.ServeContent handle content type detection
		http.ServeContent(w, r, path, stat.ModTime(), file.(io.ReadSeeker))
	})
	
	mux.Handle("/", spaHandler)

	// Configure HTTP server
	httpServer := &http.Server{
		Addr:    ":9080",
		Handler: mux,
	}

	log.Printf("Starting Cubik server on http://localhost:9080")
	log.Printf("Serving frontend SPA and API endpoints")
	log.Printf("Note: For device discovery to work, ensure 'LAN Control' is enabled in the Yeelight app")

	go func() {
		<-ctx.Done()
		log.Println("Shutting down server...")
		httpServer.Shutdown(context.Background())
	}()
	// Start server (blocks until error or shutdown)
	if err := httpServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		return fmt.Errorf("server error: %w", err)
	}

	return nil
}
