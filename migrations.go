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

//go:embed migrations/*.sql
var migrationsFS embed.FS

func RunMigrations(db *sql.DB) error {
	sourceDriver, sourceErr := iofs.New(migrationsFS, "migrations")
	if sourceErr != nil {
		return fmt.Errorf("failed to create migration source: %w", sourceErr)
	}

	databaseDriver, dbErr := sqlite.WithInstance(db, &sqlite.Config{})
	if dbErr != nil {
		return fmt.Errorf("failed to create database driver: %w", dbErr)
	}

	m, migrateErr := migrate.NewWithInstance("iofs", sourceDriver, "sqlite", databaseDriver)
	if migrateErr != nil {
		return fmt.Errorf("failed to create migrate instance: %w", migrateErr)
	}

	if err := m.Up(); err != nil && !errors.Is(err, migrate.ErrNoChange) {
		return fmt.Errorf("failed to run migrations: %w", err)
	}

	return nil
}
