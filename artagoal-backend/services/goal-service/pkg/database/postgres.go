package database

import (
	"database/sql"
	"time"

	_ "github.com/lib/pq"
)

// NewPostgresConnection membuka koneksi database PostgreSQL.
// dsn contoh: "postgres://user:password@localhost:5432/artagoal?sslmode=disable"
// Pool disetel defensif agar tahan cold start: koneksi basi didaur ulang
// berkala dan ledakan konkurensi dibatasi, bukan menumpuk tanpa batas.
func NewPostgresConnection(dsn string) (*sql.DB, error) {
	db, err := sql.Open("postgres", dsn)
	if err != nil {
		return nil, err
	}
	db.SetMaxOpenConns(25)
	db.SetMaxIdleConns(5)
	db.SetConnMaxLifetime(5 * time.Minute)
	if err := db.Ping(); err != nil {
		return nil, err
	}
	return db, nil
}
