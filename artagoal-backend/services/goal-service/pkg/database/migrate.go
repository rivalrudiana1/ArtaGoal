package database

import (
	"database/sql"
	"fmt"
	"io/fs"
	"log"
	"sort"
	"strings"
)

// RunMigrations menerapkan file .sql dari fsys (hasil go:embed atas
// direktori migrations/) berurutan leksikal, masing-masing tepat sekali
// per service. Status pelacakan disimpan di tabel schema_migrations
// (kunci service/filename) sehingga aman dijalankan ulang di setiap boot
// (file migrasi juga memakai IF NOT EXISTS).
//
// Service diikutkan dalam kunci karena auth-service & goal-service berbagi
// satu database fisik: tanpa namespace, "001_init.sql" milik satu service
// akan menandai migrasi service lain sebagai sudah diterapkan.
func RunMigrations(db *sql.DB, service string, fsys fs.FS, logger *log.Logger) error {
	if logger == nil {
		logger = log.Default()
	}

	if _, err := db.Exec(`CREATE TABLE IF NOT EXISTS schema_migrations (
		filename TEXT PRIMARY KEY,
		applied_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
	)`); err != nil {
		return fmt.Errorf("migrate: schema_migrations: %w", err)
	}

	entries, err := fs.ReadDir(fsys, ".")
	if err != nil {
		return fmt.Errorf("migrate: read dir: %w", err)
	}
	names := make([]string, 0, len(entries))
	for _, e := range entries {
		if e.IsDir() || !strings.HasSuffix(e.Name(), ".sql") {
			continue
		}
		names = append(names, e.Name())
	}
	sort.Strings(names)

	for _, name := range names {
		key := service + "/" + name
		var already bool
		if err := db.QueryRow(
			`SELECT EXISTS(SELECT 1 FROM schema_migrations WHERE filename = $1)`, key,
		).Scan(&already); err != nil {
			return fmt.Errorf("migrate: check %s: %w", key, err)
		}
		if already {
			continue
		}

		content, err := fs.ReadFile(fsys, name)
		if err != nil {
			return fmt.Errorf("migrate: read %s: %w", key, err)
		}

		tx, err := db.Begin()
		if err != nil {
			return fmt.Errorf("migrate: begin %s: %w", key, err)
		}
		if _, err := tx.Exec(string(content)); err != nil {
			_ = tx.Rollback()
			return fmt.Errorf("migrate: apply %s: %w", key, err)
		}
		if _, err := tx.Exec(
			`INSERT INTO schema_migrations (filename) VALUES ($1)`, key,
		); err != nil {
			_ = tx.Rollback()
			return fmt.Errorf("migrate: track %s: %w", key, err)
		}
		if err := tx.Commit(); err != nil {
			return fmt.Errorf("migrate: commit %s: %w", key, err)
		}
		logger.Printf("migrate: applied %s", key)
	}

	return nil
}
