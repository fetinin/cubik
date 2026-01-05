package main

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	_ "modernc.org/sqlite"
)

// InitDB initializes SQLite database connection with proper configuration
// Returns *sql.DB instance and error
func InitDB(ctx context.Context) (*sql.DB, error) {
	// Open database with URI and query parameters
	// cache=shared: allows multiple connections to share cache
	// mode=rwc: read-write-create mode
	db, err := sql.Open("sqlite", "file:cubik.db?cache=shared&mode=rwc")
	if err != nil {
		return nil, fmt.Errorf("failed to open database: %w", err)
	}

	// Configure connection pool
	// SQLite performs best with limited connections due to single-writer design
	db.SetMaxOpenConns(25)                  // Reasonable for embedded DB
	db.SetMaxIdleConns(5)                   // Balance memory vs connection overhead
	db.SetConnMaxLifetime(time.Minute * 5) // Periodic refresh

	// Verify connection
	if err := db.PingContext(ctx); err != nil {
		db.Close()
		return nil, fmt.Errorf("failed to ping database: %w", err)
	}

	// Enable Write-Ahead Logging for better concurrency
	if _, err := db.ExecContext(ctx, "PRAGMA journal_mode = WAL"); err != nil {
		db.Close()
		return nil, fmt.Errorf("failed to enable WAL mode: %w", err)
	}

	// Enable foreign key constraints
	if _, err := db.ExecContext(ctx, "PRAGMA foreign_keys = ON"); err != nil {
		db.Close()
		return nil, fmt.Errorf("failed to enable foreign keys: %w", err)
	}

	// Set busy timeout for lock retries (5 seconds)
	if _, err := db.ExecContext(ctx, "PRAGMA busy_timeout = 5000"); err != nil {
		db.Close()
		return nil, fmt.Errorf("failed to set busy timeout: %w", err)
	}

	return db, nil
}

// CloseDB gracefully closes database connection
func CloseDB(db *sql.DB) error {
	if db == nil {
		return nil
	}
	if err := db.Close(); err != nil {
		return fmt.Errorf("failed to close database: %w", err)
	}
	return nil
}
