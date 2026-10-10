package database

import (
	"database/sql"
	"time"

	_ "github.com/lib/pq"
)

// NewPostgresConnection membuka koneksi database PostgreSQL.
// dsn contoh: "postgres://user:password@localhost:5432/artagoal?sslmode=disable"
// Pool disetel agresif terhadap idle agar koneksi dibuang SEBELUM Supabase
// sempat memutusnya di latar belakang (anti stale connection / cold start):
// idle >1 menit didaur ulang, umur maksimum 3 menit, maks 10 koneksi terbuka.
func NewPostgresConnection(dsn string) (*sql.DB, error) {
	db, err := sql.Open("postgres", dsn)
	if err != nil {
		return nil, err
	}
	db.SetMaxOpenConns(10)
	db.SetMaxIdleConns(5)
	db.SetConnMaxLifetime(3 * time.Minute)
	db.SetConnMaxIdleTime(1 * time.Minute)
	if err := db.Ping(); err != nil {
		return nil, err
	}
	return db, nil
}
