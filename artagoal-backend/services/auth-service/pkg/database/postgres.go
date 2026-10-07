package database

import (
	"database/sql"

	_ "github.com/lib/pq"
)

// NewPostgresConnection membuka koneksi database PostgreSQL.
// dsn contoh: "postgres://user:password@localhost:5432/artagoal?sslmode=disable"
func NewPostgresConnection(dsn string) (*sql.DB, error) {
	db, err := sql.Open("postgres", dsn)
	if err != nil {
		return nil, err
	}
	if err := db.Ping(); err != nil {
		return nil, err
	}
	return db, nil
}
