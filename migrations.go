package main

import (
	"database/sql"
	"embed"
	"errors"
	"fmt"

	"github.com/golang-migrate/migrate/v4"
	"github.com/golang-migrate/migrate/v4/database/sqlite"
	"github.com/golang-migrate/migrate/v4/source/iofs"
)

// Embed migration files into the binary
//
//go:embed migrations/*.sql
var migrationsFS embed.FS

// RunMigrations executes all pending database migrations
// Uses embedded migration files
func RunMigrations(db *sql.DB) error {
	// Create migration source from embedded filesystem
	sourceDriver, err := iofs.New(migrationsFS, "migrations")
	if err != nil {
		return fmt.Errorf("failed to create migration source: %w", err)
	}

	// Wrap existing *sql.DB with migrate driver
	databaseDriver, err := sqlite.WithInstance(db, &sqlite.Config{})
	if err != nil {
		return fmt.Errorf("failed to create database driver: %w", err)
	}

	// Create migrate instance
	m, err := migrate.NewWithInstance(
		"iofs",
		sourceDriver,
		"sqlite",
		databaseDriver,
	)
	if err != nil {
		return fmt.Errorf("failed to create migrate instance: %w", err)
	}

	// Run all pending migrations
	if err := m.Up(); err != nil {
		// ErrNoChange is not an error - it means all migrations are already applied
		if errors.Is(err, migrate.ErrNoChange) {
			return nil
		}
		return fmt.Errorf("failed to run migrations: %w", err)
	}

	return nil
}
