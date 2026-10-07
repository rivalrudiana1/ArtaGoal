package postgres

import (
	"context"
	"database/sql"
	"errors"
	"fmt"

	"github.com/lib/pq"

	"artagoal/auth-service/internal/domain/entity"
	"artagoal/auth-service/internal/repository"
)

// UserPostgresRepository adalah implementasi repository.UserRepository
// memakai database/sql + driver github.com/lib/pq.
type UserPostgresRepository struct {
	db *sql.DB
}

// Pastikan kontrak terpenuhi saat compile.
var _ repository.UserRepository = (*UserPostgresRepository)(nil)

// NewUserPostgresRepository membuat instance baru.
func NewUserPostgresRepository(db *sql.DB) *UserPostgresRepository {
	return &UserPostgresRepository{db: db}
}

// scanner adalah abstraksi atas *sql.Row dan *sql.Rows agar logika
// scan terpusat di satu tempat.
type scanner interface {
	Scan(dest ...any) error
}

// scanUser memetakan satu baris users ke entity.User.
// avatar_url dibaca NullString agar aman bila berisi NULL (baris lama).
func scanUser(s scanner) (entity.User, error) {
	var u entity.User
	var avatar sql.NullString
	if err := s.Scan(
		&u.ID,
		&u.Name,
		&u.Email,
		&u.PasswordHash,
		&avatar,
		&u.CreatedAt,
		&u.UpdatedAt,
	); err != nil {
		return entity.User{}, err
	}
	if avatar.Valid {
		u.AvatarURL = avatar.String
	}
	return u, nil
}

const userColumns = `id, name, email, password_hash, avatar_url, created_at, updated_at`

// mapError memetakan error driver ke sentinel domain.
func mapError(op, key string, err error) error {
	if errors.Is(err, sql.ErrNoRows) {
		return fmt.Errorf("postgres: %s %s: %w", op, key, entity.ErrUserNotFound)
	}
	var pqErr *pq.Error
	if errors.As(err, &pqErr) && pqErr.Code == "23505" {
		return fmt.Errorf("postgres: %s %s: %w", op, key, entity.ErrEmailAlreadyExists)
	}
	return fmt.Errorf("postgres: %s %s: %w", op, key, err)
}

// CreateUser menyimpan user baru. Kolom id diserahkan ke default
// database (gen_random_uuid()), lalu dikembalikan via RETURNING.
// Pelanggaran unique email dipetakan ke ErrEmailAlreadyExists.
func (r *UserPostgresRepository) CreateUser(ctx context.Context, user entity.User) (entity.User, error) {
	const query = `
		INSERT INTO users (name, email, password_hash)
		VALUES ($1, $2, $3)
		RETURNING id, created_at, updated_at`

	if err := r.db.QueryRowContext(ctx, query,
		user.Name,
		user.Email,
		user.PasswordHash,
	).Scan(&user.ID, &user.CreatedAt, &user.UpdatedAt); err != nil {
		return entity.User{}, mapError("CreateUser", user.Email, err)
	}
	return user, nil
}

// GetUserByEmail mencari user berdasarkan email (sudah dinormalisasi lowercase).
func (r *UserPostgresRepository) GetUserByEmail(ctx context.Context, email string) (entity.User, error) {
	query := `SELECT ` + userColumns + ` FROM users WHERE email = $1`

	user, err := scanUser(r.db.QueryRowContext(ctx, query, email))
	if err != nil {
		return entity.User{}, mapError("GetUserByEmail", email, err)
	}
	return user, nil
}

// GetUserByID mencari user berdasarkan id.
func (r *UserPostgresRepository) GetUserByID(ctx context.Context, id string) (entity.User, error) {
	query := `SELECT ` + userColumns + ` FROM users WHERE id = $1`

	user, err := scanUser(r.db.QueryRowContext(ctx, query, id))
	if err != nil {
		return entity.User{}, mapError("GetUserByID", id, err)
	}
	return user, nil
}

// UpdateProfile memperbarui nama dan avatar user, lalu mengembalikan
// entity terbaru. Mengembalikan ErrUserNotFound bila id tidak ada.
func (r *UserPostgresRepository) UpdateProfile(ctx context.Context, id, name, avatarURL string) (entity.User, error) {
	const query = `UPDATE users SET name = $1, avatar_url = $2 WHERE id = $3`

	res, err := r.db.ExecContext(ctx, query, name, avatarURL, id)
	if err != nil {
		return entity.User{}, mapError("UpdateProfile", id, err)
	}
	affected, err := res.RowsAffected()
	if err != nil {
		return entity.User{}, fmt.Errorf("postgres: UpdateProfile %s rows affected: %w", id, err)
	}
	if affected == 0 {
		return entity.User{}, fmt.Errorf("postgres: UpdateProfile %s: %w", id, entity.ErrUserNotFound)
	}

	updated, err := r.GetUserByID(ctx, id)
	if err != nil {
		return entity.User{}, fmt.Errorf("postgres: UpdateProfile reload %s: %w", id, err)
	}
	return updated, nil
}

// UpdatePassword mengganti hash password user.
// Mengembalikan ErrUserNotFound bila id tidak ada.
func (r *UserPostgresRepository) UpdatePassword(ctx context.Context, id, passwordHash string) error {
	const query = `UPDATE users SET password_hash = $1, updated_at = NOW() WHERE id = $2`

	res, err := r.db.ExecContext(ctx, query, passwordHash, id)
	if err != nil {
		return mapError("UpdatePassword", id, err)
	}
	affected, err := res.RowsAffected()
	if err != nil {
		return fmt.Errorf("postgres: UpdatePassword %s rows affected: %w", id, err)
	}
	if affected == 0 {
		return fmt.Errorf("postgres: UpdatePassword %s: %w", id, entity.ErrUserNotFound)
	}
	return nil
}
