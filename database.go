package main

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	_ "modernc.org/sqlite"
)

func InitDB(ctx context.Context, dbPath string) (*sql.DB, error) {
	dbURI := fmt.Sprintf("file:%s?cache=shared&mode=rwc", dbPath)
	db, err := sql.Open("sqlite", dbURI)
	if err != nil {
		return nil, fmt.Errorf("failed to open database: %w", err)
	}

	db.SetMaxOpenConns(25)
	db.SetMaxIdleConns(5)
	db.SetConnMaxLifetime(5 * time.Minute)

	if pingErr := db.PingContext(ctx); pingErr != nil {
		db.Close()
		return nil, fmt.Errorf("failed to ping database: %w", pingErr)
	}

	pragmas := []string{
		"PRAGMA journal_mode = WAL",
		"PRAGMA foreign_keys = ON",
		"PRAGMA busy_timeout = 5000",
	}
	for _, pragma := range pragmas {
		if _, execErr := db.ExecContext(ctx, pragma); execErr != nil {
			db.Close()
			return nil, fmt.Errorf("failed to execute %s: %w", pragma, execErr)
		}
	}

	return db, nil
}

func CloseDB(db *sql.DB) error {
	if db == nil {
		return nil
	}
	if err := db.Close(); err != nil {
		return fmt.Errorf("failed to close database: %w", err)
	}
	return nil
}
