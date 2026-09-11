package store

import (
	"database/sql"
	_ "embed"
	"fmt"
	"strings"

	_ "modernc.org/sqlite"
)

//go:embed migrations/001_initial.sql
var migration001 string

//go:embed migrations/002_dependencies.sql
var migration002 string

// DB wraps sql.DB and provides migration support
type DB struct {
	*sql.DB
}

// Open opens a SQLite database and applies migrations
func Open(dbPath string) (*DB, error) {
	db, err := sql.Open("sqlite", dbPath)
	if err != nil {
		return nil, fmt.Errorf("open database: %w", err)
	}

	// Enable WAL mode for better concurrency
	if _, err := db.Exec("PRAGMA journal_mode=WAL"); err != nil {
		db.Close()
		return nil, fmt.Errorf("enable WAL: %w", err)
	}

	// Enable foreign keys
	if _, err := db.Exec("PRAGMA foreign_keys=ON"); err != nil {
		db.Close()
		return nil, fmt.Errorf("enable foreign keys: %w", err)
	}

	wrappedDB := &DB{db}

	if err := wrappedDB.migrate(); err != nil {
		db.Close()
		return nil, fmt.Errorf("migrate: %w", err)
	}

	return wrappedDB, nil
}

// OpenTest opens an in-memory SQLite database for testing
func OpenTest() (*DB, error) {
	return Open(":memory:")
}

// migrate applies all migrations in order
func (db *DB) migrate() error {
	migrations := []string{
		migration001,
		migration002,
	}

	for i, migration := range migrations {
		if _, err := db.Exec(migration); err != nil {
			return fmt.Errorf("migration %d: %w", i+1, err)
		}
	}

	return nil
}

// Transaction executes a function within a database transaction
func (db *DB) Transaction(fn func(tx *sql.Tx) error) error {
	tx, err := db.Begin()
	if err != nil {
		return fmt.Errorf("begin transaction: %w", err)
	}
	defer tx.Rollback()

	if err := fn(tx); err != nil {
		return err
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("commit transaction: %w", err)
	}

	return nil
}

// ErrNotFound is returned when a record is not found
var ErrNotFound = fmt.Errorf("record not found")

// ErrInvalidTransition is returned when an invalid lifecycle transition is attempted
var ErrInvalidTransition = fmt.Errorf("invalid lifecycle transition")

// ErrClaimFailed is returned when a task claim fails (already claimed or not in proposed state)
var ErrClaimFailed = fmt.Errorf("claim failed")

// ErrSameProject is returned when a dependency crosses project boundaries
var ErrSameProject = fmt.Errorf("dependency must be within the same project")

// ErrLifecycleBypass is returned when a generic update tries to mutate lifecycle fields
var ErrLifecycleBypass = fmt.Errorf("cannot mutate lifecycle fields through generic update")

// ErrReleaseFailed is returned when a release fails: task not in active state or not assigned to caller
var ErrReleaseFailed = fmt.Errorf("release failed: task not in active state or not assigned to caller")

// joinSQL joins SQL strings with newlines
func joinSQL(parts ...string) string {
	return strings.Join(parts, "\n")
}
