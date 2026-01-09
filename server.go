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

func corsMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type")

		if r.Method == "OPTIONS" {
			w.WriteHeader(http.StatusOK)
			return
		}

		next.ServeHTTP(w, r)
	})
}

func StartServer(ctx context.Context, db *sql.DB, port string) error {
	handler := &APIHandler{db: db}
	srv, err := api.NewServer(handler)
	if err != nil {
		return fmt.Errorf("failed to create server: %w", err)
	}

	mux := http.NewServeMux()
	mux.Handle("/api/", corsMiddleware(srv))

	frontendSubFS, err := fs.Sub(frontendFS, "front/build")
	if err != nil {
		return fmt.Errorf("failed to create frontend sub-filesystem: %w", err)
	}

	spaHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		path := strings.TrimPrefix(r.URL.Path, "/")
		if path == "" {
			path = "index.html"
		}

		file, err := frontendSubFS.Open(path)
		if err != nil {
			http.NotFound(w, r)
			return
		}
		defer file.Close()

		stat, err := file.Stat()
		if err != nil {
			http.Error(w, "Error reading file", http.StatusInternalServerError)
			return
		}

		http.ServeContent(w, r, path, stat.ModTime(), file.(io.ReadSeeker))
	})
	mux.Handle("/", spaHandler)

	httpServer := &http.Server{
		Addr:    ":" + port,
		Handler: mux,
	}

	log.Printf("Starting Cubik server on http://localhost:%s", port)

	go func() {
		<-ctx.Done()
		log.Println("Shutting down server...")
		httpServer.Shutdown(context.Background())
	}()

	if err := httpServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		return fmt.Errorf("server error: %w", err)
	}
	return nil
}
