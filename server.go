package main

import (
	"context"
	"database/sql"
	"fmt"
	"log"
	"net/http"

	"cubik/api"
)

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

// StartServer initializes and runs the HTTP API server on port 8080
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

	// Configure HTTP server with CORS middleware
	httpServer := &http.Server{
		Addr:    ":8080",
		Handler: corsMiddleware(srv),
	}

	log.Printf("Starting Cubik API server on http://localhost:8080")
	log.Printf("Note: For device discovery to work, ensure 'LAN Control' is enabled in the Yeelight app")

	go func() {
		<-ctx.Done()
		log.Println("Shutting down server...")
		httpServer.Shutdown(context.Background())
	}()
	// Start server (blocks until error or shutdown)
	if err := httpServer.ListenAndServe(); err != nil {
		return fmt.Errorf("server error: %w", err)
	}

	return nil
}
