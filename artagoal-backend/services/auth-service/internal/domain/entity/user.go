package entity

import (
	"errors"
	"time"
)

// Sentinel errors domain auth.
var (
	ErrEmailAlreadyExists = errors.New("email already registered")
	ErrInvalidCredentials = errors.New("invalid email or password")
	ErrUserNotFound       = errors.New("user not found")
)

// User memetakan tabel users di PostgreSQL.
// PasswordHash tidak pernah diserialisasi ke JSON.
type User struct {
	ID           string    `json:"id"`
	Name         string    `json:"name"`
	Email        string    `json:"email"`
	PasswordHash string    `json:"-"`
	CreatedAt    time.Time `json:"created_at"`
	UpdatedAt    time.Time `json:"updated_at"`
}

// RegisterRequest adalah payload POST /api/v1/auth/register.
type RegisterRequest struct {
	Name     string `json:"name"`
	Email    string `json:"email"`
	Password string `json:"password"`
}

// LoginRequest adalah payload POST /api/v1/auth/login.
type LoginRequest struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

// UserInfo adalah representasi publik user (tanpa password hash).
type UserInfo struct {
	ID    string `json:"id"`
	Name  string `json:"name"`
	Email string `json:"email"`
}

// AuthResponse dikembalikan setiap operasi auth yang sukses.
type AuthResponse struct {
	Token string   `json:"token"`
	User  UserInfo `json:"user"`
}

// ToInfo memetakan User ke representasi publiknya.
func (u User) ToInfo() UserInfo {
	return UserInfo{ID: u.ID, Name: u.Name, Email: u.Email}
}
