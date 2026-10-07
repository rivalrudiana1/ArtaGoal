package entity

import (
	"errors"
	"time"
)

// ErrNotificationNotFound menandai notifikasi tidak ada / bukan milik peminta.
var ErrNotificationNotFound = errors.New("notification not found")

// Notification memetakan tabel notifications di PostgreSQL.
// Dibuat oleh reminder worker (pengingat tenggat) dan dibaca via API.
type Notification struct {
	ID        string    `json:"id"`
	UserID    string    `json:"user_id"`
	GoalID    *string   `json:"goal_id,omitempty"`
	Title     string    `json:"title"`
	Message   string    `json:"message"`
	IsRead    bool      `json:"is_read"`
	CreatedAt time.Time `json:"created_at"`
}
