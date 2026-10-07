package repository

import (
	"context"

	"artagoal/auth-service/internal/domain/entity"
)

// UserRepository adalah kontrak persistensi untuk aggregate User.
// Implementasi konkret (mis. PostgreSQL) tinggal di subpaket postgres.
type UserRepository interface {
	CreateUser(ctx context.Context, user entity.User) (entity.User, error)
	GetUserByEmail(ctx context.Context, email string) (entity.User, error)
	GetUserByID(ctx context.Context, id string) (entity.User, error)
	// UpdateProfile memperbarui nama dan avatar user.
	// avatarURL adalah path statis (mis. "/uploads/avatars/abc.jpg").
	UpdateProfile(ctx context.Context, id, name, avatarURL string) (entity.User, error)
}

