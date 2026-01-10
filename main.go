package main

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"os/signal"
	"sync"
	"syscall"
)

func main() {
	ctx, _ := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	if err := run(ctx); err != nil {
		slog.Error("Application error", "error", err)
		os.Exit(1)
	}
}

func run(ctx context.Context) error {
	cfg, err := LoadConfig()
	if err != nil {
		return fmt.Errorf("failed to load configuration: %w", err)
	}

	db, err := InitDB(ctx, cfg.ServerDBPath)
	if err != nil {
		return fmt.Errorf("failed to initialize database: %w", err)
	}
	defer func() {
		if closeErr := CloseDB(db); closeErr != nil {
			slog.Error("Failed to close database", "error", closeErr)
		}
	}()

	if migrationErr := RunMigrations(db); migrationErr != nil {
		return fmt.Errorf("failed to run migrations: %w", migrationErr)
	}

	var wg sync.WaitGroup
	wg.Go(func() {
		if serverErr := StartServer(ctx, db, cfg.ServerPort); serverErr != nil {
			slog.Error("Server error", "error", serverErr)
			os.Exit(1)
		}
	})

	<-ctx.Done()
	slog.Info("Shutting down...")
	wg.Wait()

	return nil
}
