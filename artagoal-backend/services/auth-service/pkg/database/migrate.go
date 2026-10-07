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
// direktori migrations/) berurutan leksikal, masing-masing tepat sekali.
// Status pelacakan disimpan di tabel schema_migrations sehingga aman
// dijalankan ulang di setiap boot (file migrasi juga memakai IF NOT EXISTS).
//
// Contoh pemakaian di main:
//
//	//go:embed ../../migrations/*.sql
//	var migrationFS embed.FS
//	if err := database.RunMigrations(db, migrationFS, log.Default()); err != nil {
//		log.Fatalf("migrasi gagal: %v", err)
//	}
func RunMigrations(db *sql.DB, fsys fs.FS, logger *log.Logger) error {
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
		var already bool
		if err := db.QueryRow(
			`SELECT EXISTS(SELECT 1 FROM schema_migrations WHERE filename = $1)`, name,
		).Scan(&already); err != nil {
			return fmt.Errorf("migrate: check %s: %w", name, err)
		}
		if already {
			continue
		}

		content, err := fs.ReadFile(fsys, name)
		if err != nil {
			return fmt.Errorf("migrate: read %s: %w", name, err)
		}

		tx, err := db.Begin()
		if err != nil {
			return fmt.Errorf("migrate: begin %s: %w", name, err)
		}
		if _, err := tx.Exec(string(content)); err != nil {
			_ = tx.Rollback()
			return fmt.Errorf("migrate: apply %s: %w", name, err)
		}
		if _, err := tx.Exec(
			`INSERT INTO schema_migrations (filename) VALUES ($1)`, name,
		); err != nil {
			_ = tx.Rollback()
			return fmt.Errorf("migrate: track %s: %w", name, err)
		}
		if err := tx.Commit(); err != nil {
			return fmt.Errorf("migrate: commit %s: %w", name, err)
		}
		logger.Printf("migrate: applied %s", name)
	}

	return nil
}
