// Package migrations menyematkan berkas *.sql direktori ini ke dalam binary
// agar migrasi otomatis terbawa saat runtime Docker (tanpa file eksternal).
package migrations

import "embed"

// FS berisi seluruh berkas migrasi SQL.
//
//go:embed *.sql
var FS embed.FS
