package database

import (
	"database/sql"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strconv"
)

var migrationNamePattern = regexp.MustCompile(`^(\d+)_.*\.up\.sql$`)

type migrationFile struct {
	version int64
	name    string
	path    string
}

func RunMigrations(db *sql.DB, dir string) error {
	if err := ensureMigrationsTable(db); err != nil {
		return err
	}

	files, err := loadMigrationFiles(dir)
	if err != nil {
		return err
	}

	for _, mf := range files {
		applied, err := isMigrationApplied(db, mf.version)
		if err != nil {
			return err
		}
		if applied {
			continue
		}

		content, err := os.ReadFile(mf.path)
		if err != nil {
			return fmt.Errorf("read migration %s: %w", mf.name, err)
		}

		tx, err := db.Begin()
		if err != nil {
			return fmt.Errorf("start tx for migration %s: %w", mf.name, err)
		}

		if _, err := tx.Exec(string(content)); err != nil {
			_ = tx.Rollback()
			return fmt.Errorf("run migration %s: %w", mf.name, err)
		}

		if _, err := tx.Exec(
			`INSERT INTO schema_migrations (version, name) VALUES ($1, $2)`,
			mf.version,
			mf.name,
		); err != nil {
			_ = tx.Rollback()
			return fmt.Errorf("record migration %s: %w", mf.name, err)
		}

		if err := tx.Commit(); err != nil {
			return fmt.Errorf("commit migration %s: %w", mf.name, err)
		}
	}

	return nil
}

func ensureMigrationsTable(db *sql.DB) error {
	const q = `
CREATE TABLE IF NOT EXISTS schema_migrations (
	version BIGINT PRIMARY KEY,
	name TEXT NOT NULL,
	applied_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);`
	_, err := db.Exec(q)
	if err != nil {
		return fmt.Errorf("ensure schema_migrations table: %w", err)
	}
	return nil
}

func loadMigrationFiles(dir string) ([]migrationFile, error) {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil, fmt.Errorf("read migrations dir %s: %w", dir, err)
	}

	var files []migrationFile
	for _, e := range entries {
		if e.IsDir() {
			continue
		}

		matches := migrationNamePattern.FindStringSubmatch(e.Name())
		if len(matches) != 2 {
			continue
		}

		version, err := strconv.ParseInt(matches[1], 10, 64)
		if err != nil {
			return nil, fmt.Errorf("parse migration version from %s: %w", e.Name(), err)
		}

		files = append(files, migrationFile{
			version: version,
			name:    e.Name(),
			path:    filepath.Join(dir, e.Name()),
		})
	}

	sort.Slice(files, func(i, j int) bool {
		return files[i].version < files[j].version
	})

	return files, nil
}

func isMigrationApplied(db *sql.DB, version int64) (bool, error) {
	const q = `SELECT EXISTS(SELECT 1 FROM schema_migrations WHERE version = $1);`
	var exists bool
	if err := db.QueryRow(q, version).Scan(&exists); err != nil {
		return false, fmt.Errorf("check migration %d: %w", version, err)
	}
	return exists, nil
}
