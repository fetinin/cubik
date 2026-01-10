package main

import (
	"context"
	"cubik/api"
	"database/sql"
	"embed"
	"fmt"
	"io"
	"io/fs"
	"log/slog"
	"net/http"
	"strings"
)

//go:embed front/build/**
var frontendFS embed.FS

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

func StartServer(ctx context.Context, db *sql.DB, port string) error {
	handler := &APIHandler{db: db}
	srv, srvErr := api.NewServer(handler)
	if srvErr != nil {
		return fmt.Errorf("failed to create server: %w", srvErr)
	}

	mux := http.NewServeMux()
	mux.Handle("/api/", corsMiddleware(srv))

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
